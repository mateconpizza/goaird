package main

import (
	"context"
	"log"
	"log/slog"
)

const (
	appName    = "goairdrop"
	appVersion = "0.1.1"
)

func main() {
	f, err := setupLogger()
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			logger.Error("Failed closing log file", slog.String("error", err.Error()))
		}
	}()

	server := setupServer(addrFlag)
	registerHandlers()
	ctx := setupInterruptHandler(context.Background(), server.Shutdown)
	serverErr := startServer(server)

	waitForShutdown(ctx, serverErr)
}
