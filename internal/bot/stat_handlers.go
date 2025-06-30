package bot

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"gopkg.in/telebot.v4"
)

// statisticHandler sends a message to the user with options for statistics.
// It prompts the user to pick which statistic they want to view.
func (b *Bot) statisticHandler(ctx telebot.Context) error {
	return ctx.Send("📈 Pick statistic what do you want", statMenu)
}

// statisticHandlerToday handles the request for today's statistics from the user.
// It logs the user's request, generates the statistics string for the current day,
// and sends the response back to the user. In case of an error during the
// generation of the statistics, it sends an internal error message.
func (b *Bot) statisticHandlerToday(ctx telebot.Context) error {
	b.log.Info("User requested stats", "user", ctx.Sender().ID, "duration", "day")
	endDate := time.Now()

	responseText, err := generateStatisticString(b, ctx.Sender().ID, endDate, endDate)
	if err != nil {
		return ctx.Send(ErrInternal)
	}

	return ctx.Send(responseText, telebot.ModeMarkdown)
}

// statisticHandlerMonth handles the user's request for monthly statistics.
// It logs the request, calculates the start and end dates for the current month,
// generates the statistics string, and sends the response back to the user.
// If an error occurs during the generation of the statistics, it sends an internal error message.
func (b *Bot) statisticHandlerMonth(ctx telebot.Context) error {
	b.log.Info("User requested stats", "user", ctx.Sender().ID, "duration", "month")
	endDate := time.Now()
	startDate := time.Date(endDate.Year(), endDate.Month(), 1, 0, 0, 0, 0, endDate.Location())

	responseText, err := generateStatisticString(b, ctx.Sender().ID, startDate, endDate)
	if err != nil {
		return ctx.Send(ErrInternal)
	}

	return ctx.Send(responseText, telebot.ModeMarkdown)
}

// statisticHandlerYear handles the statistics request for the year.
// It logs the user's request, calculates the start and end dates for the current year,
// generates the statistics string, and sends the response back to the user.
// If an error occurs during the generation of the statistics string, it sends an internal error message.
func (b *Bot) statisticHandlerYear(ctx telebot.Context) error {
	b.log.Info("User requested stats", "user", ctx.Sender().ID, "duration", "year")
	endDate := time.Now()
	startDate := time.Date(endDate.Year(), time.January, 1, 0, 0, 0, 0, endDate.Location())

	responseText, err := generateStatisticString(b, ctx.Sender().ID, startDate, endDate)
	if err != nil {
		return ctx.Send(ErrInternal)
	}

	return ctx.Send(responseText, telebot.ModeMarkdown)
}

// backHandler handles the event when a user returns to the bot.
// It sends a welcome back message along with the authentication menu.
func (b *Bot) backHandler(ctx telebot.Context) error {
	return ctx.Send("🤖 Welcome back", authMenu)
}

// generateStatisticString generates a formatted string containing statistics for a user
// within a specified date range. It retrieves task summaries from the bot's repository,
// formats them into a human-readable string, and appends a random encouragement phrase.
//
// Parameters:
// - bot: A pointer to the Bot instance used to access the repository.
// - userID: The ID of the user for whom the statistics are generated.
// - startDate: The start date for the statistics period.
// - endDate: The end date for the statistics period.
//
// Returns:
// - A formatted string containing the user's statistics and a random encouragement phrase.
// - An error if the task summary retrieval fails.
func generateStatisticString(bot *Bot, userID int64, startDate, endDate time.Time) (string, error) {
	var builder strings.Builder

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	summaries, err := bot.repo.GetTaskSummary(timeoutCtx, userID, startDate, endDate)
	if err != nil {
		return "", fmt.Errorf("failed to get task summary: %w", err)
	}

	builder.WriteString("🐘 *Your stats*:\n\n")

	for _, summary := range summaries {
		if summary.Type == "Total" {
			builder.WriteString(fmt.Sprintf("\n👑 %s: %d\n", summary.Type, summary.Count))
		} else {
			builder.WriteString(fmt.Sprintf(" • %s: %d\n", summary.Type, summary.Count))
		}
	}

	encouragementPhrases := []string{
		"_Well, you tried!_",
		"_They pay pennies for repairs\n\t(c) Confucius_",
		"_Maybe you could do better, but as it is_",
		"_If you want more repairs, find the nearest box and fuck it up_",
	}

	randomIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(encouragementPhrases))))
	if err != nil {
		return "", fmt.Errorf("failed to generate random integer: %w", err)
	}
	randomPhrase := encouragementPhrases[randomIndex.Int64()]

	builder.WriteString("\n\\*\\*\\*\n")
	builder.WriteString(randomPhrase)

	return builder.String(), err
}
