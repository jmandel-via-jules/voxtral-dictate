//go:build ignore

// mock_server.go - standalone mock STT server for testing.
// Run with: go run mock_server.go
// It accepts POST /v1/audio/transcriptions (Mistral-compatible)
// and returns a canned transcription.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/v1/audio/transcriptions", func(w http.ResponseWriter, r *http.Request) {
		// Read the uploaded audio to get its size
		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "no file: "+err.Error(), 400)
			return
		}
		defer file.Close()
		data, _ := io.ReadAll(file)
		model := r.FormValue("model")
		log.Printf("Received %s (%d bytes), model=%s", header.Filename, len(data), model)

		// Return a fake transcription
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"text": fmt.Sprintf("[mock transcription of %d bytes of audio]", len(data)),
		})
	})

	log.Println("Mock STT server on :9090")
	log.Fatal(http.ListenAndServe(":9090", nil))
}
