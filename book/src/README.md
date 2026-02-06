# voxtral-dictate

System-wide speech-to-text dictation for Linux. A single Go binary runs as a
long-running daemon (near-zero idle CPU) and a toggle client. Press a hotkey,
speak, press again to stop. Text appears wherever your cursor is.

Built for i3/Sway/X11/Wayland. Supports Mistral's Voxtral models — both local
(llama.cpp, vLLM) and cloud (Mistral API).

**[GitHub Repository](https://github.com/jmandel-via-jules/voxtral-dictate)**

## Quick Start

```bash
# Build
git clone https://github.com/jmandel-via-jules/voxtral-dictate.git
cd voxtral-dictate
go build -o dictate .

# Configure
mkdir -p ~/.config/dictate
cp config.example.toml ~/.config/dictate/config.toml
# Edit config.toml — set your backend and typing method

# Start daemon
./dictate daemon &

# Toggle with i3:
# bindsym $mod+d exec /path/to/dictate toggle
```

## Architecture

```
i3 keybinding
    │
    ▼
┌──────────┐   Unix socket    ┌─────────────────────────────────┐
│ dictate   │──── toggle ────▶│ dictate daemon (always running)  │
│ toggle    │◀── started ─────│                                  │
└──────────┘                  │  ┌──────────┐                    │
                              │  │ Recorder │ pw-record/arecord  │
                              │  │ (PCM)    │ subprocess pipe    │
                              │  └────┬─────┘                    │
                              │       │ []byte chunks            │
                              │  ┌────▼──────┐                   │
                              │  │ Backend   │ WebSocket / HTTP  │
                              │  │ (STT)     │ to model server   │
                              │  └────┬──────┘                   │
                              │       │ text fragments           │
                              │  ┌────▼──────┐                   │
                              │  │ Typist    │ xdotool/wtype/etc │
                              │  │ (inject)  │ into focused app  │
                              │  └───────────┘                   │
                              └─────────────────────────────────┘
```

## Backends

| Backend | Latency | Needs | Cost |
|---|---|---|---|
| `mistral-realtime` | <500ms streaming | Internet + API key | $0.006/min |
| `mistral-batch` | 2-5s per chunk | Internet + API key | $0.003/min |
| `vllm-realtime` | <500ms streaming | Local GPU ≥16GB | Free |
| `llamacpp` | 2-5s per chunk | Local GPU or CPU | Free |
| `mock` | Instant (fake) | Nothing | For testing |

## Model Options

| Model | Size | VRAM (Q4) | Quality | Best For |
|---|---|---|---|---|
| Voxtral Mini 3B | 2.5GB Q4_K_M | ~4GB | Good | Desktop, any GPU |
| Voxtral Small 24B | 14GB Q4_K_M | ~16GB | Excellent | Best accuracy |
| Voxtral Realtime 4B | ~8GB bf16 | ~16GB | Good + streaming | Lowest latency |

See **[Model Reference](models.md)** for full quant matrix and download links.

## Text Injection Methods

| Method | Works on | Notes |
|---|---|---|
| `xdotool` | X11 (i3, etc.) | Most reliable for X11. Default. |
| `ydotool` | X11 + Wayland | Universal. Needs `ydotoold` daemon. |
| `wtype` | Wayland (wlroots) | Sway, Hyprland only. |
| `dotool` | X11 + Wayland | Universal alternative. |

Set in `~/.config/dictate/config.toml` under `[typing]`.
