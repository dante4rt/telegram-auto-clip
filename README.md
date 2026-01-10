# Telegram Auto Clipper

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Telegram Bot](https://img.shields.io/badge/Telegram-Bot-26A5E4?logo=telegram)](https://core.telegram.org/bots)

A Telegram bot that creates viral-ready clips from YouTube videos using AI.

## Features

- **Smart Segment Detection** - Uses YouTube heatmap or Gemini AI to find engaging moments
- **Dynamic Duration** - 15-60 second clips based on content type
- **AI Captions** - Generates captions in Bahasa Indonesia with hashtags
- **Concurrent Processing** - Handles multiple requests simultaneously

## Prerequisites

- Go 1.24+
- [FFmpeg](https://ffmpeg.org/)
- [yt-dlp](https://github.com/yt-dlp/yt-dlp)

```bash
# macOS
brew install ffmpeg yt-dlp

# Linux
sudo apt install ffmpeg
pip install yt-dlp
```

## Quick Start

1. **Clone & setup**

   ```bash
   git clone https://github.com/dante4rt/telegram-auto-clip.git
   cd telegram-auto-clip
   cp .env.example .env
   ```

2. **Edit `.env`**

   ```env
   TELEGRAM_BOT_TOKEN=your_token    # From @BotFather
   GEMINI_API_KEY=your_key          # From aistudio.google.com/apikey
   ```

3. **Run**

   ```bash
   go run main.go
   ```

## Usage

```text
/clip https://youtube.com/watch?v=VIDEO_ID
```

## Configuration

Edit `config.json` to customize (all optional):

| Option                       | Description                          | Default |
| ---------------------------- | ------------------------------------ | ------- |
| `max_clip_duration_sec`      | Maximum clip length                  | 60      |
| `min_heatmap_score`          | Minimum engagement score (0-1)       | 0.15    |
| `max_ai_video_duration_sec`  | Max video length for AI analysis     | 1200    |
| `fallback_clip_duration_sec` | Fallback clip length                 | 45      |
| `fallback_start_percent`     | Start position for fallback          | 0.2     |
| `cookies_file`               | Path to cookies.txt for YouTube auth | ""      |

## Authentication

> [!NOTE]
> YouTube may require authentication for high-quality downloads on some servers.

For servers getting "Sign in to confirm you're not a bot" errors:

1. **Export YouTube cookies** from your browser using a cookie exporter extension
2. Save as `cookies.txt` in Netscape format
3. Set `cookies_file` in `config.json`:

   ```json
   {
     "cookies_file": "cookies.txt"
   }
   ```

> [!WARNING]
> Cookies may expire after some time. Re-export if you get authentication errors.

## Segment Selection

The bot uses multiple strategies to find the best clip:

```text
┌─────────────────────────────────────────────────────────┐
│  1. YouTube Heatmap  →  "Most Replayed" data            │
│  2. Gemini AI        →  Video analysis (if < 20 min)    │
│  3. Fallback         →  20% into video or first 45 sec  │
└─────────────────────────────────────────────────────────┘
```

> [!TIP]
> Videos with more views tend to have better heatmap data for accurate segment detection.

## Troubleshooting

> [!IMPORTANT]
> The Gemini free tier has a limit of 20 requests/day. Consider upgrading for heavy usage.

| Issue                | Solution                           |
| -------------------- | ---------------------------------- |
| 360p quality         | Export fresh YouTube cookies       |
| "Sign in to confirm" | Add cookies.txt to config          |
| Video too large      | Clips are auto-compressed to <50MB |
| Quota exceeded       | Wait 24h or upgrade Gemini plan    |

## Contributing

1. Fork the repo
2. Create feature branch (`git checkout -b feature/amazing`)
3. Commit changes (`git commit -m 'Add feature'`)
4. Push (`git push origin feature/amazing`)
5. Open a Pull Request

## License

MIT - see [LICENSE](LICENSE)

## Acknowledgments

- [Gemini AI](https://ai.google.dev/) - Video analysis & captions
- [yt-dlp](https://github.com/yt-dlp/yt-dlp) - YouTube downloading
- [telebot](https://github.com/tucnak/telebot) - Telegram bot framework
