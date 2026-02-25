package main

import (
	"bingwa-service/internal/app"
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("[MAIN] No .env file found, relying on system env vars")
	}
	srv := app.NewServer()

	// Run server in a separate goroutine so we can listen for shutdown signals
	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("❌ Server failed to start: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")

	// Cleanup logic (DB, Redis, etc.) can go here
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_ = ctx
	defer cancel()

	// You can later expose srv.Shutdown(ctx) if you manage DB or Redis pools
	log.Println("✅ Server stopped gracefully")
}
