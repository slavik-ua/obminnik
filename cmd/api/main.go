package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"simple-orderbook/internal/adapters/repository"
	"simple-orderbook/internal/api"
	"simple-orderbook/internal/database"
	"simple-orderbook/internal/db"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Could not load .env file")
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL is not set in .env file")
	}

	pool, err := database.NewPostgresPool(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Could create database pool: %s, %v", dbURL, err)
	}
	defer pool.Close()

	store := db.NewStore(pool)
	repo := repository.NewPostgresOrderRepository(store)
	handler := api.NewOrderHandler(repo)

	mux := http.NewServeMux()

	mux.HandleFunc("/order", handler.CreateOrder)

	server := &http.Server{
		Addr:    ":8000",
		Handler: mux,

		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Println("Started listening on :8000...")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting gracefully")
}
