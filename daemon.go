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
)

type Daemon struct {
	cfg     *Config
	typist  *Typist
	mu      sync.Mutex
	active  bool
	cancel  context.CancelFunc // cancels current dictation session
}

func runDaemon(cfg *Config) {
	d := &Daemon{
		cfg:    cfg,
		typist: NewTypist(cfg.Typing),
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
}

func (d *Daemon) runSession(ctx context.Context) {
	defer func() {
		d.mu.Lock()
		d.active = false
		d.cancel = nil
		d.mu.Unlock()
		log.Println("Session ended")
	}()

	backend, err := NewBackend(d.cfg)
	if err != nil {
		log.Printf("backend init: %v", err)
		return
	}

	rec := NewRecorder(d.cfg.Audio)
	audioCh, err := rec.Start(ctx)
	if err != nil {
		log.Printf("recorder start: %v", err)
		return
	}

	textCh := make(chan string, 32)

	// Run backend transcription
	go func() {
		defer close(textCh)
		if err := backend.Transcribe(ctx, audioCh, textCh); err != nil {
			if ctx.Err() == nil {
				log.Printf("transcribe error: %v", err)
			}
		}
	}()

	// Type out text as it arrives
	for text := range textCh {
		d.typist.Type(text)
	}
}
