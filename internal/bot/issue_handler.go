package bot

import (
	"context"
	"fmt"
	"time"

	"gopkg.in/telebot.v4"
)

// reportIssueHandler handles the request to report a bug or feature.
// It displays information about how to submit issues on GitHub with a direct link.
func (b *Bot) reportIssueHandler(ctx telebot.Context) error {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	b.log.Info("User requested issue reporting info", "user", ctx.Sender().ID)
	b.metrics.CommandReceived.WithLabelValues("report_issue").Inc()

	// Build the message with title and description
	title := b.t(timeoutCtx, ctx, "issue.title")
	description := b.t(timeoutCtx, ctx, "issue.description")
	message := fmt.Sprintf("%s\n\n%s", title, description)

	b.metrics.SentMessages.WithLabelValues("text").Inc()
	return ctx.Send(message, telebot.ModeMarkdown)
}
