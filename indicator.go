package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

type Indicator interface {
	On()
	Off()
	Close()
}

type IndicatorSet struct {
	indicators []Indicator
}

func NewIndicatorSet(cfgs []IndicatorConfig) *IndicatorSet {
	var indicators []Indicator
	for _, c := range cfgs {
		switch c.Type {
		case "led":
			indicators = append(indicators, newLEDIndicator(c))
		case "dunstify":
			indicators = append(indicators, newDunstifyIndicator(c))
		case "command":
			indicators = append(indicators, newCommandIndicator(c))
		default:
			log.Printf("unknown indicator type: %q", c.Type)
		}
	}
	return &IndicatorSet{indicators: indicators}
}

func (s *IndicatorSet) On() {
	for _, ind := range s.indicators {
		ind.On()
	}
}

func (s *IndicatorSet) Off() {
	for _, ind := range s.indicators {
		ind.Off()
	}
}

func (s *IndicatorSet) Close() {
	for _, ind := range s.indicators {
		ind.Close()
	}
}

// --- LED indicator ---

type ledIndicator struct {
	ledNumber    int
	mode         string // "on" or "blink"
	savedWasOn   bool
	procPath     string
	sysfsPath    string
}

func newLEDIndicator(c IndicatorConfig) *ledIndicator {
	mode := c.Mode
	if mode == "" {
		mode = "blink"
	}
	return &ledIndicator{
		ledNumber: c.LEDNumber,
		mode:      mode,
		procPath:  "/proc/acpi/ibm/led",
		sysfsPath: fmt.Sprintf("/sys/class/leds/tpacpi::power/brightness"),
	}
}

func (l *ledIndicator) On() {
	// Save current state via sysfs
	data, err := os.ReadFile(l.sysfsPath)
	if err != nil {
		log.Printf("indicator/led: read brightness: %v", err)
		l.savedWasOn = true // assume on
	} else {
		l.savedWasOn = strings.TrimSpace(string(data)) != "0"
	}

	// Activate
	cmd := fmt.Sprintf("%d %s", l.ledNumber, l.mode)
	if err := os.WriteFile(l.procPath, []byte(cmd), 0644); err != nil {
		log.Printf("indicator/led: write %q to %s: %v (check permissions)", cmd, l.procPath, err)
	}
}

func (l *ledIndicator) Off() {
	restore := "off"
	if l.savedWasOn {
		restore = "on"
	}
	cmd := fmt.Sprintf("%d %s", l.ledNumber, restore)
	if err := os.WriteFile(l.procPath, []byte(cmd), 0644); err != nil {
		log.Printf("indicator/led: write %q to %s: %v", cmd, l.procPath, err)
	}
}

func (l *ledIndicator) Close() {
	l.Off()
}

// --- Dunstify indicator ---

type dunstifyIndicator struct {
	message string
	urgency string
}

const dunstifyReplaceID = "999111"

func newDunstifyIndicator(c IndicatorConfig) *dunstifyIndicator {
	msg := c.Message
	if msg == "" {
		msg = "Dictating..."
	}
	urg := c.Urgency
	if urg == "" {
		urg = "critical"
	}
	return &dunstifyIndicator{message: msg, urgency: urg}
}

func (d *dunstifyIndicator) On() {
	cmd := exec.Command("dunstify", "-a", "dictate", "-r", dunstifyReplaceID, "-t", "0", "-u", d.urgency, d.message)
	if err := cmd.Run(); err != nil {
		log.Printf("indicator/dunstify: on: %v", err)
	}
}

func (d *dunstifyIndicator) Off() {
	cmd := exec.Command("dunstify", "-C", dunstifyReplaceID)
	if err := cmd.Run(); err != nil {
		log.Printf("indicator/dunstify: off: %v", err)
	}
}

func (d *dunstifyIndicator) Close() {
	d.Off()
}

// --- Command indicator ---

type commandIndicator struct {
	startCmd string
	stopCmd  string
}

func newCommandIndicator(c IndicatorConfig) *commandIndicator {
	return &commandIndicator{startCmd: c.StartCmd, stopCmd: c.StopCmd}
}

func (ci *commandIndicator) On() {
	if ci.startCmd == "" {
		return
	}
	if err := exec.Command("sh", "-c", ci.startCmd).Run(); err != nil {
		log.Printf("indicator/command: start: %v", err)
	}
}

func (ci *commandIndicator) Off() {
	if ci.stopCmd == "" {
		return
	}
	if err := exec.Command("sh", "-c", ci.stopCmd).Run(); err != nil {
		log.Printf("indicator/command: stop: %v", err)
	}
}

func (ci *commandIndicator) Close() {
	ci.Off()
}
