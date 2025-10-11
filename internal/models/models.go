package models

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const DefaultPageSize = 20

type Product struct {
	ID          string  `json:"id"`
	Image       string  `json:"image"`
	Name        string  `json:"name"`
	Weight      int     `json:"weight"`
	Price       int     `json:"price"`
	Rating      float32 `json:"rating"`
	Description string  `json:"description"`
	// Размер скидки.
	Discount   int      `json:"discount,omitempty"`
	Reviews    []Review `json:"reviews"`
	IsFavorite bool     `json:"isFavorite"`
	Available  bool     `json:"-"`
}

type Review struct {
	Rating    int       `json:"rating"`
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"createdAt"`
	Content   string    `json:"content"`
	Images    []string  `json:"images"`
}

type PostReviewRequest struct {
	Rating  int      `json:"rating"`
	Content string   `json:"content"`
	Images  []string `json:"images"`
}

type ProductPreview struct {
	ID          string  `json:"id"`
	Image       string  `json:"image"`
	Name        string  `json:"name"`
	Weight      int     `json:"weight"`
	Price       int     `json:"price"`
	Rating      float32 `json:"rating"`
	ReviewCount int     `json:"reviewCount"`
	IsFavorite  bool    `json:"isFavorite"`
	// Размер скидки.
	Discount int `json:"discount,omitempty"`
}

func (p *Product) ToPreview() ProductPreview {
	return ProductPreview{
		ID:          p.ID,
		Name:        p.Name,
		Price:       p.Price,
		Image:       p.Image,
		Rating:      p.Rating,
		Weight:      p.Weight,
		Discount:    p.Discount,
		ReviewCount: len(p.Reviews),
	}
}

type ProductsList struct {
	CurrentPage int              `json:"currentPage"`
	TotalPages  int              `json:"totalPages"`
	Data        []ProductPreview `json:"data"`
}

type Category struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Image string `json:"image"`
}
type AuthTokenClaims struct {
	*jwt.RegisteredClaims

	Nickname  string `json:"nickname"`
	IsTeacher bool   `json:"isTeacher"`
}

type ContextClaimsKey struct{}

func ClaimsFromContext(ctx context.Context) *AuthTokenClaims {
	claims, _ := ctx.Value(ContextClaimsKey{}).(*AuthTokenClaims)

	return claims
}

type UserProfile struct {
	Phone    string `json:"phone"`
	Name     string `json:"name"`
	Birthday string `json:"birthday"`
	Image    string `json:"imageUri"`
}

type UpdateUserRequest struct {
	Name     string `json:"name"`
	Birthday string `json:"birthday"`
	Image    string `json:"imageUri"`
}

type Address struct {
	ID string `json:"id"`
	// Массив [долгота, широта].
	Coordinates  []float64 `json:"coordinates"`
	AddressLine  string    `json:"addressLine"`
	Floor        string    `json:"floor"`
	Entrance     string    `json:"entrance"`
	IntercomCode string    `json:"intercomCode"`
	Comment      string    `json:"comment"`
}

type OrderStatus string

const (
	OrderStatusActive    OrderStatus = "active"
	OrderStatusCompleted OrderStatus = "completed"
)

type Order struct {
	ID           string      `json:"id"`
	Status       OrderStatus `json:"status"`
	DeliveryDate string      `json:"deliveryDate"`
	Address      Address     `json:"address"`
	// Стоимость товаров в заказе.
	OrderPrice int `json:"orderPrice"`
	// Стоимость доставки.
	DeliveryPrice int `json:"deliveryPrice"`
	// Общая стоимость.
	TotalPrice int         `json:"totalPrice"`
	TotalItems int         `json:"totalItems"`
	Items      []OrderItem `json:"items"`
	CreatedAt  time.Time   `json:"-"`
}

type OrderItem struct {
	ID       string `json:"id"`
	Image    string `json:"image"`
	Name     string `json:"name"`
	Weight   int    `json:"weight"`
	Price    int    `json:"price"`
	Quantity int    `json:"quantity"`
}

type CartResponse struct {
	// Сколько минут займет доставка.
	DeliveryTime int `json:"deliveryTime"`
	// Стоимость товаров в заказе.
	OrderPrice int `json:"orderPrice"`
	// Стоимость доставки.
	DeliveryPrice int `json:"deliveryPrice"`
	// Общая стоимость.
	TotalPrice int                `json:"totalPrice"`
	TotalItems int                `json:"totalItems"`
	Items      []CartResponseItem `json:"items"`
}

type CartResponseItem struct {
	ProductID string `json:"id"`
	Image     string `json:"image"`
	Name      string `json:"name"`
	Weight    int    `json:"weight"`
	Price     int    `json:"price"`
	Quantity  int    `json:"quantity"`
	Available bool   `json:"available"`
}

type CartItem struct {
	ProductID string `json:"id"`
	Quantity  int    `json:"quantity"`
}

type OrderRequest struct {
	PaymentMethod string `json:"paymentMethod"`
	// Id выбранного адерса.
	AddressID string `json:"addressid"`
}

// Wallet models
type AccountType string

const (
	AccountTypeCard    AccountType = "card"
	AccountTypeSavings AccountType = "savings"
)

type Account struct {
	ID      string      `json:"id"`
	Type    AccountType `json:"type"`
	Balance int         `json:"balance"` // Баланс в рублях
}

type Wallet struct {
	Accounts []Account `json:"accounts"`
}

type Transaction struct {
	Amount int       `json:"amount"` // Сумма в рублях (отрицательная для трат, положительная для доходов)
	Title  string    `json:"title"`
	Time   time.Time `json:"time"`
	Icon   string    `json:"icon"`
}

type TransactionsByDate map[string][]Transaction

type TransactionsResponse struct {
	CurrentPage int                `json:"currentPage"`
	TotalPages  int                `json:"totalPages"`
	Data        TransactionsByDate `json:"data"`
}

type TopupRequest struct {
	AccountID string `json:"accountId"`
	Amount    int    `json:"amount"` // Сумма пополнения в рублях (максимум 1000 рублей в сутки)
}

type TopupResponse struct {
	Balance int `json:"balance"` // Новый баланс в рублях
}

type TransferRequest struct {
	FromAccountID string `json:"fromAccountId"`
	ToPhoneNumber string `json:"toPhoneNumber"`
	Amount        int    `json:"amount"` // Сумма перевода в рублях
}

type TransferResponse struct {
	Balance int `json:"balance"` // Новый баланс отправителя в рублях
}
