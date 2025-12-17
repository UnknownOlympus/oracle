package bot

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/UnknownOlympus/oracle/internal/repository"
	"github.com/google/uuid"
	"gopkg.in/telebot.v4"
	"gopkg.in/telebot.v4/react"
)

// var userStates = make(map[int64]string)

const (
	// stateAwaitingEmail indicates that the bot is waiting for the user's email input.
	stateAwaitingEmail = "email"

	// stateAwaitingLocation indicates that the bot is waiting fot the user's location input.
	stateAwaitingLocation = "location"

	// stateComment indicates that the bot is waiting fot the user's text comment input.
	stateComment = "comment"

	// stateComment indicates that the bot is waiting fot the user's text broadcast input.
	stateAwaitingBroadcast = "broadcast"

	// ErrInternal is the error message returned when there is an internal server error.
	ErrInternal = "🚫 Internal server error, please try again later"
)

// startHandler process command /start.
func (b *Bot) startHandler(ctx telebot.Context) error {
	var responseText string
	var selectedMenu *telebot.ReplyMarkup
	userID := ctx.Sender().ID
	metricLabel := "text"

	b.log.Info("User started the bot", "id", userID, "username", ctx.Sender().Username)
	b.metrics.CommandReceived.WithLabelValues("start").Inc()

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	startTime := time.Now()
	isAuth, err := b.usrepo.IsUserAuthenticated(timeoutCtx, userID)
	b.metrics.DBQueryDuration.WithLabelValues("is_user_authenticated").Observe(time.Since(startTime).Seconds())

	switch {
	case err != nil:
		responseText = b.t(timeoutCtx, ctx, "error.internal")
		selectedMenu = b.buildMainMenu(timeoutCtx, ctx)
		metricLabel = "error"
	case isAuth:
		responseText = b.t(timeoutCtx, ctx, "welcome.authenticated")
		isAdmin, adminErr := b.usrepo.IsAdmin(timeoutCtx, userID)
		if adminErr != nil {
			b.log.ErrorContext(timeoutCtx, "Failed to check admin status", "error", adminErr)
			responseText = b.t(timeoutCtx, ctx, "error.internal")
			selectedMenu = b.buildMainMenu(timeoutCtx, ctx)
			metricLabel = "error"
		} else {
			selectedMenu = b.buildAuthMenuWithTranslations(timeoutCtx, ctx, isAdmin)
		}
	case !isAuth:
		responseText = b.t(timeoutCtx, ctx, "welcome.unauthenticated")
		selectedMenu = b.buildMainMenu(timeoutCtx, ctx)
		b.metrics.NewUsers.Inc()
	}

	b.metrics.SentMessages.WithLabelValues(metricLabel).Inc()

	return ctx.Send(responseText, selectedMenu)
}

// authHandler handles the authentication process for the bot.
// It prompts the user to enter their email address, which is required for
// verification in the US system. The user's state is updated to indicate
// that the bot is awaiting the email input.
func (b *Bot) authHandler(ctx telebot.Context) error {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	b.stateManager.Set(ctx.Sender().ID, UserState{WaitingFor: stateAwaitingEmail})
	b.metrics.CommandReceived.WithLabelValues("login").Inc()
	b.metrics.SentMessages.WithLabelValues("text").Inc()
	return ctx.Send(b.t(timeoutCtx, ctx, "login.prompt"))
}

// routeTextHandler routes text messages to appropriate handlers based on button text or state.
func (b *Bot) routeTextHandler(ctx telebot.Context) error {
	text := ctx.Text()
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Get user's language to check button text
	lang := b.getUserLanguage(timeoutCtx, ctx)

	// Try to match button text in current language first
	// Then try both languages as fallback (in case language preference is not set correctly)
	languages := []string{lang}
	if lang != "en" {
		languages = append(languages, "en")
	} else {
		languages = append(languages, "uk")
	}

	for _, checkLang := range languages {
		switch text {
		// Main menu buttons
		case b.localizer.Get(checkLang, "menu.login"):
			return b.authHandler(ctx)
		case b.localizer.Get(checkLang, "menu.about_me"):
			return b.infoHandler(ctx)
		case b.localizer.Get(checkLang, "menu.active_tasks"):
			return b.activeTasksHandler(ctx)
		case b.localizer.Get(checkLang, "menu.tasks_near"):
			return b.nearTasksHandler(ctx)
		case b.localizer.Get(checkLang, "menu.my_statistic"):
			return b.statistic(ctx)
		case b.localizer.Get(checkLang, "menu.create_report"):
			return b.reportHandler(ctx)
		case b.localizer.Get(checkLang, "menu.language"):
			return b.languageHandler(ctx)
		case b.localizer.Get(checkLang, "menu.admin_panel"):
			return b.adminPanelHandler(ctx)
		case b.localizer.Get(checkLang, "menu.logout"):
			return b.logoutHandler(ctx)
		case b.localizer.Get(checkLang, "menu.broadcast"):
			return b.broadcastInitiateHandler(ctx)
		case b.localizer.Get(checkLang, "menu.today"):
			return b.statisticHandlerToday(ctx)
		case b.localizer.Get(checkLang, "menu.this_month"):
			return b.statisticHandlerMonth(ctx)
		case b.localizer.Get(checkLang, "menu.this_year"):
			return b.statisticHandlerYear(ctx)
		case b.localizer.Get(checkLang, "menu.back"):
			return b.backHandler(ctx)
		}
	}

	// If not a button, handle as regular text (email, comment, broadcast, etc.)
	return b.textHandler(ctx)
}

// textHandler processes incoming text messages from users. It checks the user's state,
// validates the provided email, and attempts to link the Telegram ID with the email.
// If successful, it sends a confirmation message; otherwise, it handles various error cases
// such as already linked accounts or user not found, providing appropriate feedback to the user.
func (b *Bot) textHandler(ctx telebot.Context) error {
	userID := ctx.Sender().ID
	state, ok := b.stateManager.Get(userID)
	if !ok {
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		b.metrics.SentMessages.WithLabelValues("reply").Inc()
		return ctx.Reply(b.t(timeoutCtx, ctx, "general.use_buttons"))
	}

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	switch state.WaitingFor {
	case stateAwaitingEmail:
		email := ctx.Text()
		b.log.Debug("User is trying to authenticate", "user", userID, "email", email)
		return b.loginInputHandler(timeoutCtx, ctx, userID, email)
	case stateComment:
		comment := ctx.Text()
		b.log.Debug("User is trying to add comment", "user", userID, "comment_length", len(comment))
		return b.commentConfirmationHandler(ctx, state.TaskID, comment)
	case stateAwaitingBroadcast:
		text := ctx.Text()
		b.log.Debug("User is trying to send broadcast message to everyone", "user", userID)
		return b.broadcastMessageHandler(timeoutCtx, ctx, text)
	default:
		b.log.Error("Get unknown state", "state", state.WaitingFor)
		b.metrics.SentMessages.WithLabelValues("error").Inc()
		return ctx.Send(ErrInternal)
	}
}

func (b *Bot) loginInputHandler(ctx context.Context, bCtx telebot.Context, userID int64, email string) error {
	startTime := time.Now()
	err := b.usrepo.LinkTelegramIDByEmail(ctx, userID, email)
	b.metrics.DBQueryDuration.WithLabelValues("link_telegram_id").Observe(time.Since(startTime).Seconds())
	if err != nil {
		if errors.Is(err, repository.ErrUserAlreadyLinked) {
			b.log.InfoContext(ctx, "User already linked to another id", "user", userID, "email", email)
			_ = bCtx.Bot().React(bCtx.Recipient(), bCtx.Message(), react.React(react.ThumbDown))
			b.metrics.SentMessages.WithLabelValues("reaction").Inc()
			b.metrics.SentMessages.WithLabelValues("user_error").Inc()
			return bCtx.Send(b.t(ctx, bCtx, "login.error.already_linked"))
		}
		if errors.Is(err, repository.ErrIDExists) {
			b.log.InfoContext(ctx, "User already has connection with another employee", "user", userID, "email", email)
			b.metrics.SentMessages.WithLabelValues("reaction").Inc()
			b.metrics.SentMessages.WithLabelValues("user_error").Inc()
			_ = bCtx.Bot().React(bCtx.Recipient(), bCtx.Message(), react.React(react.ThumbDown))
			return bCtx.Send(b.t(ctx, bCtx, "login.error.id_exists"))
		}
		if errors.Is(err, repository.ErrUserNotFound) {
			b.log.InfoContext(ctx, "User with this email not found", "user", userID, "email", email)
			b.metrics.SentMessages.WithLabelValues("reaction").Inc()
			b.metrics.SentMessages.WithLabelValues("user_error").Inc()
			_ = bCtx.Bot().React(bCtx.Recipient(), bCtx.Message(), react.React(react.ThumbDown))
			b.stateManager.Set(userID, UserState{WaitingFor: stateAwaitingEmail})
			return bCtx.Send(b.t(ctx, bCtx, "login.error.not_found"))
		}
		b.log.ErrorContext(ctx, "Failed to link telegram id with employee", "error", err)
		b.metrics.SentMessages.WithLabelValues("error").Inc()
		return bCtx.Send(b.t(ctx, bCtx, "error.internal"))
	}

	isAdmin, err := b.usrepo.IsAdmin(ctx, userID)
	if err != nil {
		b.log.ErrorContext(ctx, "Failed to check admin status", "error", err)
		b.metrics.SentMessages.WithLabelValues("error").Inc()
		return bCtx.Send(b.t(ctx, bCtx, "error.internal"))
	}

	menu := b.buildAuthMenuWithTranslations(ctx, bCtx, isAdmin)

	b.log.InfoContext(ctx, "User successfully authenticated", "user", userID, "email", email)
	b.metrics.SentMessages.WithLabelValues("reaction").Inc()
	b.metrics.SentMessages.WithLabelValues("text").Inc()
	_ = bCtx.Bot().React(bCtx.Recipient(), bCtx.Message(), react.React(react.ThumbUp))
	return bCtx.Send(b.t(ctx, bCtx, "login.success"), menu)
}

func (b *Bot) commentConfirmationHandler(ctx telebot.Context, taskID int, commentText string) error {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	startTime := time.Now()
	user, err := b.tarepo.GetEmployee(timeoutCtx, ctx.Sender().ID)
	b.metrics.DBQueryDuration.WithLabelValues("get_employee").Observe(time.Since(startTime).Seconds())
	if err != nil {
		b.log.Error("Failed to get employee data", "error", err)
		b.metrics.SentMessages.WithLabelValues("error").Inc()
		return ctx.Send(b.t(timeoutCtx, ctx, "error.internal"))
	}

	formattedComment := fmt.Sprintf("👤 %s: %s", user.ShortName, commentText)
	messageText := b.tWithData(timeoutCtx, ctx, "comment.preview", map[string]interface{}{
		"comment": formattedComment,
	})

	confirmationID := uuid.New().String()
	cacheKey := fmt.Sprintf("oracle:comment_confirm:%s", confirmationID)
	const cacheTTL = 5 * time.Minute

	err = b.redisClient.Set(timeoutCtx, cacheKey, commentText, cacheTTL).Err()
	if err != nil {
		b.log.Error("Failed to save comment to confirmation cache", "error", err)
		b.metrics.SentMessages.WithLabelValues("error").Inc()
		return ctx.Send(b.t(timeoutCtx, ctx, "error.internal"))
	}

	callbackData := fmt.Sprintf("%d|%s", taskID, confirmationID)

	confirmMenu := &telebot.ReplyMarkup{ResizeKeyboard: true}
	btnAccept := confirmMenu.Data(b.t(timeoutCtx, ctx, "comment.button.accept"), "comment_accept", callbackData)
	btnDecline := confirmMenu.Data(b.t(timeoutCtx, ctx, "comment.button.decline"), "comment_decline")
	confirmMenu.Inline(confirmMenu.Row(btnAccept, btnDecline))

	b.log.Debug("Succesfully get comment from user, sending confiramtion request.", "user", ctx.Sender().ID)
	b.metrics.SentMessages.WithLabelValues("text").Inc()
	return ctx.Send(messageText, confirmMenu, telebot.ModeMarkdown)
}

// locationHandler processes the user's location sent via a message.
// It retrieves tasks within a specified radius of the user's location
// and sends back a response with the nearest tasks or an appropriate
// message if no tasks are found. It also handles user state management
// and logs relevant information for monitoring purposes.
func (b *Bot) locationHandler(ctx telebot.Context) error {
	userID := ctx.Sender().ID
	latitude := ctx.Message().Location.Lat
	longitude := ctx.Message().Location.Lng
	radius := 15
	state, ok := b.stateManager.Get(userID)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	b.log.Info("User sent geolocation", "user", userID, "latitude", latitude, "longitude", longitude)

	if ok && state.WaitingFor == stateAwaitingLocation {
		startTime := time.Now()
		tasks, err := b.tarepo.GetTasksInRadius(timeoutCtx, latitude, longitude, radius)
		b.metrics.DBQueryDuration.WithLabelValues("get_tasks_in_radius").Observe(time.Since(startTime).Seconds())
		if err != nil {
			b.log.Error("Failed to get nearest tasks", "error", err)
			b.metrics.SentMessages.WithLabelValues("error").Inc()
			return ctx.Send(b.t(timeoutCtx, ctx, "error.internal"))
		}

		if len(tasks) == 0 {
			b.metrics.SentMessages.WithLabelValues("text").Inc()
			return ctx.Send(b.t(timeoutCtx, ctx, "tasks.near.none"))
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

		menu := &telebot.ReplyMarkup{InlineKeyboard: rows}
		responseText := b.tWithData(timeoutCtx, ctx, "tasks.near.title", map[string]interface{}{
			"radius": radius,
		})
		b.metrics.SentMessages.WithLabelValues("text").Inc()
		return ctx.Send(responseText, menu)
	}

	b.metrics.SentMessages.WithLabelValues("text").Inc()
	return ctx.Send(b.t(timeoutCtx, ctx, "tasks.near.unsolicited"))
}
