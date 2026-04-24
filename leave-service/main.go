package main

import (
	"log"
	"net/http"
	"os"

	"github.com/asdlc-repos/prev-demo092/leave-service/internal/handlers"
	"github.com/asdlc-repos/prev-demo092/leave-service/internal/store"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9090"
	}

	s := store.New()
	h := handlers.New(s)

	addr := ":" + port
	log.Printf("leave-service listening on %s", addr)
	if err := http.ListenAndServe(addr, handlers.LoggingMiddleware(h.Routes())); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
