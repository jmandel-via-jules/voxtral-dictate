package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

// Typist injects text as keystrokes into the focused window.
type Typist struct {
	method string
}

func NewTypist(cfg TypingConfig) *Typist {
	return &Typist{method: cfg.Method}
}

// Type sends text to the focused window.
func (t *Typist) Type(text string) {
	if text == "" {
		return
	}
	var err error
	switch t.method {
	case "xdotool":
		err = t.xdotool(text)
	case "ydotool":
		err = t.ydotool(text)
	case "wtype":
		err = t.wtype(text)
	case "dotool":
		err = t.dotool(text)
	default:
		err = fmt.Errorf("unknown typing method: %s", t.method)
	}
	if err != nil {
		log.Printf("typist error (%s): %v", t.method, err)
	}
}

func runCmd(timeout time.Duration, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.WaitDelay = timeout
	return cmd.Run()
}

func (t *Typist) xdotool(text string) error {
	return runCmd(10*time.Second, "xdotool", "type", "--clearmodifiers", "--", text)
}

func (t *Typist) ydotool(text string) error {
	return runCmd(10*time.Second, "ydotool", "type", "--", text)
}

func (t *Typist) wtype(text string) error {
	return runCmd(10*time.Second, "wtype", "--", text)
}

func (t *Typist) dotool(text string) error {
	cmd := exec.Command("dotool")
	cmd.Stdin = strings.NewReader("type " + text + "\n")
	cmd.WaitDelay = 10 * time.Second
	return cmd.Run()
}
