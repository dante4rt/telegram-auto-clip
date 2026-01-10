# Telegram Auto Clipper

Telegram bot that clips YouTube videos to 60s vertical format with AI captions.

## Requirements

```bash
# macOS
brew install yt-dlp ffmpeg

# Linux
sudo apt install ffmpeg
pip install yt-dlp

# Windows
# Download ffmpeg: https://ffmpeg.org/download.html
# Download yt-dlp: https://github.com/yt-dlp/yt-dlp/releases
```

## Setup

```bash
cp .env.example .env
```

Edit `.env`:

```text
TELEGRAM_BOT_TOKEN=xxx  # Get from @BotFather
GEMINI_API_KEY=xxx      # Get from https://aistudio.google.com/apikey
```

## Run

```bash
go run main.go
```

## Usage

In Telegram:

```text
/clip https://youtube.com/watch?v=xxxxx
```

Bot will:

1. Find most engaging 60s segment (heatmap or AI)
2. Convert to vertical 9:16
3. Generate caption + hashtags
4. Send clip back

## Commands

| Command | Description |
| --------- | ------------- |
| `/clip <url>` | Create clip from YouTube video |
| `/help` | Show usage |
