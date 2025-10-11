package service

import (
	"context"
	"fmt"
	"sync"

	"eats-backend/internal/models"

	"go.uber.org/zap"
)

type ProductService interface {
	GetProductByID(ctx context.Context, id string) (models.Product, error)
	ProductExists(id string) bool
}

type Cart struct {
	items map[string]map[string]*models.CartItem

	productService ProductService
	logger         *zap.SugaredLogger

	mux sync.RWMutex
}

func NewCart(productService ProductService, logger *zap.SugaredLogger, items map[string]map[string]*models.CartItem) *Cart {
	return &Cart{
		items:          items,
		productService: productService,
		logger:         logger,
	}
}

func (s *Cart) GetCart(ctx context.Context) (models.CartResponse, error) {
	userID := models.ClaimsFromContext(ctx).ID

	response := models.CartResponse{
		DeliveryTime:  15,
		DeliveryPrice: 150,
		Items:         make([]models.CartResponseItem, 0),
	}

	s.mux.RLock()
	defer s.mux.RUnlock()

	if cart, ok := s.items[userID]; ok {
		if len(cart) > 0 {
			for _, item := range cart {
				responseItem, err := s.getCartResponseItem(ctx, item)
				if err != nil {
					s.logger.Errorf("failed to get cart response item: %v", err)

					continue
				}

				if responseItem.Available {
					response.OrderPrice += responseItem.Price * responseItem.Quantity
					response.TotalItems += responseItem.Quantity
				}

				response.Items = append(response.Items, responseItem)
			}
		}
	}

	response.TotalPrice = response.DeliveryPrice + response.OrderPrice

	return response, nil
}

func (s *Cart) AddItem(ctx context.Context, productID string) (int, error) {
	userID := models.ClaimsFromContext(ctx).ID

	if !s.productService.ProductExists(productID) {
		return 0, fmt.Errorf("%w: product %s does not exist", models.ErrNotFound, productID)
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	if _, ok := s.items[userID]; !ok {
		s.items[userID] = make(map[string]*models.CartItem)
	}

	if _, ok := s.items[userID][productID]; !ok {
		s.items[userID][productID] = &models.CartItem{
			ProductID: productID,
			Quantity:  1,
		}

		return 1, nil
	}

	s.items[userID][productID].Quantity++

	return s.items[userID][productID].Quantity, nil
}

func (s *Cart) RemoveItem(ctx context.Context, productID string) (int, error) {
	userID := models.ClaimsFromContext(ctx).ID

	if !s.productService.ProductExists(productID) {
		return 0, fmt.Errorf("%w: product %s does not exist", models.ErrNotFound, productID)
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	if _, ok := s.items[userID]; !ok {
		s.items[userID] = make(map[string]*models.CartItem)
	}

	if _, ok := s.items[userID][productID]; !ok {
		return 0, nil
	}

	s.items[userID][productID].Quantity--
	if s.items[userID][productID].Quantity <= 0 {
		delete(s.items[userID], productID)

		return 0, nil
	}

	return s.items[userID][productID].Quantity, nil

}

func (s *Cart) ClearCart(ctx context.Context) {
	userID := models.ClaimsFromContext(ctx).ID

	s.mux.Lock()
	defer s.mux.Unlock()

	delete(s.items, userID)

	return
}

func (s *Cart) getCartResponseItem(ctx context.Context, item *models.CartItem) (models.CartResponseItem, error) {
	result := models.CartResponseItem{
		ProductID: item.ProductID,
		Quantity:  item.Quantity,
	}

	product, err := s.productService.GetProductByID(ctx, item.ProductID)
	if err != nil {
		return models.CartResponseItem{}, fmt.Errorf("failed to get product by id: %w", err)
	}

	result.Name = product.Name
	result.Weight = product.Weight
	result.Price = product.Price
	result.Available = product.Available
	result.Image = product.Image

	return result, nil
}

// GetBackupData возвращает данные для бэкапа
func (s *Cart) GetBackupData() interface{} {
	s.mux.RLock()
	defer s.mux.RUnlock()

	// Создаем копию данных для бэкапа
	backupData := make(map[string]map[string]*models.CartItem)
	for userID, cart := range s.items {
		backupCart := make(map[string]*models.CartItem)
		for productID, item := range cart {
			backupItem := &models.CartItem{
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
			}
			backupCart[productID] = backupItem
		}
		backupData[userID] = backupCart
	}

	return backupData
}

// GetBackupFileName возвращает имя файла для бэкапа
func (s *Cart) GetBackupFileName() string {
	return "cart_items"
}
