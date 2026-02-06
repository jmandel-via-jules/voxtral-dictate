# Linux Dictation Landscape

*Research snapshot from February 2026*

This doc captures the state of system-wide speech-to-text dictation on Linux
so we don't have to re-research it if we need to change approaches.

## Existing Tools

| Tool | Stars | Backend | Wayland | Pluggable |
|---|---|---|---|---|
| **nerd-dictation** | 1,744 | VOSK | ❌ (xdotool) | Hackable (1 Python file) |
| **hyprwhspr** | 791 | Parakeet/Whisper/REST/WebSocket | ✅ | ✅ REST API accepts any URL |
| **Speech Note** | 1,345 | Whisper/VOSK/DeepSpeech | ✅ | ✅ Multiple built-in |
| **voxtype** | 308 | whisper.cpp + remote | ✅ Wayland-first | ✅ + LLM post-processing |
| **OpenWhispr** | 1,043 | Whisper + Parakeet | ✅ | ✅ |
| **keyless** | 18 | Whisper (Candle GGUF) | ✅ | Limited |

## Text Injection Methods

| Tool | X11 | Wayland wlroots | Wayland GNOME | Mechanism |
|---|---|---|---|---|
| **xdotool** | ✅ | ❌ | ❌ | X11 XTest extension |
| **ydotool** | ✅ | ✅ | ✅ | Linux uinput (kernel-level) |
| **dotool** | ✅ | ✅ | ✅ | Linux uinput |
| **wtype** | ❌ | ✅ | ❌ | Wayland virtual-keyboard protocol |
| **clipboard+paste** | ✅ | ✅ | ✅ | wl-copy/xclip + Ctrl+V |

**Recommended fallback chain** (from voxtype): wtype → dotool → ydotool → clipboard

**ydotool gotchas:**
- Requires the `ydotoold` daemon running
- User must be in the `input` group
- Version 1.0+ has different CLI than older versions

**xdotool gotchas:**
- `--clearmodifiers` flag needed to avoid modifier key interference
- Unicode can be unreliable for some characters

## Audio Capture on Linux

| Tool | Audio System | Format | Notes |
|---|---|---|---|
| `pw-record` | PipeWire | Any | Modern default, recommended |
| `arecord` | ALSA | Any | Oldest, most compatible |
| `parecord` | PulseAudio | Any | Legacy |
| `ffmpeg -f pulse` | PulseAudio | Any | When you need transcoding |
| `sox rec` | ALSA/PulseAudio | Any | Swiss army knife |

PipeWire is the modern default on most distros. `pw-record` should be tried first.

## Why We Built Our Own

- No existing tool supports Voxtral as a backend
- Most tools are Python (slow startup, dependency management)
- nerd-dictation is the closest in philosophy but X11-only and VOSK-locked
- hyprwhspr has the best architecture (pluggable backends) but is Python/heavy
- We wanted a single Go binary with zero runtime dependencies

## Related Projects

- **voxtral.c** (antirez): Pure C Voxtral inference, streaming API. Could be
  integrated as a backend if we wanted to skip the HTTP server entirely.
- **wyoming protocol**: Home Assistant's STT protocol. Someone made a
  Voxtral adapter (voxtral_wyoming).
- **Open WebUI**: Has native Mistral STT integration (set AUDIO_STT_ENGINE=mistral).
