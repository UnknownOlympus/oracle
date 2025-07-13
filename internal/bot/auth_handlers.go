package bot

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Houeta/radireporter-bot/internal/report"
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
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	startTime := time.Now()
	user, err := b.repo.GetEmployee(timeoutCtx, ctx.Sender().ID)
	b.metrics.DBQueryDuration.WithLabelValues("get_employee").Observe(time.Since(startTime).Seconds())
	if err != nil {
		b.log.Error("Failed to get employee data", "error", err)
		b.metrics.SentMessages.WithLabelValues("error").Inc()
		return ctx.Send(ErrInternal)
	}

	b.metrics.SentMessages.WithLabelValues("text").Inc()
	responseText := fmt.Sprintf(`
		🤦‍♂️ *These mortals again…*

		*Name:* %s
		*Position:* %s
		*Email:* %s
		*Phone:* %s
		
		💬 Okay, I saved this somewhere… or not.
	`,
		user.FullName, user.Position, user.Email, user.Phone)

	return ctx.Send(responseText, telebot.ModeMarkdown)
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

	startTime := time.Now()
	details, err := b.repo.GetTaskDetailsByID(timeoutCtx, taskID)
	b.metrics.DBQueryDuration.WithLabelValues("get_active_tasks").Observe(time.Since(startTime).Seconds())
	if err != nil {
		b.log.Error("Failed to get task details", "error", err, "taskID", taskID)
		b.metrics.SentMessages.WithLabelValues("error").Inc()
		return ctx.Respond(&telebot.CallbackResponse{Text: "Error retrieving data."})
	}

	// format detail information
	messageText := fmt.Sprintf(
		"*Task details #%d*\n\n"+
			"*Type:* %s\n"+
			"*Created:* %s\n"+
			"*Client Name:* %s\n"+
			"*Address:* %s\n"+
			"*Description:* %s\n"+
			"*Assigned to:* %s",
		details.ID,
		details.Type,
		details.CreationDate.Format("02.01.2006"),
		details.CustomerName,
		details.Address,
		details.Description,
		strings.Join(details.Executors, ", "),
	)
	if details.Latitude.Valid && details.Longitude.Valid {
		mapURL := fmt.Sprintf("https://maps.google.com/?q=%f,%f", details.Latitude.Float64, details.Longitude.Float64)
		messageText += fmt.Sprintf("\n\n[📍 Open on map](%s)", mapURL)
	} else {
		messageText += "\n\n📍 *Location not added yet*"
	}

	b.metrics.SentMessages.WithLabelValues("edit").Inc()
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
	_ = ctx.Respond(&telebot.CallbackResponse{Text: "🔧 I'll do it now!!! Wait...😩"})
	userID := ctx.Sender().ID
	b.log.Info("User requested report", "user", userID, "data", ctx.Callback().Unique)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var from, to time.Time
	var periodMetric string
	now := time.Now()

	switch ctx.Callback().Unique {
	case "report_period_current_month":
		periodMetric = "current_1m"
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		to = from.AddDate(0, 1, 0).Add(-1 * time.Nanosecond)
	case "report_period_last_month":
		periodMetric = "last_1m"
		from = time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location())
		to = from.AddDate(0, 1, 0).Add(-1 * time.Nanosecond)
	case "report_period_last_7_days":
		periodMetric = "last_7d"
		from = now.AddDate(0, 0, -7)
		to = now
	default:
		b.metrics.SentMessages.WithLabelValues("error").Inc()
		return ctx.Edit("💩 Unsupported time period", ctx.Message().ReplyMarkup)
	}

	startTime := time.Now()
	reportBuffer, err := report.GenerateExcelReport(timeoutCtx, b.repo, userID, from, to)
	b.metrics.ReportGeneration.WithLabelValues(periodMetric).Observe(time.Since(startTime).Seconds())
	if err != nil {
		if errors.Is(err, report.ErrNoTasks) {
			b.metrics.SentMessages.WithLabelValues("edit").Inc()
			return ctx.Edit("💩 There are no completed tasks for the report for the selected period.",
				ctx.Message().ReplyMarkup)
		}

		b.metrics.SentMessages.WithLabelValues("error").Inc()
		b.log.Error("Failed to generate report", "error", err, "user", userID)
		return ctx.Edit(ErrInternal, ctx.Message().ReplyMarkup)
	}

	reportFile := &telebot.Document{
		File:     telebot.FromReader(reportBuffer),
		FileName: fmt.Sprintf("report_%s_%s.xlsx", from.Format("2006-01-02"), to.Format("2006-01-02")),
		MIME:     "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	}

	reponseText := fmt.Sprintf(
		"💩 Your report for the period %s to %s is ready.\n"+
			"Just pass it on to Tanz and leave me alone 😩",
		from.Format("02.01.2006"), to.Format("02.01.2006"),
	)

	b.metrics.SentMessages.WithLabelValues("edit").Inc()
	_ = ctx.Edit(reponseText, ctx.Message().ReplyMarkup)

	b.metrics.SentMessages.WithLabelValues("text").Inc()
	return ctx.Send(reportFile)
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
		"🧳 I'm ready, but first provide your geolocation\n\n*NOTE:* This feature is in beta testing\\.\nIf you see any errors: please report them to your admin\\.",
		nearMenu,
		telebot.ModeMarkdownV2,
	)
}
