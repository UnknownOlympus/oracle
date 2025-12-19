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

	b.metrics.SentMessages.WithLabelValues("respond").Inc()
	_ = ctx.Respond(&telebot.CallbackResponse{Text: "✅"})

	// Verify the language was actually changed
	newLang, err := b.usrepo.GetUserLanguage(timeoutCtx, userID)
	if err != nil {
		b.log.ErrorContext(timeoutCtx, "Failed to verify language change", "error", err, "userID", userID)
		b.metrics.SentMessages.WithLabelValues("error").Inc()
		return ctx.Send(b.localizer.Get("en", "error.internal"))
	}

	b.log.InfoContext(
		timeoutCtx,
		"Language verified after change",
		"userID",
		userID,
		"newLang",
		newLang,
		"expected",
		langCode,
	)

	// Build menu with new language
	menu := b.menuBuilder.Build(timeoutCtx, ctx, MenuMore, userID)

	// Get confirmation message in new language
	confirmMsg := b.localizer.Get(langCode, "language.changed")

	b.log.InfoContext(timeoutCtx, "Sending menu in new language", "userID", userID, "language", langCode)

	b.metrics.SentMessages.WithLabelValues("text").Inc()
	return ctx.Send(confirmMsg, menu)
}
