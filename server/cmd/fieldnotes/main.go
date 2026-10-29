package main

import (
	"context"
	"errors"
	"fieldnotes/server/internal/inspection"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	store, err := inspection.NewStore(env("DATA_FILE", "../data/issues.json"))
	if err != nil {
		log.Fatalf("Cannot start with data file: %v", err)
	}
	origins := strings.Split(env("CORS_ORIGINS", "http://localhost:5173"), ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}
	server := &http.Server{Addr: env("ADDR", ":8080"), Handler: inspection.NewHandler(store, inspection.Options{CORSOrigins: origins, WebDir: os.Getenv("WEB_DIR")}), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	failed := make(chan error, 1)
	go func() { log.Printf("FieldNotes listening on %s", server.Addr); failed <- server.ListenAndServe() }()
	select {
	case err := <-failed:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server: %v", err)
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			log.Printf("Graceful shutdown: %v", err)
			_ = server.Close()
		}
	}
}
func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
