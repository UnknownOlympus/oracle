package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Houeta/radireporter-bot/config"
	"github.com/Houeta/radireporter-bot/internal/bot"
	"github.com/Houeta/radireporter-bot/internal/repository"
)

// Constants for different environment types.
const (
	envLocal = "local"
	envDev   = "development"
	envProd  = "production"
)

// main is the entry point of the application.
func main() {
	// Create a context that will be canceled when an interrupt signal is received.
	// This allows for graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	// Load application configuration.
	cfg := config.MustLoad()

	// Set up the logger based on the environment.
	logger := setupLogger(cfg.Env)

	// Initialize the database connection.
	dtb, err := repository.NewDatabase(
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.Name,
	)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}

	// Create a new repository instance using the database connection.
	repo := repository.NewRepository(dtb)

	// Initialize the bot with logger, repository, token, and poller timeout.
	radiBot, err := bot.NewBot(logger, repo, cfg.Token, cfg.PollerTimeout)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}
	defer stop() // Ensure stop is called to release resources related to signal handling.
	defer dtb.Close()

	// Log that the application has started.
	logger.InfoContext(ctx, "Application started. Press Ctrl+C to stop.")

	// Start the bot in a goroutine to allow main to listen for signals.
	go func() {
		radiBot.Start()
	}()

	// Wait for the context to be canceled (e.g., by Ctrl+C).
	<-ctx.Done()

	// Log that a shutdown signal has been received.
	logger.InfoContext(ctx, "Shutdown signal received. Stopping application...")

	// Stop the bot gracefully.
	radiBot.Stop()

	// Log graceful shutdown completion.
	logger.InfoContext(ctx, "Application stopped gracefully.")
}

// setupLogger initializes and returns a logger based on the environment provided.
func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level:     slog.LevelDebug,
				AddSource: true,
				ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
					return a
				},
			}),
		)
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level:     slog.LevelInfo,
				AddSource: false,
				ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
					return a
				},
			}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level:     slog.LevelWarn,
				AddSource: false,
				ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
					if a.Key == slog.TimeKey {
						return slog.Attr{}
					}
					return a
				},
			}),
		)
	default:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level:     slog.LevelError,
				AddSource: false,
				ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
					if a.Key == slog.TimeKey {
						return slog.Attr{}
					}
					return a
				},
			}),
		)

		log.Error(
			"The env parameter was not specified	 or was invalid. Logging will be minimal, by default.",
			slog.String("available_envs", "local, development, production"))
	}

	return log
}
