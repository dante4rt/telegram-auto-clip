package main

import (
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"telegram-auto-clip/internal/bot"
	"telegram-auto-clip/internal/config"
	"telegram-auto-clip/internal/logger"
	"telegram-auto-clip/internal/proxy"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		logger.Info("No .env file found, using environment variables")
	}

	proxy.Init()
	if proxy.Count() > 0 {
		logger.Info("Loaded %d proxies", proxy.Count())
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

	adminID, _ := strconv.ParseInt(os.Getenv("ADMIN_USER_ID"), 10, 64)
	if adminID != 0 {
		logger.Info("Admin user: %d", adminID)
	}

	b, err := bot.New(telegramToken, geminiKey, cfg, adminID)
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
