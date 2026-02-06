package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
)

// LlamaCppBackend sends accumulated audio chunks to llama.cpp's
// /v1/chat/completions endpoint with audio content.
// Not true streaming — accumulates chunkSeconds of audio, then sends.
type LlamaCppBackend struct {
	url          string
	sampleRate   int
	chunkSeconds int
}

func NewLlamaCppBackend(url string, sampleRate, chunkSeconds int) *LlamaCppBackend {
	if chunkSeconds <= 0 {
		chunkSeconds = 3
	}
	return &LlamaCppBackend{
		url:          url,
		sampleRate:   sampleRate,
		chunkSeconds: chunkSeconds,
	}
}

func (b *LlamaCppBackend) Transcribe(ctx context.Context, audioCh <-chan []byte, textCh chan<- string) error {
	bytesPerChunkPeriod := b.sampleRate * 2 * b.chunkSeconds // 2 bytes per sample, mono

	var accum []byte

	for {
		select {
		case <-ctx.Done():
			// Flush remaining audio
			if len(accum) > 0 {
				b.sendChunk(ctx, accum, textCh)
			}
			return nil
		case chunk, ok := <-audioCh:
			if !ok {
				if len(accum) > 0 {
					b.sendChunk(ctx, accum, textCh)
				}
				return nil
			}
			accum = append(accum, chunk...)
			if len(accum) >= bytesPerChunkPeriod {
				b.sendChunk(ctx, accum, textCh)
				accum = nil
			}
		}
	}
}

func (b *LlamaCppBackend) sendChunk(ctx context.Context, pcm []byte, textCh chan<- string) {
	audioB64 := base64.StdEncoding.EncodeToString(pcm)

	reqBody := map[string]any{
		"messages": []map[string]any{{
			"role": "user",
			"content": []map[string]any{
				{
					"type":        "input_audio",
					"input_audio": audioB64,
				},
				{
					"type": "text",
					"text": "Transcribe the audio exactly. Output only the transcription.",
				},
			},
		}},
		"temperature": 0.0,
		"stream":      false,
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", b.url, bytes.NewReader(body))
	if err != nil {
		log.Printf("llamacpp request build: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("llamacpp request: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		data, _ := io.ReadAll(resp.Body)
		log.Printf("llamacpp %d: %s", resp.StatusCode, data)
		return
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("llamacpp decode: %v", err)
		return
	}

	if len(result.Choices) > 0 {
		text := result.Choices[0].Message.Content
		if text != "" {
			select {
			case textCh <- text:
			case <-ctx.Done():
			}
		}
	}
}
