# voxtral-dictate

System-wide speech-to-text dictation for Linux (i3/sway/X11/Wayland).

A single Go binary acts as both the long-running daemon and the toggle client.
The daemon idles with near-zero CPU until toggled, then captures mic audio,
streams it to a Voxtral STT backend (local or cloud), and injects transcribed
text as keystrokes into the focused window.

## Quick Start

```bash
go build -o dictate .
cp config.example.toml ~/.config/dictate/config.toml
# edit config to choose backend + typing method

# Start daemon
./dictate daemon &

# Toggle dictation on/off
./dictate toggle

# i3 config:
#   bindsym $mod+d exec ~/dictate toggle
```

## Backends

- **mistral-realtime** — Mistral API WebSocket streaming ($0.006/min)
- **vllm-realtime** — Local vLLM serving Voxtral Realtime 4B
- **llamacpp** — Local llama.cpp with Voxtral Mini 3B GGUF (chunk-based)

## Typing Methods

- **xdotool** — X11 (default for i3)
- **ydotool** — Universal (X11 + Wayland), needs ydotoold
- **wtype** — Wayland wlroots (Sway, Hyprland)
- **dotool** — Universal alternative
