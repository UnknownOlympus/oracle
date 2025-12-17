package bot

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/UnknownOlympus/olympus-protos/gen/go/scraper/olympus"
	"github.com/UnknownOlympus/oracle/internal/i18n"
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
	localizer    *i18n.Localizer
}

var (
	// inline buttons for report period.
	btnReportPeriodCurrent = telebot.InlineButton{Unique: "report_period_current_month"}
	btnReportPeriodLast    = telebot.InlineButton{Unique: "report_period_last_month"}
	btnReportPeriod7Days   = telebot.InlineButton{Unique: "report_period_last_7_days"}

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

	localizer, err := i18n.NewLocalizer()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize localizer: %w", err)
	}

	botInstance := &Bot{
		bot:          bot,
		log:          log,
		usrepo:       usrepo,
		tarepo:       tarepo,
		metrics:      metrics,
		redisClient:  redisClient,
		hermesClient: hermesClient,
		stateManager: stateManager,
		localizer:    localizer,
	}

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
	b.bot.Handle("/language", b.languageHandler)
	b.bot.Handle(telebot.OnText, b.routeTextHandler)
	b.bot.Handle(&btnTaskDetails, b.taskDetailsHandler)
	b.bot.Handle(telebot.OnLocation, b.locationHandler)

	// Language selection callbacks
	b.bot.Handle("\flanguage_en", b.languageChangeHandler)
	b.bot.Handle("\flanguage_uk", b.languageChangeHandler)

	// Inline button callbacks
	b.bot.Handle(&btnReportPeriodCurrent, b.generatorReportHandler)
	b.bot.Handle(&btnReportPeriodLast, b.generatorReportHandler)
	b.bot.Handle(&btnReportPeriod7Days, b.generatorReportHandler)
	b.bot.Handle("\fleave_comment", b.addCommentHandler)
	b.bot.Handle("\fcomment_accept", b.commentAcceptHandler)
	b.bot.Handle("\fcomment_decline", b.commentDeclineHandler)
}

// getUserLanguage retrieves the user's language preference from the database.
// It returns the language code, falling back to auto-detection from Telegram if not set.
func (b *Bot) getUserLanguage(ctx context.Context, tCtx telebot.Context) string {
	userID := tCtx.Sender().ID

	// Try to get saved language preference
	lang, err := b.usrepo.GetUserLanguage(ctx, userID)
	if err != nil {
		b.log.WarnContext(ctx, "Failed to get user language, using default", "error", err, "userID", userID)
		return "en"
	}

	// If language is not set, try to detect from Telegram and save it
	if lang == "en" && tCtx.Sender().LanguageCode != "" {
		detectedLang := i18n.NormalizeLanguageCode(tCtx.Sender().LanguageCode)
		if detectedLang != "en" {
			// Save detected language asynchronously
			go func() {
				saveCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				if err = b.usrepo.SetUserLanguage(saveCtx, userID, detectedLang); err != nil {
					b.log.ErrorContext(saveCtx, "Failed to save detected language", "error", err, "userID", userID)
				}
			}()
			return detectedLang
		}
	}

	return lang
}

// t is a shorthand method for getting translations.
func (b *Bot) t(ctx context.Context, tCtx telebot.Context, key string) string {
	lang := b.getUserLanguage(ctx, tCtx)
	return b.localizer.Get(lang, key)
}

// tWithData is a shorthand method for getting translations with placeholder data.
func (b *Bot) tWithData(ctx context.Context, tCtx telebot.Context, key string, data map[string]interface{}) string {
	lang := b.getUserLanguage(ctx, tCtx)
	return b.localizer.GetWithData(lang, key, data)
}
