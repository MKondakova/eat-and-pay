package service

//go:generate mockgen -destination=products_mock.go -source=products.go -package=service

import (
	"cmp"
	"context"
	"eats-backend/internal/models"
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"

	api "eats-backend/api/generated"
)

type UserService interface {
	IsFavourite(ctx context.Context, productID string) bool
	AddFavourite(ctx context.Context, id string)
	RemoveFavourite(ctx context.Context, id string)
}

const defaultPageSize = 20

type ProductsService struct {
	api.UnimplementedHandler

	userService UserService

	products            []*models.Product
	productsPerCategory map[string][]*models.Product
	productIndex        map[string]*models.Product

	categories map[string]models.Category
}

func NewProductsService(
	userService UserService,
	products []*models.Product,
	productIDsPerCategory map[string][]string,
	categories map[string]models.Category,
) *ProductsService {
	index := make(map[string]*models.Product, len(products))

	for i := range products {
		index[products[i].ID] = products[i]
	}

	productsPerCategory := make(map[string][]*models.Product)
	for category, IDs := range productIDsPerCategory {
		productsPerCategory[category] = make([]*models.Product, len(IDs))
		for i, ID := range IDs {
			productsPerCategory[category][i] = index[ID]
		}
	}

	return &ProductsService{
		userService:         userService,
		products:            products,
		productIndex:        index,
		categories:          categories,
		productsPerCategory: productsPerCategory,
	}
}

func (s *ProductsService) GetCategories() []models.Category {
	categories := slices.SortedFunc(maps.Values(s.categories), func(a models.Category, b models.Category) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return categories
}

func (s *ProductsService) GetProductsList(ctx context.Context, page, pageSize int, category string) (models.ProductsList, error) {
	products := s.products

	if category != "" {
		if _, categoryExists := s.categories[category]; !categoryExists {
			return models.ProductsList{}, errors.New("category not found")
		}

		products = s.productsPerCategory[category]

	}

	productsAmount := len(products)
	totalPages := int(math.Ceil(float64(productsAmount) / float64(pageSize)))

	paginationStart := (page - 1) * defaultPageSize

	if paginationStart >= productsAmount {
		return models.ProductsList{
			CurrentPage: page,
			TotalPages:  totalPages,
			Data:        nil,
		}, nil
	}

	paginationEnd := paginationStart + defaultPageSize
	if paginationEnd > productsAmount {
		paginationEnd = productsAmount
	}

	listLen := paginationEnd - paginationStart
	result := make([]models.ProductPreview, 0, listLen)

	for i := paginationStart; i < paginationEnd; i++ {
		product := products[i]
		preview := product.ToPreview()
		preview.IsFavorite = s.userService.IsFavourite(ctx, product.ID)

		result = append(result, preview)
	}

	return models.ProductsList{
		CurrentPage: page,
		TotalPages:  totalPages,
		Data:        result,
	}, nil
}

func (s *ProductsService) GetProductByID(ctx context.Context, id string) (models.Product, error) {
	productLink, ok := s.productIndex[id]
	if !ok {
		return models.Product{}, fmt.Errorf("%w: no such product", models.ErrNotFound)
	}

	product := *productLink
	product.IsFavorite = s.userService.IsFavourite(ctx, product.ID)

	return product, nil
}

func (s *ProductsService) AddFavourite(ctx context.Context, id string) error {
	_, ok := s.productIndex[id]
	if !ok {
		return fmt.Errorf("%w: no such product", models.ErrNotFound)
	}

	s.userService.AddFavourite(ctx, id)

	return nil
}

func (s *ProductsService) RemoveFavourite(ctx context.Context, id string) error {
	_, ok := s.productIndex[id]
	if !ok {
		return fmt.Errorf("%w: no such product", models.ErrNotFound)
	}

	s.userService.RemoveFavourite(ctx, id)

	return nil
}
