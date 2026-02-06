package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

// MockBackend simulates an STT backend for testing.
// It reads audio chunks and emits fake transcription text
// at realistic intervals to verify the full pipeline.
type MockBackend struct {
	sampleRate int
}

func NewMockBackend(sampleRate int) *MockBackend {
	return &MockBackend{sampleRate: sampleRate}
}

func (b *MockBackend) Transcribe(ctx context.Context, audioCh <-chan []byte, textCh chan<- string) error {
	words := []string{
		"The ", "quick ", "brown ", "fox ", "jumps ", "over ",
		"the ", "lazy ", "dog. ",
		"This ", "is ", "a ", "test ", "of ", "the ",
		"dictation ", "system. ",
	}

	totalBytes := 0
	wordIdx := 0

	for {
		select {
		case <-ctx.Done():
			return nil
		case chunk, ok := <-audioCh:
			if !ok {
				return nil
			}
			totalBytes += len(chunk)
			audioSec := float64(totalBytes) / float64(b.sampleRate*2)

			// Emit a word roughly every 0.5s of audio
			for audioSec > float64(wordIdx+1)*0.5 && wordIdx < len(words) {
				word := words[wordIdx%len(words)]
				log.Printf("mock: %.1fs audio -> emit %q", audioSec, word)
				select {
				case textCh <- word:
				case <-ctx.Done():
					return nil
				}
				wordIdx++
				time.Sleep(50 * time.Millisecond) // simulate processing time
			}

			if wordIdx >= len(words) {
				log.Printf("mock: done after %.1fs audio (%d bytes)", audioSec, totalBytes)
				// Drain remaining audio
				for range audioCh {
				}
				return nil
			}
		}
	}
}

func init() {
	_ = fmt.Sprintf // ensure fmt is used
}
