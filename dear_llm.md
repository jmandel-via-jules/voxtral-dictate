# LLM Guidance for voxtral-dictate

This file is for AI assistants helping modify this codebase.

## What This Is

A Linux dictation tool: hotkey → speak → text typed into focused window.
Go binary, ~1100 lines. Daemon + toggle client over Unix socket.

## Key Files to Edit

### Changing typing method
Edit `typist.go`. Each method is a function (`xdotool()`, `ydotool()`, etc.).
To add a new method: add a function, add a case to `Type()`, update config.

### Changing/adding a backend
1. Create `backend_NAME.go` implementing `Backend` interface from `backend.go`
2. Add constructor in `backend.go` `NewBackend()` switch
3. Add config struct in `config.go` under `BackendConfig`
4. Document in `config.example.toml`

The Backend interface is:
```go
type Backend interface {
    Transcribe(ctx context.Context, audioCh <-chan []byte, textCh chan<- string) error
}
```
- Read PCM s16le bytes from audioCh
- Write text fragments to textCh
- Return when audioCh closes or ctx is cancelled

### Changing audio capture
Edit `audio.go`. The `Recorder` builds command-line args for `pw-record` or
`arecord` and pipes stdout. To support a new capture tool, add to `buildArgs()`.

### Changing config
Edit `config.go` (Go structs) and `config.example.toml` (user-facing docs).
Config is TOML via `github.com/BurntSushi/toml`. The `defaultConfig()` function
provides defaults for all fields.

### Changing daemon IPC
Edit `daemon.go`. The `handleConn()` method reads commands from the Unix socket.
Currently supports `toggle` and `status`. To add commands, add cases there.

## Audio Format Convention

Everything is **PCM s16le, 16000 Hz, mono**. This is hardcoded to match Voxtral.
Don't change the sample rate without updating all backends.

## Testing Without Hardware

```bash
# Use mock backend (no model, no mic, no display server needed):
echo '[backend]\nname = "mock"' > /tmp/test.toml
DICTATE_CONFIG=/tmp/test.toml ./dictate test /path/to/audio.pcm

# Create PCM test files from any audio:
ffmpeg -i input.mp3 -ar 16000 -ac 1 -f s16le output.pcm
```

## Dependencies

- `github.com/BurntSushi/toml` — config parsing
- `github.com/coder/websocket` — WebSocket client for realtime backends
- Runtime: `pw-record` or `arecord` (audio), `xdotool`/`ydotool`/`wtype`/`dotool` (typing)

## Model Server Relationship

The dictate daemon does NOT start or manage the model server.
The model server (llama.cpp or vLLM) runs separately, always resident.
The daemon just connects to it via HTTP or WebSocket when a session starts.
See `systemd/` for example service units.

## Common Tasks

**Add a new STT provider (e.g., Groq, Deepgram):**
1. Create `backend_groq.go` implementing `Backend`
2. Probably HTTP-based like `backend_mistral_batch.go` — accumulate audio, POST WAV
3. Add to `NewBackend()` switch and config

**Add notification on toggle:**
In `daemon.go`, `startDictation()` / `stopDictation()` — add `exec.Command("notify-send", ...)` calls.

**Add VAD (voice activity detection):**
Wrap the `audioCh` in a filtering goroutine that only forwards chunks containing
speech. Silero VAD ONNX is the standard approach. Would need an ONNX runtime
dependency or a subprocess.

**Change from Unix socket to something else:**
All IPC is in `daemon.go` (`runDaemon`, `handleConn`) and `main.go` (`runToggle`).
These are the only two places that touch the socket.
