package bot

import (
	"context"
	"fmt"
	"time"

	"gopkg.in/telebot.v4"
)

// adminPanelHandler sends the admin-specific keyboard.
func (b *Bot) adminPanelHandler(ctx telebot.Context) error {
	userID := ctx.Sender().ID
	b.log.Info("Admin user accessed the admin panel", "user", userID)

	adminMenu.Reply(
		adminMenu.Row(btnBroadcast),
		adminMenu.Row(btnBack),
	)

	return ctx.Send(
		"You are king and god in this realm. Do as you please.\nDo you wish to issue a decree to the mortals, or simply revel in your power?",
		adminMenu,
	)
}

// broadcastInitiateHandler starts the broadcast process.
func (b *Bot) broadcastInitiateHandler(ctx telebot.Context) error {
	userID := ctx.Sender().ID
	b.log.Info("Admin user initiated a broadcast", "user", userID)

	// 1. Set the user's state to expect a broadcast message
	b.stateManager.Set(userID, UserState{
		WaitingFor: stateAwaitingBroadcast,
	})

	// 2. Ask the admin to send the message
	return ctx.Send("Please send the message you want to broadcast to all users.")
}

// broadcastMessageHandler confirms the broadcast and starts the sending process.
func (b *Bot) broadcastMessageHandler(ctx context.Context, bCtx telebot.Context, message string) error {
	adminID := bCtx.Sender().ID

	// 1. Get a list of all users from the database.
	users, err := b.usrepo.GetAllTgUserIDs(ctx)
	if err != nil {
		b.log.ErrorContext(ctx, "Failed to get users for broadcast", "error", err)
		return bCtx.Send(ErrInternal)
	}

	// 2. Start the broadcast in a goroutine so the bot doesn't freeze.
	go b.sendBroadcast(ctx, adminID, message, users)

	// 3. Immediately confirm to the admin that the process has started.
	numReceivers := len(users) - 1
	responseText := fmt.Sprintf(
		"✅ Broadcast started. Your message will be sent to %d users.",
		numReceivers,
	)
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
	reportText := fmt.Sprintf(
		"🏁 Broadcast finished!\n\nSuccessfully sent: %d\nFailed to send: %d",
		successfulSends,
		failedSends,
	)
	if _, err = b.bot.Send(telebot.ChatID(adminID), reportText); err != nil {
		b.log.WarnContext(ctx, "Failed to send result message to admin", "admin", adminID, "error", err)
	}
}
