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
	"net/url"
	"slices"
	"sync"
	"time"

	api "eats-backend/api/generated"
)

type FavouritesService interface {
	IsFavourite(ctx context.Context, productID string) bool
	AddFavourite(ctx context.Context, id string)
	RemoveFavourite(ctx context.Context, id string)
}

const defaultPageSize = 20

type ProductsService struct {
	api.UnimplementedHandler

	favourites FavouritesService

	products            []*models.Product
	productsPerCategory map[string][]*models.Product
	productIndex        map[string]*models.Product

	categories map[string]models.Category

	mux sync.RWMutex
}

func NewProductsService(
	favourites FavouritesService,
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
		favourites:          favourites,
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
	s.mux.RLock()
	defer s.mux.RUnlock()

	products := s.products

	if category != "" && category != "favourite" {
		if _, categoryExists := s.categories[category]; !categoryExists {
			return models.ProductsList{}, errors.New("category not found")
		}

		products = s.productsPerCategory[category]

	}

	if category == "favourite" {
		products = make([]*models.Product, 0)
		for _, product := range s.products {
			if s.favourites.IsFavourite(ctx, product.ID) {
				products = append(products, product)
			}
		}
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
		preview.IsFavorite = s.favourites.IsFavourite(ctx, product.ID)

		result = append(result, preview)
	}

	return models.ProductsList{
		CurrentPage: page,
		TotalPages:  totalPages,
		Data:        result,
	}, nil
}

func (s *ProductsService) GetProductByID(ctx context.Context, id string) (models.Product, error) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	productLink, ok := s.productIndex[id]
	if !ok {
		return models.Product{}, fmt.Errorf("%w: no such product", models.ErrNotFound)
	}

	product := *productLink
	product.IsFavorite = s.favourites.IsFavourite(ctx, product.ID)

	return product, nil
}

func (s *ProductsService) AddFavourite(ctx context.Context, id string) error {
	_, ok := s.productIndex[id]
	if !ok {
		return fmt.Errorf("%w: no such product", models.ErrNotFound)
	}

	s.favourites.AddFavourite(ctx, id)

	return nil
}

func (s *ProductsService) RemoveFavourite(ctx context.Context, id string) error {
	_, ok := s.productIndex[id]
	if !ok {
		return fmt.Errorf("%w: no such product", models.ErrNotFound)
	}

	s.favourites.RemoveFavourite(ctx, id)

	return nil
}

func (s *ProductsService) ProductExists(id string) bool {
	_, ok := s.productIndex[id]

	return ok
}

func (s *ProductsService) AddReview(ctx context.Context, review models.PostReviewRequest, productID string) error {
	name := models.ClaimsFromContext(ctx).Nickname

	if review.Rating > 5 || review.Rating < 1 {
		return fmt.Errorf("%w: rating must be between 1 and 5", models.ErrBadRequest)
	}

	for _, image := range review.Images {
		if _, err := url.Parse(image); err != nil {
			return fmt.Errorf("%w: invalid image: %s must be url", models.ErrBadRequest, image)
		}
	}

	if _, ok := s.productIndex[productID]; !ok {
		return fmt.Errorf("%w: no such product", models.ErrNotFound)
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	newReview := models.Review{
		Rating:    review.Rating,
		Author:    name,
		CreatedAt: time.Now(),
		Content:   review.Content,
		Images:    review.Images,
	}

	product := s.productIndex[productID]
	if product.Reviews == nil {
		product.Reviews = make([]models.Review, 0)
	}

	product.Reviews = append(product.Reviews, newReview)

	return nil
}
