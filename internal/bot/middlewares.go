package bot

import (
	"context"
	"log/slog"
	"time"

	"gopkg.in/telebot.v4"
)

// AuthMiddleware check if Telegram ID is linked to permitted user.
func (b *Bot) AuthMiddleware(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(ctx telebot.Context) error {
		userID := ctx.Sender().ID

		b.log.With(
			slog.String("op", "Bot.AuthMiddleware"),
		)

		startTime := time.Now()
		isAllowed, err := b.repo.IsUserAuthenticated(context.Background(), userID)
		b.metrics.DBQueryDuration.WithLabelValues("is_user_authenticated").Observe(time.Since(startTime).Seconds())
		if err != nil {
			b.log.Error("Failed to authenticate telegram user from DB", "id", userID, "error", err)
			b.metrics.SentMessages.WithLabelValues("text").Inc()
			_ = ctx.Send("Access verification error.")
		}

		if !isAllowed {
			b.log.Info("Access denied", "username", ctx.Sender().Username, "id", userID)
			if ctx.Callback() != nil {
				b.metrics.SentMessages.WithLabelValues("respond").Inc()
				_ = ctx.Respond(&telebot.CallbackResponse{
					Text:      "Access denied. Please log in.",
					ShowAlert: true,
				})
			} else {
				b.metrics.SentMessages.WithLabelValues("text").Inc()
				_ = ctx.Send("Access to this function is denied. Please log in via /start.")
			}
			return nil
		}

		b.log.Debug("Access granted", "username", ctx.Sender().Username, "id", userID)
		return next(ctx)
	}
}
