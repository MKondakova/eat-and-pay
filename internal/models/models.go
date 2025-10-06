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
	Weight      float64 `json:"weight"`
	Price       float64 `json:"price"`
	Rating      float32 `json:"rating"`
	Description string  `json:"description"`
	// Размер скидки.
	Discount   float64  `json:"discount,omitempty"`
	Reviews    []Review `json:"reviews"`
	IsFavorite bool     `json:"isFavorite"`
}

type Review struct {
	Rating    int       `json:"rating"`
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"createdAt"`
	Content   string    `json:"content"`
	Images    []string  `json:"images"`
}

type ProductPageInfo struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	Article           string  `json:"article"`
	Category          string  `json:"category"`
	Description       string  `json:"description"`
	ImageURL          string  `json:"imageUrl"`
	OldPrice          float64 `json:"oldPrice,omitempty"`
	Price             float64 `json:"price"`
	Rating            float64 `json:"rating,omitempty"`
	WarehouseQuantity int     `json:"warehouseQuantity,omitempty"`
	OrdersCount       int     `json:"ordersCount,omitempty"`
}

type ProductPreview struct {
	ID          string  `json:"id"`
	Image       string  `json:"image"`
	Name        string  `json:"name"`
	Weight      float64 `json:"weight"`
	Price       float64 `json:"price"`
	Rating      float32 `json:"rating"`
	ReviewCount int     `json:"reviewCount"`
	IsFavorite  bool    `json:"isFavorite"`
	// Размер скидки.
	Discount float64 `json:"discount,omitempty"`
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
