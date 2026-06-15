package main

import (
	"log"
	"net/http"
	"time"

	"precious-metal-dashboard/backend/internal/api"
	"precious-metal-dashboard/backend/internal/config"
	"precious-metal-dashboard/backend/internal/database"
)

func main() {
	cfg := config.Load()
	if len(cfg.JWTSecret) < 32 {
		log.Println("warning: JWT_SECRET should be at least 32 characters outside development")
	}
	db, err := database.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	server := &http.Server{Addr: ":" + cfg.Port, Handler: api.New(db, cfg), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	log.Printf("API listening on http://localhost:%s", cfg.Port)
	log.Fatal(server.ListenAndServe())
}
