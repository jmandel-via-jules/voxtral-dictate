# Design & Architecture

## Problem

We want macOS-style system-wide dictation on Linux: press a hotkey, speak,
text appears wherever the cursor is. Must work with i3/X11, optionally Wayland.
Must support both cloud APIs and fully-local inference for privacy.

## Architecture Decisions

### 1. Daemon + Toggle (Unix socket IPC)

**Decision:** Long-running daemon communicates with a tiny toggle client via
Unix domain socket.

**Why:**
- The daemon stays resident so sessions start instantly (no process startup delay)
- Unix sockets are the lowest-latency local IPC (no TCP overhead, no port conflicts)
- The toggle client is a ~1ms operation — perfect for hotkey binding
- Socket is world-writable so it works without sudo

**Alternatives considered:**
- D-Bus: heavier, more complex, overkill for on/off toggle
- Named pipe (FIFO): can't easily send response back
- Signal (SIGUSR1): no way to get response, race conditions
- HTTP: port conflicts, TCP overhead, unnecessary for localhost

### 2. Single binary (subcommands)

**Decision:** `dictate daemon` and `dictate toggle` are the same binary.

**Why:**
- One thing to build, install, distribute
- Shared config loading code
- `toggle` needs to know the socket path, which comes from the same config

### 3. Model server is external

**Decision:** The dictate daemon does NOT manage model server lifecycle.
The model server (llama.cpp, vLLM) runs as a separate process/service.

**Why:**
- Model loading takes 10-30 seconds. Must be done once at boot, not per-session.
- Model servers are general-purpose — you might use the same server for other apps
- Separation of concerns: dictate handles audio+typing, model server handles inference
- Different failure modes: model server crash shouldn't kill dictation daemon
- Systemd manages restarts, resource limits, logging for each independently

### 4. Audio capture via subprocess pipe (no CGo)

**Decision:** Capture mic audio by piping from `pw-record` (PipeWire) or
`arecord` (ALSA) as a subprocess.

**Why:**
- No CGo dependency (PortAudio bindings are fragile and complicate cross-compilation)
- pw-record/arecord handle all the ALSA/PulseAudio/PipeWire complexity
- Pipe-based: dead simple, reliable, easy to test
- Falls back automatically: tries pw-record first, then arecord

**Tradeoff:** Extra process per session. Acceptable — these are tiny processes.

### 5. Go (not Python)

**Decision:** Entire daemon and client written in Go.

**Why:**
- Single static binary, no dependency management
- Instant startup for the toggle client (~1ms)
- Proper concurrency (goroutines for audio, backend, typist)
- Low memory footprint for long-running daemon (~5MB idle)
- The actual ML inference is in the model server; this is just plumbing

**Python would have been worse because:**
- Slower startup (100-200ms) for toggle client
- venv/pip dependency management
- Higher memory footprint
- GIL complicates concurrent audio+network+typing

### 6. Backend interface

**Decision:** `Backend` interface with `Transcribe(ctx, audioCh, textCh)`.

```go
type Backend interface {
    Transcribe(ctx context.Context, audioCh <-chan []byte, textCh chan<- string) error
}
```

**Why:**
- Uniform interface for all backends (cloud streaming, cloud batch, local)
- Channel-based: natural Go concurrency pattern
- Context-based cancellation: clean session shutdown
- Text comes as fragments — backends that stream emit word-by-word,
  batch backends emit full chunks. Typist doesn't care.

### 7. WebSocket backend shared between Mistral and vLLM

**Decision:** One `WebSocketBackend` implementation serves both Mistral Realtime
API and local vLLM Realtime.

**Why:** They use the same protocol (Mistral designed it, vLLM implemented it).
Only difference is the URL and auth header.

**Protocol:** Connect to `wss://api.mistral.ai/v1/audio/transcriptions/realtime?model=...`,
wait for `session.created`, send `session.update` with `audio_format`
(encoding + sample_rate), then stream audio via `input_audio.append` messages
(base64-encoded PCM). Receive `transcription.text.delta` events with incremental
text. Send `input_audio.end` when done.

### 8. ydotoold lifecycle management

**Decision:** When `typing.method = "ydotool"`, the daemon auto-starts `ydotoold`
if it isn't already running, and kills it on shutdown (only if we started it).

**Why:** ydotool requires a running daemon (`ydotoold`) for uinput access. Rather
than requiring users to set up a separate service, the dictate daemon manages it
transparently. If ydotoold is already running (e.g. via systemd), we leave it alone.

### 9. Voice Activity Detection (VAD) — burst-based architecture

**Decision:** Energy-based VAD splits audio into speech bursts. Each burst gets
its own WebSocket connection. Silence = no connection = no billing.

**Why:**
- Mistral Realtime API bills at $0.006/min for the duration of the connection
- Streaming silence with keepalives still maintains a billable session
- Burst-based approach: connect on speech onset, disconnect after trailing silence
- Pre-buffer ring (3 chunks, ~1.4s) preserves word onsets that precede VAD trigger
- Trailing grace period (21 chunks, ~10s) avoids thrashing on natural pauses

**How it works:**
- `vadBursts()` returns a `<-chan (<-chan []byte)` — a channel of burst channels
- Each burst channel opens when speech is detected (RMS energy > threshold) and
  closes after the trailing silence period expires
- `handleBurst()` connects a backend per burst, with retry/backoff within the burst
- When VAD is disabled, a single passthrough burst covers the entire session

## Data Flow

```
Session start (toggle ON):
  1. Daemon creates context with cancel
  2. Spawns Recorder goroutine → audioCh (chan []byte)
  3. VAD splits audioCh into speech bursts (chan of chans)
  4. For each burst:
     a. Connect backend (WebSocket)
     b. Stream audio → textCh (chan string)
     c. Type text as it arrives
     d. On burst end (trailing silence), disconnect backend
  5. Wait for next burst...

Session stop (toggle OFF):
  1. Cancel context
  2. Recorder subprocess gets killed, audioCh closes
  3. VAD closes, all burst channels close
  4. Backend sees closed channel or cancelled context, disconnects
  5. Session goroutines are cleaned up
```

## Audio Format

All audio in the system is **PCM s16le, 16kHz, mono** — this is what Voxtral expects.
- 2 bytes per sample × 16000 samples/sec = 32,000 bytes/sec
- At chunk_ms=480: each chunk = 15,360 bytes
- 1 minute of audio = ~1.9MB of raw PCM

For llama.cpp which expects WAV, we wrap the PCM in a minimal 44-byte WAV header
on the fly (see `pcmToWAV()` in backend_llamacpp.go).

## Configuration Priority

1. `DICTATE_CONFIG` env var → custom config path
2. `~/.config/dictate/config.toml` → default location
3. Built-in defaults → works with zero config (if backend is mock)

API key: `MISTRAL_API_KEY` env var overrides config file `api_key` field.

## Typing Method Details

| Method | Mechanism | Latency | Unicode | Limitations |
|---|---|---|---|---|
| xdotool | X11 XTest extension | ~1ms/char | Good | X11 only, modifier key issues |
| ydotool | Linux uinput (kernel) | ~1ms/char | Good | Needs ydotoold (auto-managed) + input group |
| wtype | Wayland virtual-keyboard | ~1ms/char | Good | wlroots compositors only |
| dotool | Linux uinput | ~1ms/char | Good | Needs uinput access |

All methods type the full text fragment as received from the backend.
For streaming backends this means word-by-word; for batch backends it's
the full chunk (several seconds of speech at once).

## Future Improvements

- **Audio feedback:** Play a short beep/tone on toggle to confirm start/stop
- **i3bar/waybar integration:** ~~Show dictation status in the status bar~~
  Done — see `contrib/bumblebee-dictate.py` for bumblebee-status module
- **Streaming for llamacpp:** llama.cpp may eventually get a proper
  /v1/audio/transcriptions endpoint (GitHub issue #15291)
- **Voxtral Realtime in llama.cpp:** The Realtime 4B model currently only
  works in vLLM; community contributions to llama.cpp are welcomed
- **Auto-punctuation:** Post-process text through an LLM for punctuation/formatting
- **Per-session backend override:** Allow toggle command to specify backend
