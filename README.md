# Telegram Auto Clipper

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Telegram Bot](https://img.shields.io/badge/Telegram-Bot-26A5E4?logo=telegram)](https://core.telegram.org/bots)

A Telegram bot that automatically creates viral-ready clips from YouTube videos using AI.

## Features

- **Smart Segment Detection** - Uses YouTube heatmap data or Gemini AI to find the most engaging moments
- **Dynamic Duration** - Clips are 15-60 seconds based on content type
- **AI Captions** - Generates catchy captions in Bahasa Indonesia with relevant hashtags
- **Concurrent Processing** - Handles multiple requests simultaneously

## Prerequisites

- Go 1.24+
- [yt-dlp](https://github.com/yt-dlp/yt-dlp)
- [FFmpeg](https://ffmpeg.org/)

```bash
# macOS
brew install yt-dlp ffmpeg

# Linux
sudo apt install ffmpeg && pip install yt-dlp
```

## Quick Start

1. Clone the repo

   ```bash
   git clone https://github.com/dante4rt/telegram-auto-clip.git
   cd telegram-auto-clip
   ```

2. Configure environment

   ```bash
   cp .env.example .env
   ```

   Edit `.env`:

   ```text
   TELEGRAM_BOT_TOKEN=your_token    # Get from @BotFather
   GEMINI_API_KEY=your_key          # Get from aistudio.google.com/apikey
   ```

3. Run

   ```bash
   go run main.go
   ```

## Usage

Send to your bot in Telegram:

```text
/clip https://youtube.com/watch?v=VIDEO_ID
```

The bot will:

1. Analyze the video to find the best moment
2. Download and process the segment
3. Generate an AI caption
4. Send the clip back

## Contributing

Contributions are welcome! Feel free to:

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Gemini AI](https://ai.google.dev/) for video analysis and caption generation
- [yt-dlp](https://github.com/yt-dlp/yt-dlp) for YouTube downloading
- [telebot](https://github.com/tucnak/telebot) for Telegram bot framework
