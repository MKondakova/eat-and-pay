package service

import (
	"context"
	"sync"

	"eats-backend/internal/models"
)

type Cart struct {
	items map[string]map[string]*models.CartItems

	mux sync.RWMutex
}

func NewCart() *Cart {
	return &Cart{
		items: make(map[string]map[string]*models.CartItems),
	}
}

func (s *Cart) GetCart(ctx context.Context) (models.CartResponse, error) {
	return models.CartResponse{}, nil
}

func (s *Cart) AddItem(ctx context.Context, productID string) (int, error) {
	return 0, nil
}

func (s *Cart) RemoveItem(ctx context.Context, productID string) (int, error) {
	return 0, nil
}

func (s *Cart) ClearCart(ctx context.Context) {
	return
}
