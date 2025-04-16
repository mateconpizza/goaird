package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// Message represents the structure of the incoming messages.
type Message struct {
	Type    string `json:"type"`
	Content string `json:"content"`
	Action  string `json:"action"`
}

// Response represents the structure of the outgoing messages.
type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// handleOpenAction opens the URL in the default browser.
func handleOpenAction(msg Message) Response {
	resp := Response{
		Success: true,
		Message: "Opened text: " + msg.Content,
	}

	err := openURL(msg.Content)
	if err != nil {
		resp.Success = false
		resp.Message = "Error opening text: " + msg.Content
	}

	return resp
}

// getClientIP returns the IP address of the client.
func getClientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return strings.SplitN(forwarded, ",", 2)[0]
	}

	return r.RemoteAddr
}

// webhookHandler handles the incoming webhook requests.
func webhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	clientIP := getClientIP(r)

	logger.Info("Received request",
		slog.String("ip", clientIP),
		slog.String("user_agent", r.UserAgent()),
		slog.String("path", r.URL.Path),
	)

	var msg Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "Error decoding JSON", http.StatusBadRequest)
		logger.Error("Error decoding JSON", slog.String("error", err.Error()))
		return
	}

	logger.Info("Received action", slog.String("action", msg.Action), slog.String("ip", clientIP))

	var resp Response
	switch msg.Action {
	case "open":
		resp = handleOpenAction(msg)
	default:
		resp = Response{Success: false, Message: "Unknown action: %s" + msg.Action}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Error("Error encoding response", slog.String("error", err.Error()))
	}

	logger.Info("Sent response",
		slog.Bool("success", resp.Success),
		slog.String("message", resp.Message),
	)
}

func setupInterruptHandler(
	ctx context.Context,
	shutdownFunc func(context.Context) error,
) context.Context {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	ctx, cancel := context.WithCancel(ctx)

	go func() {
		<-sigChan
		logger.Debug("Received signal, initiating graceful shutdown")

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := shutdownFunc(shutdownCtx); err != nil {
			logger.Error("Graceful shutdown failed", slog.String("error", err.Error()))
		}

		cancel()
	}()

	return ctx
}

// setupServer configures and returns an HTTP server.
func setupServer(addr string) *http.Server {
	server := &http.Server{
		Addr:         addr,
		Handler:      nil,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return server
}

// startServer starts the HTTP server in a goroutine and returns an error
// channel.
func startServer(server *http.Server) chan error {
	serverErr := make(chan error)
	go func() {
		logger.Info("Starting server", slog.String("addr", addrFlag))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	return serverErr
}

// registerHandlers sets up all HTTP route handlers.
func registerHandlers() {
	http.HandleFunc("/wh", webhookHandler)
}

// waitForShutdown waits for server to finish or be interrupted.
func waitForShutdown(ctx context.Context, serverErr chan error) {
	select {
	case err := <-serverErr:
		logger.Error("Server error", slog.String("error", err.Error()))
	case <-ctx.Done():
		logger.Info("Server stopped gracefully")
	}
}
