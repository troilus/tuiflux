package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/anomalyco/tuiflux/internal/app"
	"github.com/anomalyco/tuiflux/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Println("No config found. Starting setup...")
		cfg = config.Setup()
	}

	tuiApp := app.New(cfg)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		tuiApp.Stop()
	}()

	if err := tuiApp.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
