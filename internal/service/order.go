package service

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"eats-backend/internal/models"

	"github.com/google/uuid"
)

const DeliveryTime = time.Minute * 10

type CartService interface {
	ClearCart(ctx context.Context)
	GetCart(ctx context.Context) (models.CartResponse, error)
}

type AddressChecker interface {
	GetAddressByID(ctx context.Context, addressID string) (models.Address, error)
}

type OrderService struct {
	orders         map[string][]*models.Order
	addressService AddressChecker
	cartService    CartService

	mux sync.RWMutex
}

func NewOrderService(addressService AddressChecker, cartService CartService, orders map[string][]*models.Order) *OrderService {
	return &OrderService{
		orders:         orders,
		addressService: addressService,
		cartService:    cartService,
	}
}

func (s *OrderService) GetOrders(ctx context.Context) ([]*models.Order, error) {
	userID := models.ClaimsFromContext(ctx).ID

	s.mux.RLock()
	defer s.mux.RUnlock()

	if _, ok := s.orders[userID]; !ok {
		return []*models.Order{}, nil
	}

	result := make([]*models.Order, 0, len(s.orders[userID]))

	for _, order := range s.orders[userID] {
		if order.Status == models.OrderStatusActive && order.CreatedAt.Add(DeliveryTime).Before(time.Now()) {
			order.Status = models.OrderStatusCompleted
			order.DeliveryDate = formatRu(order.CreatedAt.Add(DeliveryTime))
		}

		result = append(result, order)
	}

	slices.Reverse(result)
	return result, nil

}

func (s *OrderService) MakeNewOrder(ctx context.Context, orderRequest *models.OrderRequest) error {
	userID := models.ClaimsFromContext(ctx).ID

	address, err := s.addressService.GetAddressByID(ctx, orderRequest.AddressID)
	if err != nil {
		return fmt.Errorf("get address: %w", err)
	}

	cart, err := s.cartService.GetCart(ctx)
	if err != nil {
		return fmt.Errorf("get cart: %w", err)
	}

	items := make([]models.OrderItem, 0)

	for _, item := range cart.Items {
		if !item.Available {
			continue
		}

		items = append(items, models.OrderItem{
			ID:       item.ProductID,
			Image:    item.Image,
			Name:     item.Name,
			Weight:   item.Weight,
			Price:    item.Price,
			Quantity: item.Quantity,
		})
	}

	if len(items) == 0 {
		return fmt.Errorf("%w: cart is empty", models.ErrBadRequest)
	}

	s.cartService.ClearCart(ctx)

	newOrder := &models.Order{
		ID:            uuid.NewString(),
		Status:        models.OrderStatusActive,
		Address:       address,
		OrderPrice:    cart.OrderPrice,
		DeliveryPrice: cart.DeliveryPrice,
		TotalPrice:    cart.TotalPrice,
		TotalItems:    cart.TotalItems,
		Items:         items,
		CreatedAt:     time.Now(),
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	if _, ok := s.orders[userID]; !ok {
		s.orders[userID] = make([]*models.Order, 0)
	}

	s.orders[userID] = append(s.orders[userID], newOrder)

	return nil
}

func formatRu(t time.Time) string {
	months := map[time.Month]string{
		time.January:   "января",
		time.February:  "февраля",
		time.March:     "марта",
		time.April:     "апреля",
		time.May:       "мая",
		time.June:      "июня",
		time.July:      "июля",
		time.August:    "августа",
		time.September: "сентября",
		time.October:   "октября",
		time.November:  "ноября",
		time.December:  "декабря",
	}

	return fmt.Sprintf("%d %s в %02d:%02d",
		t.Day(),
		months[t.Month()],
		t.Hour(),
		t.Minute(),
	)
}

// GetBackupData возвращает данные для бэкапа
func (s *OrderService) GetBackupData() interface{} {
	s.mux.RLock()
	defer s.mux.RUnlock()

	// Создаем копию данных для бэкапа
	backupData := make(map[string][]*models.Order)
	for userID, orders := range s.orders {
		backupOrders := make([]*models.Order, len(orders))
		for i, order := range orders {
			// Создаем копию заказа
			backupOrder := &models.Order{
				ID:            order.ID,
				Status:        order.Status,
				Address:       order.Address,
				OrderPrice:    order.OrderPrice,
				DeliveryPrice: order.DeliveryPrice,
				TotalPrice:    order.TotalPrice,
				TotalItems:    order.TotalItems,
				Items:         make([]models.OrderItem, len(order.Items)),
				CreatedAt:     order.CreatedAt,
				DeliveryDate:  order.DeliveryDate,
			}

			// Копируем элементы заказа
			for j, item := range order.Items {
				backupOrder.Items[j] = models.OrderItem{
					ID:       item.ID,
					Image:    item.Image,
					Name:     item.Name,
					Weight:   item.Weight,
					Price:    item.Price,
					Quantity: item.Quantity,
				}
			}

			backupOrders[i] = backupOrder
		}
		backupData[userID] = backupOrders
	}

	return backupData
}

// GetBackupFileName возвращает имя файла для бэкапа
func (s *OrderService) GetBackupFileName() string {
	return "orders"
}
