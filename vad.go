package main

import (
	"context"
	"encoding/binary"
	"log"
	"math"
)

// vadBursts watches an audio stream and yields speech bursts. Each burst is a
// channel of audio chunks that starts with pre-buffered audio before speech
// onset and closes after trailing silence. The caller should connect a backend
// for each burst, then disconnect when the burst channel closes.
//
// If VAD is disabled, yields a single burst that mirrors the input forever.
func vadBursts(ctx context.Context, in <-chan []byte, cfg VADConfig) <-chan (<-chan []byte) {
	bursts := make(chan (<-chan []byte), 1)

	if !cfg.Enabled {
		// VAD disabled — single infinite burst
		go func() {
			defer close(bursts)
			ch := make(chan []byte, cap(in))
			select {
			case bursts <- ch:
			case <-ctx.Done():
				close(ch)
				return
			}
			defer close(ch)
			for {
				select {
				case <-ctx.Done():
					return
				case chunk, ok := <-in:
					if !ok {
						return
					}
					ch <- chunk
				}
			}
		}()
		return bursts
	}

	go func() {
		defer close(bursts)

		type state int
		const (
			silent   state = iota
			speaking
			trailing
		)

		st := silent
		trailLeft := 0
		var burst chan []byte

		// Rolling pre-buffer
		ring := make([][]byte, cfg.PreBufferN)
		ringPos := 0
		ringFull := false

		pushRing := func(chunk []byte) {
			ring[ringPos] = chunk
			ringPos++
			if ringPos >= cfg.PreBufferN {
				ringPos = 0
				ringFull = true
			}
		}

		startBurst := func() chan []byte {
			ch := make(chan []byte, 16)
			select {
			case bursts <- ch:
			case <-ctx.Done():
				close(ch)
				return nil
			}
			// Flush pre-buffer
			n := cfg.PreBufferN
			if !ringFull {
				n = ringPos
			}
			start := 0
			if ringFull {
				start = ringPos
			}
			for i := 0; i < n; i++ {
				idx := (start + i) % cfg.PreBufferN
				if ring[idx] != nil {
					ch <- ring[idx]
					ring[idx] = nil
				}
			}
			ringPos = 0
			ringFull = false
			return ch
		}

		emit := func(chunk []byte) {
			if burst == nil {
				return
			}
			select {
			case burst <- chunk:
			case <-ctx.Done():
			}
		}

		for {
			select {
			case <-ctx.Done():
				if burst != nil {
					close(burst)
				}
				return
			case chunk, ok := <-in:
				if !ok {
					if burst != nil {
						close(burst)
					}
					return
				}

				loud := rmsEnergy(chunk) >= cfg.Threshold

				switch st {
				case silent:
					if loud {
						st = speaking
						log.Println("vad: speech started")
						burst = startBurst()
						if burst == nil {
							return
						}
						emit(chunk)
					} else {
						pushRing(chunk)
					}
				case speaking:
					emit(chunk)
					if !loud {
						st = trailing
						trailLeft = cfg.TrailChunks
					}
				case trailing:
					emit(chunk)
					if loud {
						st = speaking
					} else {
						trailLeft--
						if trailLeft <= 0 {
							st = silent
							log.Println("vad: speech ended, closing burst")
							close(burst)
							burst = nil
						}
					}
				}
			}
		}
	}()

	return bursts
}

// rmsEnergy computes the root-mean-square energy of PCM s16le audio.
func rmsEnergy(pcm []byte) float64 {
	n := len(pcm) / 2
	if n == 0 {
		return 0
	}
	var sum float64
	for i := 0; i < n; i++ {
		sample := int16(binary.LittleEndian.Uint16(pcm[i*2 : i*2+2]))
		sum += float64(sample) * float64(sample)
	}
	return math.Sqrt(sum / float64(n))
}
