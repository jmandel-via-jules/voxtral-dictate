package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// WebSocketBackend works with both Mistral Realtime API and local vLLM Realtime.
// They use the same WebSocket protocol.
type WebSocketBackend struct {
	url        string
	model      string
	apiKey     string
	sampleRate int
}

func NewWebSocketBackend(url, model, apiKey string, sampleRate int) *WebSocketBackend {
	return &WebSocketBackend{
		url:        url,
		model:      model,
		apiKey:     apiKey,
		sampleRate: sampleRate,
	}
}

type wsSessionUpdate struct {
	Type    string    `json:"type"`
	Session wsSession `json:"session"`
}

type wsSession struct {
	Model      string        `json:"model"`
	InputFmt   wsAudioFormat `json:"input_audio_format"`
	Temperature float64      `json:"temperature"`
}

type wsAudioFormat struct {
	Encoding   string `json:"encoding"`
	SampleRate int    `json:"sample_rate"`
}

type wsAudioAppend struct {
	Type  string `json:"type"`
	Audio string `json:"audio"`
}

type wsEvent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

func (b *WebSocketBackend) Transcribe(ctx context.Context, audioCh <-chan []byte, textCh chan<- string) error {
	opts := &websocket.DialOptions{}
	if b.apiKey != "" {
		opts.HTTPHeader = http.Header{
			"Authorization": {"Bearer " + b.apiKey},
		}
	}

	conn, _, err := websocket.Dial(ctx, b.url, opts)
	if err != nil {
		return fmt.Errorf("ws dial %s: %w", b.url, err)
	}
	defer conn.CloseNow()

	// Increase read limit for large responses
	conn.SetReadLimit(10 * 1024 * 1024)

	// Send session config
	err = wsjson.Write(ctx, conn, wsSessionUpdate{
		Type: "session.update",
		Session: wsSession{
			Model:       b.model,
			InputFmt:    wsAudioFormat{Encoding: "pcm_s16le", SampleRate: b.sampleRate},
			Temperature: 0.0,
		},
	})
	if err != nil {
		return fmt.Errorf("ws session update: %w", err)
	}

	log.Printf("WebSocket connected to %s (model=%s)", b.url, b.model)

	// Send audio in background
	ctx2, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		defer cancel()
		for {
			select {
			case <-ctx2.Done():
				return
			case chunk, ok := <-audioCh:
				if !ok {
					return
				}
				b64 := base64.StdEncoding.EncodeToString(chunk)
				msg := wsAudioAppend{
					Type:  "input_audio_buffer.append",
					Audio: b64,
				}
				data, _ := json.Marshal(msg)
				if err := conn.Write(ctx2, websocket.MessageText, data); err != nil {
					log.Printf("ws write error: %v", err)
					return
				}
			}
		}
	}()

	// Read text events
	for {
		var ev wsEvent
		err := wsjson.Read(ctx2, conn, &ev)
		if err != nil {
			if ctx2.Err() != nil {
				return nil // normal shutdown
			}
			return fmt.Errorf("ws read: %w", err)
		}
		switch ev.Type {
		case "transcription.text.delta":
			if ev.Text != "" {
				select {
				case textCh <- ev.Text:
				case <-ctx2.Done():
					return nil
				}
			}
		case "error":
			log.Printf("ws error event: %+v", ev)
		}
	}
}
