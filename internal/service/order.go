package service

import (
	"context"
	"eats-backend/internal/models"
	"sync"
)

type OrderService struct {
	orders map[string][]*models.Order

	mux sync.Mutex
}

func NewOrderService() *OrderService {
	return &OrderService{
		orders: make(map[string][]*models.Order),
	}
}

func (s *OrderService) GetOrders(ctx context.Context) ([]*models.Order, error) {
	return []*models.Order{}, nil
}

func (s *OrderService) MakeNewOrder(ctx context.Context, orderParameters string) error {
	return nil // where is address?
}
