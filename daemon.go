package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Daemon struct {
	cfg        *Config
	typist     *Typist
	indicators *IndicatorSet
	mu         sync.Mutex
	active     bool
	cancel     context.CancelFunc // cancels current dictation session
}

func runDaemon(cfg *Config) {
	d := &Daemon{
		cfg:        cfg,
		typist:     NewTypist(cfg.Typing),
		indicators: NewIndicatorSet(cfg.Indicator),
	}

	// Clean up stale socket
	os.Remove(cfg.Daemon.Socket)

	ln, err := net.Listen("unix", cfg.Daemon.Socket)
	if err != nil {
		log.Fatalf("listen %s: %v", cfg.Daemon.Socket, err)
	}
	defer ln.Close()
	defer os.Remove(cfg.Daemon.Socket)

	// Make socket world-writable so toggle works without sudo
	os.Chmod(cfg.Daemon.Socket, 0666)

	log.Printf("Daemon listening on %s (backend=%s, typing=%s)",
		cfg.Daemon.Socket, cfg.Backend.Name, cfg.Typing.Method)

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Shutting down...")
		d.stopDictation()
		d.indicators.Close()
		d.typist.Close()
		ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Err.Error() == "use of closed network connection" {
				return // clean shutdown
			}
			log.Printf("accept: %v", err)
			continue
		}
		go d.handleConn(conn)
	}
}

func (d *Daemon) handleConn(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 64)
	n, err := conn.Read(buf)
	if err != nil || n == 0 {
		return
	}

	cmd := string(buf[:n])
	if len(cmd) > 0 && cmd[len(cmd)-1] == '\n' {
		cmd = cmd[:len(cmd)-1]
	}

	switch cmd {
	case "toggle":
		d.mu.Lock()
		wasActive := d.active
		d.mu.Unlock()

		if wasActive {
			d.stopDictation()
			fmt.Fprintf(conn, "stopped\n")
			log.Println("Dictation stopped")
		} else {
			d.startDictation()
			fmt.Fprintf(conn, "started\n")
			log.Println("Dictation started")
		}
	case "status":
		d.mu.Lock()
		if d.active {
			fmt.Fprintf(conn, "active\n")
		} else {
			fmt.Fprintf(conn, "idle\n")
		}
		d.mu.Unlock()
	default:
		fmt.Fprintf(conn, "unknown command: %s\n", cmd)
	}
}

func (d *Daemon) startDictation() {
	ctx, cancel := context.WithCancel(context.Background())

	d.mu.Lock()
	d.active = true
	d.cancel = cancel
	d.mu.Unlock()

	d.indicators.On()
	go d.runSession(ctx)
}

func (d *Daemon) stopDictation() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.cancel != nil {
		d.cancel()
		d.cancel = nil
	}
	d.active = false
	d.indicators.Off()
}

func (d *Daemon) runSession(ctx context.Context) {
	defer func() {
		d.mu.Lock()
		d.active = false
		d.cancel = nil
		d.mu.Unlock()
		d.indicators.Off()
		log.Println("Session ended")
	}()

	rec := NewRecorder(d.cfg.Audio)
	audioCh, err := rec.Start(ctx)
	if err != nil {
		log.Printf("recorder start: %v", err)
		return
	}

	// VAD splits audio into speech bursts. Each burst is a channel that
	// opens on speech onset and closes after trailing silence. We connect
	// a backend per burst, so silence = no connection = no billing.
	bursts := vadBursts(ctx, audioCh, d.cfg.Audio.VAD)

	for burst := range bursts {
		if ctx.Err() != nil {
			return
		}
		d.handleBurst(ctx, burst)
	}
}

func (d *Daemon) handleBurst(ctx context.Context, audioCh <-chan []byte) {
	backoff := 500 * time.Millisecond
	maxBackoff := 10 * time.Second

	// Buffer to survive reconnects within a burst
	bufCh := make(chan []byte, 64)
	go func() {
		defer close(bufCh)
		for chunk := range audioCh {
			select {
			case bufCh <- chunk:
			default:
				// Buffer full — drop oldest
				select {
				case <-bufCh:
				default:
				}
				bufCh <- chunk
			}
		}
	}()

	for {
		if ctx.Err() != nil {
			return
		}

		backend, err := NewBackend(d.cfg)
		if err != nil {
			log.Printf("backend init: %v", err)
			return
		}

		textCh := make(chan string, 32)

		done := make(chan error, 1)
		go func() {
			defer close(textCh)
			done <- backend.Transcribe(ctx, bufCh, textCh)
		}()

		midLine := false
		endLine := func() {
			if midLine {
				fmt.Fprintln(os.Stderr)
				midLine = false
			}
		}

		for {
			select {
			case text, ok := <-textCh:
				if !ok {
					endLine()
					goto done
				}
				d.typist.Type(text)
				if !midLine {
					fmt.Fprintf(os.Stderr, "%s transcribed:", time.Now().Format("2006/01/02 15:04:05"))
					midLine = true
				}
				fmt.Fprint(os.Stderr, text)
				backoff = 500 * time.Millisecond
			}
		}
	done:

		err = <-done
		if ctx.Err() != nil {
			return
		}
		if err == nil {
			return // burst ended cleanly (VAD closed the channel)
		}

		endLine()
		log.Printf("transcribe error (retrying in %v): %v", backoff, err)
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return
		}
		backoff = min(backoff*2, maxBackoff)
	}
}
