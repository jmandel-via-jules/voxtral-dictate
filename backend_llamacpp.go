package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
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
	// Build a minimal WAV header around the raw PCM so llama.cpp can decode it
	wavData := pcmToWAV(pcm, b.sampleRate)
	audioB64 := base64.StdEncoding.EncodeToString(wavData)

	reqBody := map[string]any{
		"messages": []map[string]any{{
			"role": "user",
			"content": []map[string]any{
				{
					"type": "input_audio",
					"input_audio": map[string]any{
						"data":   audioB64,
						"format": "wav",
					},
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

// pcmToWAV wraps raw PCM s16le mono data in a minimal WAV header.
func pcmToWAV(pcm []byte, sampleRate int) []byte {
	var buf bytes.Buffer
	dataLen := uint32(len(pcm))
	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, uint32(36+dataLen))
	buf.WriteString("WAVEfmt ")
	binary.Write(&buf, binary.LittleEndian, uint32(16))    // chunk size
	binary.Write(&buf, binary.LittleEndian, uint16(1))     // PCM
	binary.Write(&buf, binary.LittleEndian, uint16(1))     // mono
	binary.Write(&buf, binary.LittleEndian, uint32(sampleRate))
	binary.Write(&buf, binary.LittleEndian, uint32(sampleRate*2)) // byte rate
	binary.Write(&buf, binary.LittleEndian, uint16(2))     // block align
	binary.Write(&buf, binary.LittleEndian, uint16(16))    // bits per sample
	buf.WriteString("data")
	binary.Write(&buf, binary.LittleEndian, dataLen)
	buf.Write(pcm)
	return buf.Bytes()
}
