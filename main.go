package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/carlosbelchior/betano-bot/internal/green365"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// 1. Configure Logging
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	// 2. Load Environment Variables
	if err := godotenv.Load(); err != nil {
		log.Warn().Msg(".env file not found, trying system environment variables")
	}

	config := struct {
		Email    string
		Password string
		BotID    string
	}{
		Email:    os.Getenv("GREEN365_EMAIL"),
		Password: os.Getenv("GREEN365_PASSWORD"),
		BotID:    os.Getenv("GREEN365_BOT_ID"),
	}

	if config.Email == "" || config.Password == "" || config.BotID == "" {
		log.Fatal().Msg("GREEN365_EMAIL, GREEN365_PASSWORD and GREEN365_BOT_ID must be set")
	}

	// 3. Graceful Shutdown Setup
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 4. Initialize Bot
	bot := green365.NewBot(config.Email, config.Password, config.BotID)

	// 5. Run Bot
	go func() {
		if err := bot.Run(ctx); err != nil {
			log.Fatal().Err(err).Msg("Bot application failed critically")
		}
	}()

	log.Info().Msg("Bot is running. Press CTRL+C to stop.")

	// Wait for interruption
	<-ctx.Done()
	log.Info().Msg("Shutdown signal received. Cleaning up...")
	bot.Stop()
	log.Info().Msg("Bot successfully stopped.")
}
