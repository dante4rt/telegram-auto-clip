package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"telegram-auto-clip/internal/bot"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	geminiKey := os.Getenv("GEMINI_API_KEY")

	if telegramToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is required")
	}
	if geminiKey == "" {
		log.Fatal("GEMINI_API_KEY is required")
	}

	b, err := bot.New(telegramToken, geminiKey, "tmp")
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Shutting down...")
		b.Stop()
		os.Exit(0)
	}()

	b.Start()
}
