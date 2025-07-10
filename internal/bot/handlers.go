package bot

import (
	"context"
	"errors"
	"time"

	"github.com/Houeta/radireporter-bot/internal/repository"
	"gopkg.in/telebot.v4"
	"gopkg.in/telebot.v4/react"
)

var userStates = make(map[int64]string)

const (
	// stateAwaitingEmail indicates that the bot is waiting for the user's email input.
	stateAwaitingEmail = "awaiting_email"

	// ErrInternal is the error message returned when there is an internal server error.
	ErrInternal = "🚫 Internal server error, please try again later"
)

// startHandler process command /start.
func (b *Bot) startHandler(ctx telebot.Context) error {
	var responseText string
	selectedMenu := mainMenu
	userID := ctx.Sender().ID
	metricLabel := "text"

	b.log.Info("User started the bot", "id", userID, "username", ctx.Sender().Username)
	b.metrics.CommandReceived.WithLabelValues("start").Inc()

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	startTime := time.Now()
	isAuth, err := b.repo.IsUserAuthenticated(timeoutCtx, userID)
	b.metrics.DBQueryDuration.WithLabelValues("is_user_authenticated").Observe(time.Since(startTime).Seconds())

	switch {
	case err != nil:
		responseText = ErrInternal
		metricLabel = "error"
	case isAuth:
		responseText = "🤡 Welcome to the almshouse, slave of Radionet!"
		selectedMenu = authMenu
	case !isAuth:
		responseText = "🤡 Welcome to the almshouse, slave of Radionet!\nTo access features, please log in."
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
	userStates[ctx.Sender().ID] = stateAwaitingEmail
	b.metrics.CommandReceived.WithLabelValues("login").Inc()
	b.metrics.SentMessages.WithLabelValues("text").Inc()
	return ctx.Send("📧 Enter your email address, which is specified in the US system..")
}

// textHandler processes incoming text messages from users. It checks the user's state,
// validates the provided email, and attempts to link the Telegram ID with the email.
// If successful, it sends a confirmation message; otherwise, it handles various error cases
// such as already linked accounts or user not found, providing appropriate feedback to the user.
func (b *Bot) textHandler(ctx telebot.Context) error {
	userID := ctx.Sender().ID
	state, ok := userStates[userID]
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if !ok || state != stateAwaitingEmail {
		b.metrics.SentMessages.WithLabelValues("reply").Inc()
		return ctx.Reply("🐒 Use buttons, my little monkeys. Who did I make them for?")
	}

	email := ctx.Text()
	b.log.Debug("User is trying to authenticate", "user", userID, "email", email)

	startTime := time.Now()
	err := b.repo.LinkTelegramIDByEmail(timeoutCtx, userID, email)
	b.metrics.DBQueryDuration.WithLabelValues("link_telegram_id").Observe(time.Since(startTime).Seconds())
	if err != nil {
		if errors.Is(err, repository.ErrUserAlreadyLinked) {
			b.log.Info("User already linked to another id", "user", userID, "email", email)
			_ = ctx.Bot().React(ctx.Recipient(), ctx.Message(), react.React(react.ThumbDown))
			b.metrics.SentMessages.WithLabelValues("reaction").Inc()
			b.metrics.SentMessages.WithLabelValues("user_error").Inc()
			return ctx.Send("❌User already linked to other telegram account. Log out from other account and try again.")
		}
		if errors.Is(err, repository.ErrIDExists) {
			b.log.Info("User already has connection with another employee", "user", userID, "email", email)
			b.metrics.SentMessages.WithLabelValues("reaction").Inc()
			b.metrics.SentMessages.WithLabelValues("user_error").Inc()
			_ = ctx.Bot().React(ctx.Recipient(), ctx.Message(), react.React(react.ThumbDown))
			return ctx.Send(
				"❌ This telegram ID already linked to other user. Log out from other account and try again.",
			)
		}
		if errors.Is(err, repository.ErrUserNotFound) {
			b.log.Info("User with this email not found", "user", userID, "email", email)
			b.metrics.SentMessages.WithLabelValues("reaction").Inc()
			b.metrics.SentMessages.WithLabelValues("user_error").Inc()
			_ = ctx.Bot().React(ctx.Recipient(), ctx.Message(), react.React(react.ThumbDown))
			return ctx.Send("❌ User with this email not found. Try again:")
		}
		b.log.Error("Failed to link telegram id with employee", "error", err)
		b.metrics.SentMessages.WithLabelValues("error").Inc()
		return ctx.Send(ErrInternal)
	}

	delete(userStates, userID)
	b.log.Info("User successfully authenticated", "user", userID, "email", email)
	b.metrics.SentMessages.WithLabelValues("reaction").Inc()
	b.metrics.SentMessages.WithLabelValues("text").Inc()
	_ = ctx.Bot().React(ctx.Recipient(), ctx.Message(), react.React(react.ThumbUp))
	return ctx.Send("✅ Authentication successful!", authMenu)
}
