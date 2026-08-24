package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"waternet/internal/console"
)

func main() {
	dataDir := flag.String("data", "./data", "directory for file persistence")
	addr := flag.String("addr", "127.0.0.1:8080", "HTTP listen address")
	interval := flag.Duration("interval", 5*time.Second, "control cycle interval")
	flag.Parse()

	api, err := console.Bootstrap(*dataDir)
	if err != nil {
		log.Fatalf("waternet: bootstrap: %v", err)
	}
	httpServer := console.NewHTTPServer(*addr, api)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	schedulerErrors := make(chan error, 1)
	go func() {
		ticker := time.NewTicker(*interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := api.Tick(ctx); err != nil {
					schedulerErrors <- fmt.Errorf("control cycle: %w", err)
					return
				}
			}
		}
	}()

	go func() {
		log.Printf("waternet listening on %s (data=%s)", *addr, *dataDir)
		if err := httpServer.Listen(ctx); err != nil {
			schedulerErrors <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Println("waternet shutting down")
	case err := <-schedulerErrors:
		log.Fatalf("waternet: %v", err)
	}
}
