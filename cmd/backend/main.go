package main

import (
	"log"
	"net/http"
	"time"

	api "eats-backend/api/generated"
	"eats-backend/internal/handler"
	"eats-backend/internal/service"

	"go.uber.org/zap"
)

const shutdownTimeout = 15 * time.Second

func main() {
	zapLog, err := zap.NewProduction()
	if err != nil {
		log.Fatal("can't create logger: %w", err)
	}

	logger := zapLog.Sugar()

	srv, err := api.NewServer(
		service.NewProductsService(),
		&handler.SecurityHandler{},
		api.WithMiddleware(handler.Logging(logger)))
	if err != nil {
		log.Fatal(err)
	}

	if err := http.ListenAndServe(":8080", srv); err != nil {
		log.Fatal(err)
	}
}
