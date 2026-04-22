# Tesseract

GPU-accelerated LLM desktop app. Chat, train, and serve models locally. Zero cloud, zero Python.

Powered by the [ai](https://github.com/open-ai-org/ai) CLI and [mongoose](https://github.com/open-ai-org/mongoose) GPU engine.

## Install

Download from [Releases](https://github.com/open-ai-org/tesseract/releases):

- **macOS**: `Tesseract.app` (Apple Silicon + Intel)
- **Linux**: `tesseract-linux-amd64`
- **Windows**: `tesseract-windows-amd64.exe`

Or build from source:

```bash
wails build
```

## Features

- **Chat** with any downloaded model
- **Slash commands** — type `/` for the command palette:
  - `/pull <org/model>` — download from HuggingFace
  - `/models` — list downloaded models
  - `/load <model>` — switch active model
  - `/train data=<file>` — train from scratch
  - `/gpus` — detect hardware
  - `/status` — server status
  - `/help` — all commands
- **System tray** — minimize to menu bar (macOS)
- **Auto-setup** — downloads the `ai` binary on first launch if not installed
- **GPU auto-detect** — Metal on macOS, CUDA on Linux, Vulkan/WebGPU everywhere else

## Architecture

```
Tesseract (Wails app)
  └── ai serve --daemon (inference server)
        └── mongoose (GPU compute engine)
              └── Metal / CUDA / Vulkan / CPU
```

The desktop app manages an `ai serve` daemon process that runs the OpenAI-compatible inference API on `localhost:11435`. The frontend talks to the Go backend via Wails bindings, which proxies to the serve API.

## Requirements

- macOS 13+ (Metal) / Linux (CUDA or Vulkan) / Windows (Vulkan)
- No Go, Python, or CUDA toolkit needed — the app downloads everything on first run

## Development

```bash
# Live development with hot reload
wails dev

# Production build
wails build

# Generate Go → JS bindings
wails generate module
```

## License

MIT
