package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: dictate <daemon|toggle|test FILE>\n")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "daemon":
		cfg := mustLoadConfig()
		runDaemon(cfg)
	case "toggle":
		cfg := mustLoadConfig()
		runToggle(cfg)
	case "test":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "usage: dictate test FILE.pcm\n")
			os.Exit(1)
		}
		cfg := mustLoadConfig()
		runTest(cfg, os.Args[2])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func runToggle(cfg *Config) {
	conn, err := net.Dial("unix", cfg.Daemon.Socket)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot reach daemon at %s: %v\n", cfg.Daemon.Socket, err)
		os.Exit(1)
	}
	defer conn.Close()

	_, err = conn.Write([]byte("toggle\n"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Write failed: %v\n", err)
		os.Exit(1)
	}

	buf := make([]byte, 256)
	n, _ := conn.Read(buf)
	if n > 0 {
		fmt.Print(string(buf[:n]))
	}
}
