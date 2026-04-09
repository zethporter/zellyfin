package main

import (
	"fmt"
	"os"

	"ripper/internal/config"
	"ripper/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	cfgPath := "/root/.config/zellyfin/config.toml"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}

	m := tui.New(cfg, cfgPath)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
