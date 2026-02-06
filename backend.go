package main

import (
	"context"
	"fmt"
)

// Backend streams audio to an STT service and returns text fragments.
type Backend interface {
	// Transcribe reads PCM chunks from audioCh and sends text fragments to textCh.
	// It returns when audioCh is closed or ctx is cancelled.
	Transcribe(ctx context.Context, audioCh <-chan []byte, textCh chan<- string) error
}

func NewBackend(cfg *Config) (Backend, error) {
	switch cfg.Backend.Name {
	case "mistral-realtime":
		return NewWebSocketBackend(
			"wss://api.mistral.ai/v1/audio/transcriptions/realtime?model="+cfg.Backend.MistralRT.Model,
			cfg.Backend.MistralRT.Model,
			mustGetMistralAPIKey(cfg),
			cfg.Audio.SampleRate,
		), nil
	case "mistral-batch":
		return NewMistralBatchBackend(
			mustGetMistralAPIKey(cfg),
			"voxtral-mini-latest",
			cfg.Audio.SampleRate,
			5, // 5 second chunks
		), nil
	case "vllm-realtime":
		return NewWebSocketBackend(
			cfg.Backend.VllmRT.URL,
			cfg.Backend.VllmRT.Model,
			"", // no API key for local
			cfg.Audio.SampleRate,
		), nil
	case "llamacpp":
		return NewLlamaCppBackend(
			cfg.Backend.LlamaCpp.URL,
			cfg.Audio.SampleRate,
			cfg.Backend.LlamaCpp.ChunkSeconds,
		), nil
	case "mock":
		return NewMockBackend(cfg.Audio.SampleRate), nil
	default:
		return nil, fmt.Errorf("unknown backend: %q", cfg.Backend.Name)
	}
}
