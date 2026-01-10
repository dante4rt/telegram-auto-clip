package main

import (
	"os"
	"os/signal"
	"syscall"

	"telegram-auto-clip/internal/bot"
	"telegram-auto-clip/internal/config"
	"telegram-auto-clip/internal/logger"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		logger.Info("No .env file found, using environment variables")
	}

	cfg, _ := config.Load("config.json")

	telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	geminiKey := os.Getenv("GEMINI_API_KEY")

	if telegramToken == "" {
		logger.Error("TELEGRAM_BOT_TOKEN is required")
		os.Exit(1)
	}
	if geminiKey == "" {
		logger.Error("GEMINI_API_KEY is required")
		os.Exit(1)
	}

	logger.Info("Token loaded: %s...", telegramToken[:20])
	logger.Info("Creating bot...")

	b, err := bot.New(telegramToken, geminiKey, cfg)
	if err != nil {
		logger.Error("Failed to create bot: %v", err)
		os.Exit(1)
	}

	logger.Info("Bot created successfully")

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		logger.Info("Shutting down...")
		b.Stop()
		os.Exit(0)
	}()

	b.Start()
}
