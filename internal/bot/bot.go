package bot

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/Houeta/radireporter-bot/internal/repository"
	"gopkg.in/telebot.v4"
)

// Bot contains the bot API instance and other information.
type Bot struct {
	bot  *telebot.Bot
	log  *slog.Logger
	repo repository.Interface
}

var (
	// main menu for unathorized users.
	mainMenu = &telebot.ReplyMarkup{ResizeKeyboard: true}
	// button for login.
	btnLogin = mainMenu.Text("🔐 Login")

	// inline menu for authorized users.
	authMenu = &telebot.ReplyMarkup{ResizeKeyboard: true}
	// button for report.
	btnReport = authMenu.Text("📊 Create report")
	// button for logout.
	btnLogout = authMenu.Text("🔓 Logout")
)

// NewBot creates a new bot with the given token.
func NewBot(log *slog.Logger, repo repository.Interface, token string, poller time.Duration) (*Bot, error) {
	bot, err := telebot.NewBot(telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: poller},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Telegram bot: %w", err)
	}
	log.Info("Authorized on account", "account", bot.Me.Username)

	botInstance := &Bot{bot: bot, log: log, repo: repo}

	mainMenu.Reply(
		mainMenu.Row(btnLogin),
	)
	authMenu.Reply(
		authMenu.Row(btnReport),
		authMenu.Row(btnLogout),
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

	// group for protected routes.
	authGroup := b.bot.Group()
	authGroup.Use(b.AuthMiddleware)

	// Protected routes.
	authGroup.Handle(&btnReport, b.reportHandler)
	authGroup.Handle(&btnLogout, b.logoutHandler)
}
