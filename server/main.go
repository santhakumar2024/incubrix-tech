package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL must be set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := openDatabase(ctx, databaseURL)
	if err != nil {
		log.Fatalf("initialize database: %v", err)
	}
	defer pool.Close()

	hub := newEventHub()
	dataRepository := newRepository(pool)
	dataAccess := newDAO(dataRepository)
	appService := newService(dataAccess, hub)
	appController := newController(appService, hub, os.Getenv("COOKIE_SECURE") == "true", envOr("WEB_DIST", "../dist"))
	port := envOr("PORT", "8080")
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		log.Fatalf("PORT must be a valid port number: %q", port)
	}
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           appController.routes(),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("Seatflow API listening on :%s", port)
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("serve HTTP: %v", err)
	}
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
