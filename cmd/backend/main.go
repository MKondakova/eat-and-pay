package main

import (
	"log"
	"net/http"
	"net/url"
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

	testURL, err := url.Parse("https://basket-01.wbbasket.ru/vol100/part10039/10039442/images/big/1.webp")

	productsService := service.NewProductsService(
		[]*api.Product{{
			ID:          "123",
			Image:       *testURL,
			Name:        "Что-то",
			Weight:      120,
			Price:       11111,
			Rating:      4.6,
			Description: "sdfsdfsdf",
			IsFavorite:  false,
			Discount:    api.OptFloat64{},
			Reviews:     nil,
		}},
		map[string][]string{"lubim": {"123"}},
		map[string]api.Category{"lubim": {
			ID:    "lubim",
			Name:  "Любимое",
			Image: *testURL,
		}},
	)

	srv, err := api.NewServer(
		productsService,
		&handler.SecurityHandler{},
		api.WithMiddleware(handler.Logging(logger)))
	if err != nil {
		log.Fatal(err)
	}

	if err := http.ListenAndServe(":8080", srv); err != nil {
		log.Fatal(err)
	}
}
