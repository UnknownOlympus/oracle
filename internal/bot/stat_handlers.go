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
func (b *Bot) statistic(ctx telebot.Context) error {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	menu := b.buildStatMenu(timeoutCtx, ctx)
	return ctx.Send(b.t(timeoutCtx, ctx, "statistic.title"), menu)
}

// statisticHandlerToday handles the request for today's statistics from the user.
// It logs the user's request, generates the statistics string for the current day,
// and sends the response back to the user.
func (b *Bot) statisticHandlerToday(ctx telebot.Context) error {
	b.metrics.CommandReceived.WithLabelValues("statistic").Inc()

	userID := ctx.Sender().ID

	b.log.Info("User requested stats", "user", userID, "period", "day")

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	responseText := b.processStatistic(timeoutCtx, ctx, userID, "day")

	return ctx.Send(responseText, telebot.ModeMarkdown)
}

// statisticHandlerMonth handles the user's request for monthly statistics.
// It logs the request, calculates the start and end dates for the current month,
// generates the statistics string, and sends the response back to the user.
func (b *Bot) statisticHandlerMonth(ctx telebot.Context) error {
	b.metrics.CommandReceived.WithLabelValues("statistic").Inc()

	userID := ctx.Sender().ID

	b.log.Info("User requested stats", "user", userID, "period", "month")

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	responseText := b.processStatistic(timeoutCtx, ctx, userID, "month")

	return ctx.Send(responseText, telebot.ModeMarkdown)
}

// statisticHandlerYear handles the statistics request for the year.
// It logs the user's request, calculates the start and end dates for the current year,
// generates the statistics string, and sends the response back to the user.
func (b *Bot) statisticHandlerYear(ctx telebot.Context) error {
	b.metrics.CommandReceived.WithLabelValues("statistic").Inc()

	userID := ctx.Sender().ID

	b.log.Info("User requested stats", "user", userID, "period", "year")

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	responseText := b.processStatistic(timeoutCtx, ctx, userID, "year")

	return ctx.Send(responseText, telebot.ModeMarkdown)
}

// processStatistic handles the request for statistics from the user.
// It logs the user's request, generates the statistics string for the period time,
// and sends the response back to the user. In case of an error during the
// generation of the statistics, it sends an internal error message.
func (b *Bot) processStatistic(ctx context.Context, bCtx telebot.Context, userID int64, period string) string {
	// --- 1. Create a unique cache key ---
	// The key includes the user ID and the period to keep it unique.
	cacheKey := fmt.Sprintf("oracle:statistic:%d:%s", userID, period)
	const cacheTTL = 1 * time.Hour // Statistics can be cached for a few hours

	// --- 2. Try to get the statistics from Redis first ---
	cachedStats, err := b.redisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		// Cache HIT!
		b.log.InfoContext(ctx, "Statistics found in cache", "user", userID, "key", cacheKey)
		b.metrics.SentMessages.WithLabelValues("text_cached").Inc()
		return cachedStats
	}

	// --- 3. Cache MISS - Calculate date range ---
	var from, to time.Time
	now := time.Now()

	switch period {
	case "day":
		from = now
		to = now
	case "month":
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		to = now
	case "year":
		from = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		to = now
	default:
		return "Unsupported period."
	}

	// --- 4. Generate the statistics string ---
	startTime := time.Now()
	responseText, err := generateStatisticString(b, bCtx, userID, from, to)
	b.metrics.DBQueryDuration.WithLabelValues("get_task_summary").Observe(time.Since(startTime).Seconds())
	if err != nil {
		b.metrics.SentMessages.WithLabelValues("error").Inc()
		return ErrInternal
	}

	// --- 5. Save the result to Redis ---
	err = b.redisClient.Set(ctx, cacheKey, responseText, cacheTTL).Err()
	if err != nil {
		// Just log the error, don't block the user
		b.log.ErrorContext(ctx, "Failed to save statistics to cache", "error", err, "key", cacheKey)
	}

	// --- 6. Send the response ---
	b.metrics.SentMessages.WithLabelValues("text").Inc()
	return responseText
}

// backHandler handles the event when a user returns to the bot.
// It sends a welcome back message along with the authentication menu.
func (b *Bot) backHandler(ctx telebot.Context) error {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	userID := ctx.Sender().ID
	isAdmin, err := b.usrepo.IsAdmin(timeoutCtx, userID)
	if err != nil {
		b.log.ErrorContext(timeoutCtx, "Failed to check admin status", "error", err)
		isAdmin = false
	}

	menu := b.buildAuthMenuWithTranslations(timeoutCtx, ctx, isAdmin)
	return ctx.Send(b.t(timeoutCtx, ctx, "general.welcome_back"), menu)
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
func generateStatisticString(bot *Bot, bCtx telebot.Context, userID int64, startDate, endDate time.Time) (string, error) {
	var builder strings.Builder

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	summaries, err := bot.tarepo.GetTaskSummary(timeoutCtx, userID, startDate, endDate)
	if err != nil {
		return "", fmt.Errorf("failed to get task summary: %w", err)
	}

	builder.WriteString(bot.t(timeoutCtx, bCtx, "statistic.your_stats"))
	builder.WriteString("\n\n")

	for _, summary := range summaries {
		if summary.Type == "Total" {
			builder.WriteString(fmt.Sprintf("\n👑 %s: %d\n", summary.Type, summary.Count))
		} else {
			builder.WriteString(fmt.Sprintf(" • %s: %d\n", summary.Type, summary.Count))
		}
	}

	encouragementPhrases := []string{
		bot.t(timeoutCtx, bCtx, "statistic.phrase.1"),
		bot.t(timeoutCtx, bCtx, "statistic.phrase.2"),
		bot.t(timeoutCtx, bCtx, "statistic.phrase.3"),
		bot.t(timeoutCtx, bCtx, "statistic.phrase.4"),
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
