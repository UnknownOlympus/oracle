package bot

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/UnknownOlympus/olympus-protos/gen/go/scraper/olympus"
	"github.com/UnknownOlympus/oracle/internal/metrics"
	"github.com/UnknownOlympus/oracle/internal/repository"
	"github.com/redis/go-redis/v9"
	"gopkg.in/telebot.v4"
)

// Bot contains the bot API instance and other information.
type Bot struct {
	bot          *telebot.Bot
	log          *slog.Logger
	usrepo       repository.BotManager
	tarepo       repository.TaskManager
	metrics      *metrics.Metrics
	redisClient  *redis.Client
	hermesClient olympus.ScraperServiceClient
	stateManager *StateManager
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
	// button for administrators.
	btnAdmin = authMenu.Text("👑 Admin Panel")
	// button for logout.
	btnLogout = authMenu.Text("🔓 Logout")

	adminMenu = &telebot.ReplyMarkup{ResizeKeyboard: true}
	// admin buttons.
	btnBroadcast = adminMenu.Text("📣 A Decree for the Mortals")

	// statistic menu.
	statMenu = &telebot.ReplyMarkup{ResizeKeyboard: true}
	// button for today statistic.
	btnToday = statMenu.Text("📅 Today")
	// button for this month statistic.
	btnMonth = statMenu.Text("📅 This Month")
	// button fot this year statistic.
	btnYear = statMenu.Text("📅 This Year")
	// button for back.
	btnBack = statMenu.Text("⬅️ Back to Main Menu")

	nearMenu = &telebot.ReplyMarkup{ResizeKeyboard: true}
	// button for send location.
	btnLocation = nearMenu.Location("📍  Send location")

	// inline buttons for report period.
	btnReportPeriodCurrent = telebot.InlineButton{Unique: "report_period_current_month"}
	btnReportPeriodLast    = telebot.InlineButton{Unique: "report_period_last_month"}
	btnReportPeriod7Days   = telebot.InlineButton{Unique: "report_period_last_7_days"}

	// Inline menu for comment confirmation.
	confirmMenu = &telebot.ReplyMarkup{ResizeKeyboard: true}

	// fiction button for active tasks action.
	btnTaskDetails = telebot.InlineButton{Unique: "task_details"}
)

// NewBot creates a new bot with the given token.
func NewBot(
	log *slog.Logger,
	usrepo repository.BotManager,
	tarepo repository.TaskManager,
	redisClient *redis.Client,
	hermesClient olympus.ScraperServiceClient,
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

	stateManager := NewStateManager()

	botInstance := &Bot{
		bot:          bot,
		log:          log,
		usrepo:       usrepo,
		tarepo:       tarepo,
		metrics:      metrics,
		redisClient:  redisClient,
		hermesClient: hermesClient,
		stateManager: stateManager,
	}

	mainMenu.Reply(
		mainMenu.Row(btnLogin),
	)
	statMenu.Reply(
		statMenu.Row(btnToday),
		statMenu.Row(btnMonth),
		statMenu.Row(btnYear),
		statMenu.Row(btnBack),
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
	authGroup.Handle("\fleave_comment", b.addCommentHandler)
	authGroup.Handle("\fcomment_accept", b.commentAcceptHandler)
	authGroup.Handle("\fcomment_decline", b.commentDeclineHandler)
	authGroup.Handle(&btnStatistic, b.statistic)
	authGroup.Handle(&btnLogout, b.logoutHandler)
	authGroup.Handle(&btnInfo, b.infoHandler)

	authGroup.Handle(&btnToday, b.statisticHandlerToday)
	authGroup.Handle(&btnMonth, b.statisticHandlerMonth)
	authGroup.Handle(&btnYear, b.statisticHandlerYear)
	authGroup.Handle(&btnBack, b.backHandler)

	authGroup.Handle(&btnNear, b.nearTasksHandler)

	// Handler for opening the admin panel
	authGroup.Handle(&btnAdmin, b.adminPanelHandler)
	// Handler for the broadcast feature
	authGroup.Handle(&btnBroadcast, b.broadcastInitiateHandler)
}

func (b *Bot) buildAuthMenu(isAdmin bool) *telebot.ReplyMarkup {
	rows := []telebot.Row{
		authMenu.Row(btnInfo),
		authMenu.Row(btnActiveTasks),
		authMenu.Row(btnNear),
		authMenu.Row(btnStatistic),
		authMenu.Row(btnReport),
	}
	if isAdmin {
		rows = append(rows, authMenu.Row(btnAdmin))
	}
	rows = append(rows, authMenu.Row(btnLogout))

	authMenu.Reply(rows...)

	return authMenu
}

func (b *Bot) getMenuForUser(ctx context.Context, userID int64) (*telebot.ReplyMarkup, error) {
	isAdmin, err := b.usrepo.IsAdmin(ctx, userID)
	if err != nil {
		b.log.ErrorContext(ctx, "Failed to get response from DB about user privileges", "userID", userID)
		return nil, fmt.Errorf("failed to get response from DB about user privileges: %w", err)
	}

	return b.buildAuthMenu(isAdmin), nil
}
