package bot

import (
	"context"
	"time"

	"gopkg.in/telebot.v4"
)

// languageHandler handles the language selection request from the user.
// It presents the user with a menu to choose their preferred language.
func (b *Bot) languageHandler(ctx telebot.Context) error {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	menu := &telebot.ReplyMarkup{}
	menu.Inline(
		menu.Row(menu.Data(b.t(timeoutCtx, ctx, "language.button.english"), "language_en")),
		menu.Row(menu.Data(b.t(timeoutCtx, ctx, "language.button.ukrainian"), "language_uk")),
	)

	b.metrics.SentMessages.WithLabelValues("text").Inc()
	return ctx.Send(b.t(timeoutCtx, ctx, "language.select"), menu)
}

// languageChangeHandler handles the language change request from the user.
// It updates the user's language preference in the database and sends a confirmation message.
func (b *Bot) languageChangeHandler(ctx telebot.Context) error {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	userID := ctx.Sender().ID
	callbackData := ctx.Callback().Unique
	b.log.DebugContext(timeoutCtx, "User selected language", "callbackData", callbackData, "userID", userID)

	var langCode string
	switch callbackData {
	case "language_en":
		langCode = "en"
	case "language_uk":
		langCode = "uk"
	default:
		b.log.Error("Unknown language callback", "data", callbackData)
		return ctx.Respond(&telebot.CallbackResponse{Text: "Unknown language"})
	}

	startTime := time.Now()
	err := b.usrepo.SetUserLanguage(timeoutCtx, userID, langCode)
	b.metrics.DBQueryDuration.WithLabelValues("set_user_language").Observe(time.Since(startTime).Seconds())
	if err != nil {
		b.log.ErrorContext(timeoutCtx, "Failed to set user language", "error", err, "userID", userID)
		b.metrics.SentMessages.WithLabelValues("error").Inc()
		return ctx.Respond(&telebot.CallbackResponse{Text: b.t(timeoutCtx, ctx, "error.internal")})
	}

	b.log.InfoContext(timeoutCtx, "User changed language", "userID", userID, "language", langCode)

	// Clear cache to force menu regeneration with new language
	isAdmin, err := b.usrepo.IsAdmin(timeoutCtx, userID)
	if err != nil {
		b.log.ErrorContext(timeoutCtx, "Failed to check admin status", "error", err)
		isAdmin = false
	}

	// Build menu with new language
	menu := b.buildAuthMenuWithTranslations(timeoutCtx, ctx, isAdmin)

	b.metrics.SentMessages.WithLabelValues("respond").Inc()
	_ = ctx.Respond(&telebot.CallbackResponse{Text: "✅"})

	b.metrics.SentMessages.WithLabelValues("edit").Inc()
	return ctx.Send(b.t(timeoutCtx, ctx, "language.changed"), menu)
}
