package bot

import (
	"context"
	"log/slog"

	"gopkg.in/telebot.v4"
)

// AuthMiddleware check if Telegram ID is linked to permitted user.
func (b *Bot) AuthMiddleware(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(ctx telebot.Context) error {
		userID := ctx.Sender().ID

		b.log.With(
			slog.String("op", "Bot.AuthMiddleware"),
		)

		isAllowed, err := b.repo.IsUserAuthenticated(context.Background(), userID)
		if err != nil {
			b.log.Error("Failed to authenticate telegram user from DB", "id", userID, "error", err)
			_ = ctx.Send("Access verification error.")
		}

		if !isAllowed {
			b.log.Info("Access denied", "username", ctx.Sender().Username, "id", userID)
			if ctx.Callback() != nil {
				_ = ctx.Respond(&telebot.CallbackResponse{
					Text:      "Access denied. Please log in.",
					ShowAlert: true,
				})
			} else {
				_ = ctx.Send("Access to this function is denied. Please log in via /start.")
			}
			return nil
		}

		b.log.Info("Access granted", "username", ctx.Sender().Username, "id", userID)
		return next(ctx)
	}
}
