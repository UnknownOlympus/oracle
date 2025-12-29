package bot

import (
	"context"
	"fmt"
	"time"

	"gopkg.in/telebot.v4"
)

const timeout = 5

// broadcastInitiateHandler starts the broadcast process.
func (b *Bot) broadcastInitiateHandler(ctx telebot.Context) error {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	userID := ctx.Sender().ID
	b.log.Info("Admin user initiated a broadcast", "user", userID)

	// 1. Set the user's state to expect a broadcast message
	b.stateManager.Set(userID, UserState{
		WaitingFor: stateAwaitingBroadcast,
	})

	// 2. Ask the admin to send the message
	return ctx.Send(b.t(timeoutCtx, ctx, "admin.broadcast.prompt"))
}

// broadcastMessageHandler confirms the broadcast and starts the sending process.
func (b *Bot) broadcastMessageHandler(ctx context.Context, bCtx telebot.Context, message string) error {
	adminID := bCtx.Sender().ID

	// 1. Get a list of all users from the database.
	users, err := b.usrepo.GetAllTgUserIDs(ctx)
	if err != nil {
		b.log.ErrorContext(ctx, "Failed to get users for broadcast", "error", err)
		return bCtx.Send(b.t(ctx, bCtx, "error.internal"))
	}

	// 2. Start the broadcast in a goroutine so the bot doesn't freeze.
	go b.sendBroadcast(ctx, adminID, message, users)

	// 3. Immediately confirm to the admin that the process has started.
	numReceivers := len(users) - 1
	responseText := b.tWithData(ctx, bCtx, "admin.broadcast.started", map[string]interface{}{
		"count": numReceivers,
	})
	return bCtx.Send(responseText)
}

// sendBroadcast is the background worker that sends the messages.
func (b *Bot) sendBroadcast(ctx context.Context, adminID int64, message string, userIDs []int64) {
	b.log.InfoContext(ctx, "Starting broadcast", "from_admin", adminID, "user_count", len(userIDs)-1)

	admin, err := b.tarepo.GetEmployee(ctx, adminID)
	if err != nil {
		b.log.WarnContext(ctx, "Failed to get employee data about admin", "user", adminID, "error", err)
	}

	successfulSends := 0
	failedSends := 0

	for _, userID := range userIDs {
		// Don't send the message to the admin who initiated it
		if userID == adminID {
			continue
		}

		// Send the message to one user
		formattedMessage := fmt.Sprintf("*You received a message from %s:*\n\n%s", admin.ShortName, message)
		_, err = b.bot.Send(telebot.ChatID(userID), formattedMessage, telebot.ModeMarkdown)
		if err != nil {
			// This can happen if a user has blocked the bot
			b.log.WarnContext(ctx, "Failed to send broadcast message to user", "user", userID, "error", err)
			failedSends++
		} else {
			successfulSends++
		}

		// IMPORTANT: Wait a bit between messages to avoid Telegram's rate limits
		const telegramRateTimeout = 100 * time.Millisecond
		time.Sleep(telegramRateTimeout)
	}

	// Send a final report back to the admin
	// Create a temporary telebot.Context for translation
	reportText := b.tWithData(ctx, nil, "admin.broadcast.finished", map[string]interface{}{
		"success": successfulSends,
		"failed":  failedSends,
	})
	if _, err = b.bot.Send(telebot.ChatID(adminID), reportText); err != nil {
		b.log.WarnContext(ctx, "Failed to send result message to admin", "admin", adminID, "error", err)
	}
}

// geocodingIssuesHandler displays tasks with geocoding problems for debugging.
func (b *Bot) geocodingIssuesHandler(ctx telebot.Context) error {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), timeout*time.Second)
	defer cancel()

	userID := ctx.Sender().ID
	b.log.Info("Admin requested geocoding issues view", "user", userID)

	// Fetch geocoding issues from database
	issues, err := b.tarepo.GetGeocodingIssues(timeoutCtx)
	if err != nil {
		b.log.ErrorContext(timeoutCtx, "Failed to get geocoding issues", "error", err)
		return ctx.Send(b.t(timeoutCtx, ctx, "error.internal"))
	}

	// If no issues found, inform the admin
	if len(issues) == 0 {
		return ctx.Send(b.t(timeoutCtx, ctx, "admin.geocoding.no_issues"))
	}

	// Format the response as a structured table
	// Limit to prevent Telegram message size limits (max 4096 chars)
	maxIssues := 20
	if len(issues) > maxIssues {
		issues = issues[:maxIssues]
	}

	// Build formatted response with header
	responseText := b.tWithData(timeoutCtx, ctx, "admin.geocoding.issues_header", map[string]interface{}{
		"total": len(issues),
	})
	responseText += "\n\n"

	// Format each issue as a structured entry
	for idx, issue := range issues {
		// Truncate address if too long
		address := issue.Address
		const maxAddrLen = 40
		if len(address) > maxAddrLen {
			address = address[:maxAddrLen] + "..."
		}

		// Truncate error message if too long
		errorMsg := issue.GeocodingError
		if errorMsg == "" {
			errorMsg = b.t(timeoutCtx, ctx, "admin.geocoding.no_error_yet")
		}
		const maxErrorLen = 50
		if len(errorMsg) > maxErrorLen {
			errorMsg = errorMsg[:maxErrorLen] + "..."
		}

		// Format entry with task ID, attempts, address, and error
		entryText := b.tWithData(timeoutCtx, ctx, "admin.geocoding.issue_entry", map[string]interface{}{
			"num":      idx + 1,
			"id":       issue.TaskID,
			"attempts": issue.GeocodingAttempts,
			"address":  address,
			"error":    errorMsg,
		})
		responseText += entryText + "\n"
	}

	// Add footer note if results were truncated
	if len(issues) == maxIssues {
		responseText += "\n" + b.t(timeoutCtx, ctx, "admin.geocoding.issues_truncated")
	}

	return ctx.Send(responseText, telebot.ModeMarkdown)
}

// geocodingResetHandler resets geocoding errors with confirmation.
func (b *Bot) geocodingResetHandler(ctx telebot.Context) error {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), timeout*time.Second)
	defer cancel()

	userID := ctx.Sender().ID
	b.log.Info("Admin requested geocoding errors reset", "user", userID)

	// Create confirmation inline keyboard
	confirmMenu := &telebot.ReplyMarkup{}
	btnConfirm := confirmMenu.Data(
		b.t(timeoutCtx, ctx, "admin.geocoding.reset.confirm"),
		"geocoding_reset_confirm",
		"confirm",
	)
	btnCancel := confirmMenu.Data(
		b.t(timeoutCtx, ctx, "admin.geocoding.reset.cancel"),
		"geocoding_reset_cancel",
	)
	confirmMenu.Inline(confirmMenu.Row(btnConfirm, btnCancel))

	// Send confirmation prompt
	promptText := b.t(timeoutCtx, ctx, "admin.geocoding.reset.prompt")
	return ctx.Send(promptText, confirmMenu, telebot.ModeMarkdown)
}

// geocodingResetConfirmHandler executes the geocoding reset after confirmation.
func (b *Bot) geocodingResetConfirmHandler(ctx telebot.Context) error {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), timeout*time.Second)
	defer cancel()

	userID := ctx.Sender().ID
	b.log.Info("Admin confirmed geocoding errors reset", "user", userID)

	// Execute the reset
	rowsAffected, err := b.tarepo.ResetGeocodingErrors(timeoutCtx)
	if err != nil {
		b.log.ErrorContext(timeoutCtx, "Failed to reset geocoding errors", "error", err)
		return ctx.Edit(b.t(timeoutCtx, ctx, "error.internal"))
	}

	// Send success message with count
	responseText := b.tWithData(timeoutCtx, ctx, "admin.geocoding.reset.success", map[string]interface{}{
		"count": rowsAffected,
	})
	b.log.Info("Geocoding errors reset successfully", "rows_affected", rowsAffected, "admin", userID)

	return ctx.Edit(responseText, telebot.ModeMarkdown)
}

// geocodingResetCancelHandler handles the cancel action for geocoding reset.
func (b *Bot) geocodingResetCancelHandler(ctx telebot.Context) error {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), timeout*time.Second)
	defer cancel()

	b.log.Info("Admin canceled geocoding errors reset", "user", ctx.Sender().ID)
	return ctx.Edit(b.t(timeoutCtx, ctx, "admin.geocoding.reset.canceled"), telebot.ModeMarkdown)
}
