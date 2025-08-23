package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/UnknownOlympus/oracle/internal/models"
	"github.com/UnknownOlympus/oracle/internal/report"
	"gopkg.in/telebot.v4"
)

// logoutHandler handles the logout process for a user. It removes the user's state from
// the userStates map, logs the logout action, and attempts to delete the user from the
// repository. If the deletion is successful, it sends a success message; otherwise, it
// informs the user of a failure. The operation is performed with a timeout of 3 seconds.
func (b *Bot) logoutHandler(ctx telebot.Context) error {
	userID := ctx.Sender().ID
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	delete(userStates, userID)
	b.log.Info("User logged out", "user", userID)
	b.metrics.CommandReceived.WithLabelValues("logout").Inc()

	startTime := time.Now()
	err := b.repo.DeleteUserByID(timeoutCtx, userID)
	b.metrics.DBQueryDuration.WithLabelValues("delete_user").Observe(time.Since(startTime).Seconds())
	if err != nil {
		b.metrics.SentMessages.WithLabelValues("error").Inc()
		return ctx.Send("💩 Failed to logout, please try later")
	}

	b.metrics.SentMessages.WithLabelValues("text").Inc()
	return ctx.Send("😢 Logout was successfull", mainMenu)
}

// infoHandler handles the request for user information. It logs the request, retrieves the employee data
// from the repository using the user's ID, and sends a formatted response containing the user's name,
// position, email, and phone number. In case of an error while fetching the employee data, it logs the
// error and sends an internal error message to the user.
func (b *Bot) infoHandler(ctx telebot.Context) error {
	b.log.Info("User requested info", "user", ctx.Sender().ID)
	b.metrics.CommandReceived.WithLabelValues("info").Inc()

	userID := ctx.Sender().ID
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cacheKey := fmt.Sprintf("oracle:info:user:%d", userID)
	const cacheTTL = 12 * time.Hour

	cachedUserJSON, err := b.redisClient.Get(timeoutCtx, cacheKey).Result()
	if err == nil {
		b.log.Info("Info found in cache", "user", userID, "key", cacheKey)
		b.metrics.CacheOps.WithLabelValues("get", "hit").Inc()
		var user models.Employee
		if json.Unmarshal([]byte(cachedUserJSON), &user) == nil {
			responseText := formatUserInfo(user) // Use a helper to format the text
			b.metrics.SentMessages.WithLabelValues("text_cached").Inc()
			return ctx.Send(responseText, telebot.ModeMarkdown)
		}
	}

	b.metrics.CacheOps.WithLabelValues("get", "miss").Inc()
	b.log.Info("User info not in cache, fetching from DB", "user", userID)
	startTime := time.Now()
	user, err := b.repo.GetEmployee(timeoutCtx, userID)
	b.metrics.DBQueryDuration.WithLabelValues("get_employee").Observe(time.Since(startTime).Seconds())
	if err != nil {
		b.log.Error("Failed to get employee data", "error", err)
		b.metrics.SentMessages.WithLabelValues("error").Inc()
		return ctx.Send(ErrInternal)
	}

	userJSON, err := json.Marshal(user)
	if err != nil {
		b.metrics.CacheOps.WithLabelValues("set", "error").Inc()
		b.log.Error("Failed to marshal user for caching", "error", err, "user", userID)
	} else {
		err = b.redisClient.Set(timeoutCtx, cacheKey, userJSON, cacheTTL).Err()
		if err != nil {
			b.metrics.CacheOps.WithLabelValues("set", "error").Inc()
			b.log.Error("Failed to save user to cache", "error", err, "user", userID)
		}
		b.metrics.CacheOps.WithLabelValues("set", "success").Inc()
	}

	b.metrics.SentMessages.WithLabelValues("text").Inc()
	responseText := formatUserInfo(user)

	return ctx.Send(responseText, telebot.ModeMarkdown)
}

// formatUserInfo its a helper function to keep the code DRY.
func formatUserInfo(user models.Employee) string {
	return fmt.Sprintf(`
🤦‍♂️ *These mortals again…*

*Name:* %s
*Position:* %s
*Email:* %s
*Phone:* %s

💬 Okay, I saved this somewhere… or not.
`,
		user.FullName, user.Position, user.Email, user.Phone)
}

// formatTaskDetails is a helper function for taskDetailsHandler.
func formatTaskDetails(details *models.TaskDetails) string {
	messageText := fmt.Sprintf(
		"*Task details #%d*\n\n"+
			"*Type:* %s\n"+
			"*Created:* %s",
		details.ID,
		details.Type,
		details.CreationDate.Format("02.01.2006"),
	)
	if len(details.CustomerNames) > 0 {
		messageText += fmt.Sprintf("\n*Client Name:* %s", strings.Join(details.CustomerNames, ", "))
	}
	suffixText := fmt.Sprintf(
		"\n*Address:* %s\n"+
			"*Description:* %s\n"+
			"*Assigned to:* %s",
		details.Address,
		details.Description,
		strings.Join(details.Executors, ", "),
	)
	messageText += suffixText
	if len(details.Comments) > 0 {
		messageText += fmt.Sprintf("\n*Comments:*\n- %s", strings.Join(details.Comments, ";\n- "))
	}

	if details.Latitude.Valid && details.Longitude.Valid {
		mapURL := fmt.Sprintf("https://maps.google.com/?q=%f,%f", details.Latitude.Float64, details.Longitude.Float64)
		messageText += fmt.Sprintf("\n\n[📍 Open on map](%s)", mapURL)
	} else {
		messageText += "\n\n📍 *Location not added yet*"
	}

	return messageText
}

// activeTasksHandler handles the request for active tasks from the user.
// It retrieves the active tasks assigned to the user and sends a response
// with the list of tasks. If there are no active tasks, it informs the user.
// In case of an error while fetching tasks, it sends an internal error message.
// The function also creates a dynamic inline keyboard for task selection.
func (b *Bot) activeTasksHandler(ctx telebot.Context) error {
	userID := ctx.Sender().ID
	b.log.Info("User requested active tasks", "user", userID)
	b.metrics.CommandReceived.WithLabelValues("active_tasks").Inc()

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	startTime := time.Now()
	tasks, err := b.repo.GetActiveTasksByExecutor(timeoutCtx, userID)
	b.metrics.DBQueryDuration.WithLabelValues("get_active_tasks").Observe(time.Since(startTime).Seconds())
	if err != nil {
		b.log.Error("Failed to get active tasks", "error", err, "user", userID)
		b.metrics.SentMessages.WithLabelValues("error").Inc()
		return ctx.Send(ErrInternal)
	}

	if len(tasks) == 0 {
		b.metrics.SentMessages.WithLabelValues("text").Inc()
		return ctx.Send("🎉 You have no active tasks!")
	}

	// creates dynamic inline keyboard
	var rows [][]telebot.InlineButton
	buttons := make([]telebot.InlineButton, 0, 3)

	for idx, task := range tasks {
		btn := telebot.InlineButton{
			Unique: "task_details",
			Text:   fmt.Sprintf("#%d", task.ID),
			Data:   strconv.Itoa(task.ID),
		}
		buttons = append(buttons, btn)
		if (idx+1)%3 == 0 || idx == len(tasks)-1 {
			rows = append(rows, buttons)
			buttons = nil
		}
	}

	b.metrics.SentMessages.WithLabelValues("text").Inc()
	menu := &telebot.ReplyMarkup{InlineKeyboard: rows}
	return ctx.Send("Here is a list of your active tasks:", menu)
}

// taskDetailsHandler handles the request for task details based on the task ID provided in the callback context.
// It retrieves the task details from the repository and formats them into a message.
// If the task ID is invalid or if there is an error retrieving the details, it logs the error and responds accordingly.
// The function also edits the original message with the formatted task details.
//
// Parameters:
//   - ctx: The telebot context containing the callback data and user information.
//
// Returns:
//   - error: Returns an error if there is an issue processing the request or editing the message.
func (b *Bot) taskDetailsHandler(ctx telebot.Context) error {
	b.metrics.CommandReceived.WithLabelValues("task_details").Inc()
	taskID, err := strconv.Atoi(ctx.Data())
	if err != nil {
		b.log.Error("Invalid task ID in callback", "error", err, "data", ctx.Data())
		b.metrics.SentMessages.WithLabelValues("error").Inc()
		if err = ctx.Respond(); err != nil {
			b.log.Error("Failed to send respond to callback", "error", err)
		}
	}

	userID := ctx.Sender().ID
	b.log.Info("User requested task details", "user", userID, "taskID", taskID)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cacheKey := fmt.Sprintf("oracle:task_details:%d", taskID)
	const cacheTTL = 5 * time.Minute

	cachedTaskJSON, err := b.redisClient.Get(timeoutCtx, cacheKey).Result()
	if err == nil {
		b.log.Info("Task found in cache", "task", taskID, "key", cacheKey)
		b.metrics.CacheOps.WithLabelValues("get", "hit").Inc()
		var details models.TaskDetails
		if json.Unmarshal([]byte(cachedTaskJSON), &details) == nil {
			responseText := formatTaskDetails(&details)
			b.metrics.SentMessages.WithLabelValues("edit").Inc()
			err = ctx.Edit(responseText, telebot.ModeMarkdown, ctx.Message().ReplyMarkup)
			if err != nil && !errors.Is(err, telebot.ErrSameMessageContent) {
				b.log.Error("Failed to edit message", "error", err)
			}

			return nil
		}
	}

	b.metrics.CacheOps.WithLabelValues("get", "miss").Inc()
	b.log.Info("Task details not in cache, fetching from DB", "task", taskID)

	startTime := time.Now()
	details, err := b.repo.GetTaskDetailsByID(timeoutCtx, taskID)
	b.metrics.DBQueryDuration.WithLabelValues("get_active_tasks").Observe(time.Since(startTime).Seconds())
	if err != nil {
		b.log.Error("Failed to get task details", "error", err, "taskID", taskID)
		b.metrics.SentMessages.WithLabelValues("error").Inc()
		return ctx.Respond(&telebot.CallbackResponse{Text: "Error retrieving data."})
	}

	taskJSON, err := json.Marshal(details)
	if err != nil {
		b.metrics.CacheOps.WithLabelValues("set", "error").Inc()
		b.log.Error("Failed to marshal task details for caching", "error", err, "task", taskID)
	} else {
		err = b.redisClient.Set(timeoutCtx, cacheKey, taskJSON, cacheTTL).Err()
		if err != nil {
			b.metrics.CacheOps.WithLabelValues("set", "error").Inc()
			b.log.Error("Failed to save task details to cache", "error", err, "task", taskID)
		}
		b.metrics.CacheOps.WithLabelValues("set", "success").Inc()
	}

	b.metrics.SentMessages.WithLabelValues("edit").Inc()
	messageText := formatTaskDetails(details)
	err = ctx.Edit(messageText, telebot.ModeMarkdown, ctx.Message().ReplyMarkup)
	if err != nil && !errors.Is(err, telebot.ErrSameMessageContent) {
		b.log.Error("Failed to edit message", "error", err)
	}

	return nil
}

// reportHandler handles the report request from the user. It presents the user with
// a menu to choose the reporting period, which includes options for the current month,
// the last month, and the last 7 days. It sends a message prompting the user to select
// their desired reporting period along with the corresponding inline keyboard menu.
func (b *Bot) reportHandler(ctx telebot.Context) error {
	menu := &telebot.ReplyMarkup{}
	menu.Inline(
		menu.Row(menu.Data("⌛ For the current month", "report_period_current_month")),
		menu.Row(menu.Data("⏳ For the last month", "report_period_last_month")),
		menu.Row(menu.Data("⏰ For the last 7 days", "report_period_last_7_days")),
	)

	b.metrics.SentMessages.WithLabelValues("text").Inc()
	return ctx.Send("🐷 Choose how many days you want the report for", menu)
}

// generatorReportHandler handles the generation of reports based on the user's request.
// It responds to the user with a message indicating that the report is being generated,
// determines the time period for the report based on the callback unique identifier,
// generates the report in Excel format, and sends the report back to the user.
//
// Supported time periods:
// - Current month
// - Last month
// - Last 7 days
//
// If the report generation fails or there are no completed tasks for the selected period,
// an appropriate error message is sent to the user.
func (b *Bot) generatorReportHandler(ctx telebot.Context) error {
	b.metrics.CommandReceived.WithLabelValues("report").Inc()
	b.metrics.SentMessages.WithLabelValues("respond").Inc()
	_ = ctx.Respond(&telebot.CallbackResponse{Text: "🔧 One moment, generating your report..."})

	userID := ctx.Sender().ID
	b.log.Info("User requested report", "user", userID, "data", ctx.Callback().Unique)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	from, to, periodMetric, err := b.parseReportPeriod(ctx)
	if err != nil {
		b.metrics.SentMessages.WithLabelValues("error").Inc()
		return ctx.Edit("💩 Unsupported time period", ctx.Message().ReplyMarkup)
	}

	cacheKey := fmt.Sprintf("oracle:report:user:%d:period:%s", userID, periodMetric)
	if sent, _ := b.sendCachedReportIfExists(timeoutCtx, ctx, userID, cacheKey, from, to); sent {
		return nil
	}

	return b.generateAndSendReport(timeoutCtx, ctx, userID, from, to, periodMetric, cacheKey)
}

func (b *Bot) parseReportPeriod(ctx telebot.Context) (time.Time, time.Time, string, error) {
	now := time.Now()
	switch ctx.Callback().Unique {
	case "report_period_current_month":
		from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		return from, from.AddDate(0, 1, 0).Add(-time.Nanosecond), "current_1m", nil
	case "report_period_last_month":
		from := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location())
		return from, from.AddDate(0, 1, 0).Add(-time.Nanosecond), "last_1m", nil
	case "report_period_last_7_days":
		return now.AddDate(0, 0, -7), now, "last_7d", nil
	default:
		return time.Time{}, time.Time{}, "", errors.New("unsupported period")
	}
}

func (b *Bot) sendCachedReportIfExists(
	ctx context.Context,
	tbCtx telebot.Context,
	userID int64,
	cacheKey string,
	from, to time.Time,
) (bool, error) {
	cachedReport, err := b.redisClient.Get(ctx, cacheKey).Bytes()
	if err != nil {
		b.metrics.CacheOps.WithLabelValues("get", "miss").Inc()
		return false, fmt.Errorf("failed to get report from cache: %w", err)
	}

	b.metrics.CacheOps.WithLabelValues("get", "hit").Inc()
	b.log.InfoContext(ctx, "Report found in cache", "user", userID, "key", cacheKey)

	responseText := fmt.Sprintf(
		"💩 Your report for the period %s to %s is ready.\nJust pass it on to Tanz and leave me alone 😩",
		from.Format("02.01.2006"),
		to.Format("02.01.2006"),
	)

	reportFile := &telebot.Document{
		File:     telebot.FromReader(bytes.NewReader(cachedReport)),
		FileName: fmt.Sprintf("report_%s_%s.xlsx", from.Format("2006-01-02"), to.Format("2006-01-02")),
		MIME:     "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	}

	b.metrics.SentMessages.WithLabelValues("edit").Inc()
	_ = tbCtx.Edit(responseText, tbCtx.Message().ReplyMarkup)
	b.metrics.SentMessages.WithLabelValues("file").Inc()
	return true, tbCtx.Send(reportFile)
}

func (b *Bot) generateAndSendReport(
	ctx context.Context,
	tbCtx telebot.Context,
	userID int64,
	from, to time.Time,
	periodMetric, cacheKey string,
) error {
	b.log.InfoContext(ctx, "Report not found in cache, generating a new one", "user", userID, "key", cacheKey)

	startTime := time.Now()
	excelRows, err := b.formatExcelRows(ctx, userID, from, to)
	if err != nil {
		b.log.ErrorContext(ctx, "Failed to format excel rows for report generator", "error", err)
	}
	reportBuffer, err := report.GenerateExcelReport(excelRows)
	b.metrics.ReportGeneration.WithLabelValues(periodMetric).Observe(time.Since(startTime).Seconds())
	if err != nil {
		if errors.Is(err, report.ErrNoTasks) {
			b.metrics.SentMessages.WithLabelValues("edit").Inc()
			return tbCtx.Edit("💩 There are no completed tasks for the report for the selected period.",
				tbCtx.Message().ReplyMarkup)
		}
		b.metrics.SentMessages.WithLabelValues("error").Inc()
		b.log.ErrorContext(ctx, "Failed to generate report", "error", err, "user", userID)
		return tbCtx.Edit(ErrInternal, tbCtx.Message().ReplyMarkup)
	}

	const cacheTTL = 1 * time.Hour
	if err = b.redisClient.Set(ctx, cacheKey, reportBuffer.Bytes(), cacheTTL).Err(); err != nil {
		b.metrics.CacheOps.WithLabelValues("set", "error").Inc()
		b.log.ErrorContext(ctx, "Failed to save report to cache", "error", err, "key", cacheKey)
	} else {
		b.metrics.CacheOps.WithLabelValues("set", "success").Inc()
	}

	responseText := fmt.Sprintf(
		"💩 Your report for the period %s to %s is ready.\nJust pass it on to Tanz and leave me alone 😩",
		from.Format("02.01.2006"),
		to.Format("02.01.2006"),
	)

	reportFile := &telebot.Document{
		File:     telebot.FromReader(reportBuffer),
		FileName: fmt.Sprintf("report_%s_%s.xlsx", from.Format("2006-01-02"), to.Format("2006-01-02")),
		MIME:     "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	}

	b.metrics.SentMessages.WithLabelValues("edit").Inc()
	_ = tbCtx.Edit(responseText, tbCtx.Message().ReplyMarkup)
	b.metrics.SentMessages.WithLabelValues("file").Inc()
	return tbCtx.Send(reportFile)
}

// nearTasksHandler handles the user's request for nearby tasks.
// It logs the request, increments metrics for command reception and sent messages,
// updates the user's state to await location input, and replies with a message
// prompting the user to provide their geolocation.
// This feature is currently in beta testing, and users are encouraged to report any errors.
func (b *Bot) nearTasksHandler(ctx telebot.Context) error {
	b.log.Info("User requested near tasks", "user", ctx.Sender().ID)
	b.metrics.CommandReceived.WithLabelValues("near").Inc()

	userStates[ctx.Sender().ID] = stateAwaitingLocation

	b.metrics.SentMessages.WithLabelValues("reply").Inc()
	return ctx.Reply(
		"🧳 I'm ready, but first provide your geolocation",
		nearMenu,
		telebot.ModeMarkdownV2,
	)
}
