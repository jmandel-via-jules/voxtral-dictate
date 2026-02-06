package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"
)

// runTest feeds a PCM s16le file through the configured backend
// and prints transcribed text to stdout (no xdotool needed).
func runTest(cfg *Config, pcmFile string) {
	data, err := os.ReadFile(pcmFile)
	if err != nil {
		log.Fatalf("read %s: %v", pcmFile, err)
	}

	// Calculate audio duration
	samples := len(data) / 2 // 16-bit = 2 bytes per sample
	duration := time.Duration(float64(samples) / float64(cfg.Audio.SampleRate) * float64(time.Second))
	log.Printf("Audio: %s (%.1fs, %d bytes)", pcmFile, duration.Seconds(), len(data))

	backend, err := NewBackend(cfg)
	if err != nil {
		log.Fatalf("backend: %v", err)
	}

	// Feed audio in chunks to simulate real-time streaming
	chunkBytes := cfg.Audio.SampleRate * 2 * cfg.Audio.ChunkMs / 1000
	audioCh := make(chan []byte, 64)
	go func() {
		defer close(audioCh)
		for i := 0; i < len(data); i += chunkBytes {
			end := i + chunkBytes
			if end > len(data) {
				end = len(data)
			}
			chunk := make([]byte, end-i)
			copy(chunk, data[i:end])
			audioCh <- chunk
			// Pace it roughly real-time so streaming backends work naturally
			time.Sleep(time.Duration(cfg.Audio.ChunkMs) * time.Millisecond / 4)
		}
		log.Println("All audio sent")
	}()

	textCh := make(chan string, 32)
	ctx, cancel := context.WithTimeout(context.Background(), duration+60*time.Second)
	defer cancel()

	go func() {
		defer close(textCh)
		if err := backend.Transcribe(ctx, audioCh, textCh); err != nil {
			if ctx.Err() == nil {
				log.Printf("transcribe error: %v", err)
			}
		}
	}()

	start := time.Now()
	for text := range textCh {
		elapsed := time.Since(start).Truncate(time.Millisecond)
		fmt.Printf("[%s] %s", elapsed, text)
	}
	fmt.Println()
	log.Printf("Done in %s", time.Since(start).Truncate(time.Millisecond))
}
