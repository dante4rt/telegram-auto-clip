package bot

import (
	"fmt"
	"strings"
	"time"

	"telegram-auto-clip/internal/clipper"
	"telegram-auto-clip/internal/config"
	"telegram-auto-clip/internal/logger"
	"telegram-auto-clip/internal/youtube"

	tele "gopkg.in/telebot.v3"
)

type Bot struct {
	bot     *tele.Bot
	clipper *clipper.Clipper
}

func New(token, geminiKey string, cfg *config.Config) (*Bot, error) {
	pref := tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{Timeout: time.Duration(cfg.PollTimeoutSec) * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	clip, err := clipper.New(geminiKey, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create clipper: %w", err)
	}

	return &Bot{
		bot:     b,
		clipper: clip,
	}, nil
}

func (b *Bot) Start() {
	b.bot.Handle("/start", b.handleStart)
	b.bot.Handle("/help", b.handleHelp)
	b.bot.Handle("/clip", b.handleClip)

	logger.Info("Bot started, waiting for messages...")
	b.bot.Start()
}

func (b *Bot) Stop() {
	b.clipper.Close()
	b.bot.Stop()
}

func (b *Bot) handleStart(c tele.Context) error {
	msg := `Hey! Welcome to Auto Clipper Bot!

I'll help you create viral clips from YouTube videos!

Just send:
/clip <youtube_url>

Example:
/clip https://youtube.com/watch?v=xxxxx

I'll find the best moment and convert it to vertical format!`
	return c.Send(msg)
}

func (b *Bot) handleHelp(c tele.Context) error {
	help := `Auto Clipper Bot

Commands:
/clip <url> - Create a vertical clip from YouTube

What I do:
1. Find the most engaging moment
2. Convert to vertical 9:16 (full screen)
3. Generate AI caption + hashtags
4. Send the clip back!

Clip duration is dynamic (15-60 sec) based on content!`
	return c.Send(help)
}

func (b *Bot) handleClip(c tele.Context) error {
	userID := c.Sender().ID
	chat := c.Chat()
	logger.Info("Received /clip from user %d", userID)

	args := strings.TrimSpace(c.Message().Payload)
	if args == "" {
		return c.Send("Hey, where's the YouTube link?\n\nUsage: /clip <youtube_url>\n\nExample:\n/clip https://youtube.com/watch?v=xxxxx")
	}

	url := strings.Fields(args)[0]
	logger.Info("Processing URL: %s for user %d", url, userID)

	if !youtube.IsValidYouTubeURL(url) {
		return c.Send("Hmm, that doesn't look like a valid YouTube URL. Please try again!")
	}

	// Initial status
	statusMsg, err := b.bot.Send(chat, "Got it! Processing your video...")
	if err != nil {
		logger.Error("Failed to send status: %v", err)
		return err
	}

	// Process in goroutine to handle concurrent requests
	go func() {
		// Update status message (edit the same message)
		updateStatus := func(status string) {
			logger.Info("[User %d] Status: %s", userID, status)
			_, err := b.bot.Edit(statusMsg, status)
			if err != nil {
				logger.Debug("Edit failed: %v", err)
			}
		}

		result, err := b.clipper.Process(url, updateStatus)
		if err != nil {
			logger.Error("[User %d] Processing failed: %v", userID, err)
			b.bot.Edit(statusMsg, fmt.Sprintf("Oops! Error: %v\n\nPlease try again later!", err))
			return
		}
		defer b.clipper.Cleanup(result)

		logger.Info("[User %d] Clip ready: %s", userID, result.VideoPath)
		b.bot.Edit(statusMsg, "Clip ready! Uploading...")

		caption := fmt.Sprintf(`%s

%s

---
Title: %s
Duration: %s
Platform: %s
Channel: %s`,
			result.Caption,
			result.Hashtags,
			result.Title,
			result.Duration,
			result.Platform,
			result.Channel,
		)

		video := &tele.Video{
			File:    tele.FromDisk(result.VideoPath),
			Caption: truncate(caption, 1024),
		}

		if _, err := b.bot.Send(chat, video); err != nil {
			logger.Error("[User %d] Failed to send video: %v", userID, err)
			b.bot.Edit(statusMsg, fmt.Sprintf("Upload failed: %v\n\nFile might be too large, please try again!", err))
			return
		}

		b.bot.Delete(statusMsg)
		logger.Info("[User %d] Video sent successfully", userID)
	}()

	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
