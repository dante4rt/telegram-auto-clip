package bot

import (
	"fmt"
	"log"
	"strings"
	"telegram-auto-clip/internal/clipper"
	"telegram-auto-clip/internal/youtube"

	tele "gopkg.in/telebot.v3"
)

type Bot struct {
	bot     *tele.Bot
	clipper *clipper.Clipper
}

func New(token, geminiKey, outputDir string) (*Bot, error) {
	pref := tele.Settings{
		Token: token,
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	clip, err := clipper.New(geminiKey, outputDir)
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

	log.Println("Bot started...")
	b.bot.Start()
}

func (b *Bot) Stop() {
	b.clipper.Close()
	b.bot.Stop()
}

func (b *Bot) handleStart(c tele.Context) error {
	return c.Send("Welcome to Auto Clipper Bot!\n\nSend /clip <youtube_url> to create a viral clip.")
}

func (b *Bot) handleHelp(c tele.Context) error {
	help := `Auto Clipper Bot

Commands:
/clip <url> - Create a 60s vertical clip from YouTube video

Example:
/clip https://youtube.com/watch?v=xxxxx

The bot will:
1. Find the most engaging segment
2. Convert to vertical format (9:16)
3. Generate AI caption and hashtags
4. Send the clip back to you`
	return c.Send(help)
}

func (b *Bot) handleClip(c tele.Context) error {
	args := strings.TrimSpace(c.Message().Payload)
	if args == "" {
		return c.Send("Usage: /clip <youtube_url>\n\nExample: /clip https://youtube.com/watch?v=xxxxx")
	}

	url := strings.Fields(args)[0]
	if !youtube.IsValidYouTubeURL(url) {
		return c.Send("Invalid YouTube URL. Please provide a valid YouTube link.")
	}

	// Send initial status
	statusMsg, err := b.bot.Send(c.Chat(), "Processing...")
	if err != nil {
		return err
	}

	// Process the clip
	result, err := b.clipper.Process(url, func(status string) {
		b.bot.Edit(statusMsg, status)
	})
	if err != nil {
		b.bot.Edit(statusMsg, fmt.Sprintf("Error: %v", err))
		return nil
	}
	defer b.clipper.Cleanup(result)

	// Build caption
	caption := fmt.Sprintf("%s\n\n%s\n\n---\nTitle: %s\nDuration: %s\nPlatform: %s\nChannel: %s",
		result.Caption,
		result.Hashtags,
		result.Title,
		result.Duration,
		result.Platform,
		result.Channel,
	)

	// Send video
	video := &tele.Video{
		File:    tele.FromDisk(result.VideoPath),
		Caption: truncateCaption(caption, 1024),
	}

	if err := c.Send(video); err != nil {
		b.bot.Edit(statusMsg, fmt.Sprintf("Failed to send video: %v", err))
		return nil
	}

	// Delete status message
	b.bot.Delete(statusMsg)
	return nil
}

func truncateCaption(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
