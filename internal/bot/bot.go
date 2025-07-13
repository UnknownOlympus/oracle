package bot

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/Houeta/radireporter-bot/internal/metrics"
	"github.com/Houeta/radireporter-bot/internal/repository"
	"gopkg.in/telebot.v4"
)

// Bot contains the bot API instance and other information.
type Bot struct {
	bot     *telebot.Bot
	log     *slog.Logger
	repo    repository.Interface
	metrics *metrics.Metrics
}

var (
	// main menu for unathorized users.
	mainMenu = &telebot.ReplyMarkup{ResizeKeyboard: true}
	// button for login.
	btnLogin = mainMenu.Text("🔐 Login")

	// inline menu for authorized users.
	authMenu = &telebot.ReplyMarkup{ResizeKeyboard: true}
	// button for info.
	btnInfo = authMenu.Text("🙍‍♂️ About me")
	// button for active tasks.
	btnActiveTasks = authMenu.Text("✅ Active tasks")
	// button for near tasks.
	btnNear = authMenu.Text("🗺️ Tasks near you")
	// button for statistic.
	btnStatistic = authMenu.Text("📈 My statistic")
	// button for report.
	btnReport = authMenu.Text("📊 Create report")
	// button for logout.
	btnLogout = authMenu.Text("🔓 Logout")

	// statistic menu.
	statMenu = &telebot.ReplyMarkup{ResizeKeyboard: true}
	// button for today statistic.
	btnToday = statMenu.Text("📅 Today")
	// button for this month statistic.
	btnMonth = statMenu.Text("📅 This Month")
	// button fot this year statistic.
	btnYear = statMenu.Text("📅 This Year")
	// button for back.
	btnBack = statMenu.Text("⬅️ Back")

	nearMenu = &telebot.ReplyMarkup{ResizeKeyboard: true}
	// button for send location.
	btnLocation = authMenu.Location("📍  Send location")

	// inline buttons for report period.
	btnReportPeriodCurrent = telebot.InlineButton{Unique: "report_period_current_month"}
	btnReportPeriodLast    = telebot.InlineButton{Unique: "report_period_last_month"}
	btnReportPeriod7Days   = telebot.InlineButton{Unique: "report_period_last_7_days"}

	// fiction button for active tasks action.
	btnTaskDetails = telebot.InlineButton{
		Unique: "task_details",
	}
)

// NewBot creates a new bot with the given token.
func NewBot(
	log *slog.Logger,
	repo repository.Interface,
	metrics *metrics.Metrics,
	token string,
	poller time.Duration,
) (*Bot, error) {
	bot, err := telebot.NewBot(telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: poller},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Telegram bot: %w", err)
	}
	log.Info("Authorized on account", "account", bot.Me.Username)

	botInstance := &Bot{bot: bot, log: log, repo: repo, metrics: metrics}

	mainMenu.Reply(
		mainMenu.Row(btnLogin),
	)
	authMenu.Reply(
		authMenu.Row(btnInfo),
		authMenu.Row(btnActiveTasks),
		authMenu.Row(btnNear),
		authMenu.Row(btnStatistic),
		authMenu.Row(btnReport),
		authMenu.Row(btnLogout),
	)
	statMenu.Reply(
		authMenu.Row(btnToday),
		authMenu.Row(btnMonth),
		authMenu.Row(btnYear),
		authMenu.Row(btnBack),
	)
	nearMenu.Reply(
		nearMenu.Row(btnLocation),
		nearMenu.Row(btnBack),
	)

	botInstance.registerRoutes()

	return botInstance, nil
}

// Start launches the bot to listen for updates.
func (b *Bot) Start() {
	b.log.Info("Telegram bot is starting...")
	b.bot.Start()
}

// Stop gracefully stops the Telegram bot and logs the action.
func (b *Bot) Stop() {
	b.log.Info("Telegram bot is stopped...")
	b.bot.Stop()
}

// registerRoutes configures all routes (commands).
func (b *Bot) registerRoutes() {
	// Public routes.
	b.bot.Handle("/start", b.startHandler)
	b.bot.Handle(&btnLogin, b.authHandler)
	b.bot.Handle(telebot.OnText, b.textHandler)
	b.bot.Handle(&btnTaskDetails, b.taskDetailsHandler)
	b.bot.Handle(telebot.OnLocation, b.locationHandler)

	// group for protected routes.
	authGroup := b.bot.Group()
	authGroup.Use(b.AuthMiddleware)

	// Protected routes.
	authGroup.Handle(&btnReport, b.reportHandler)
	authGroup.Handle(&btnReportPeriodCurrent, b.generatorReportHandler)
	authGroup.Handle(&btnReportPeriodLast, b.generatorReportHandler)
	authGroup.Handle(&btnReportPeriod7Days, b.generatorReportHandler)

	authGroup.Handle(&btnActiveTasks, b.activeTasksHandler)
	authGroup.Handle(&btnStatistic, b.statisticHandler)
	authGroup.Handle(&btnLogout, b.logoutHandler)
	authGroup.Handle(&btnInfo, b.infoHandler)

	authGroup.Handle(&btnToday, b.statisticHandlerToday)
	authGroup.Handle(&btnMonth, b.statisticHandlerMonth)
	authGroup.Handle(&btnYear, b.statisticHandlerYear)
	authGroup.Handle(&btnBack, b.backHandler)

	authGroup.Handle(&btnNear, b.nearTasksHandler)
}
