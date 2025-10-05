package service

import (
	"cmp"
	"context"
	"maps"
	"math"
	"slices"

	api "eats-backend/api/generated"
)

const defaultPageSize = 20

type ProductsService struct {
	api.UnimplementedHandler

	products            []*api.Product
	productsPerCategory map[string][]*api.Product
	productIndex        map[string]*api.Product

	categories map[string]api.Category
}

func NewProductsService(
	products []*api.Product,
	productIDsPerCategory map[string][]string,
	categories map[string]api.Category,
) *ProductsService {
	index := make(map[string]*api.Product, len(products))

	for i := range products {
		index[products[i].ID] = products[i]
	}

	productsPerCategory := make(map[string][]*api.Product)
	for category, IDs := range productIDsPerCategory {
		productsPerCategory[category] = make([]*api.Product, len(IDs))
		for i, ID := range IDs {
			productsPerCategory[category][i] = index[ID]
		}
	}

	return &ProductsService{
		products:            products,
		productIndex:        index,
		categories:          categories,
		productsPerCategory: productsPerCategory,
	}
}

func (s *ProductsService) CategoriesGet(_ context.Context) (api.CategoriesGetRes, error) {
	categories := slices.SortedFunc(maps.Values(s.categories), func(a api.Category, b api.Category) int {
		return cmp.Compare(a.Name, b.Name)
	})

	result := api.CategoriesGetOKApplicationJSON(categories)

	return &result, nil
}

func (s *ProductsService) ProductsGet(_ context.Context, params api.ProductsGetParams) (api.ProductsGetRes, error) {
	page, ok := params.Page.Get()
	if !ok {
		page = 1
	}

	pageSize, ok := params.PageSize.Get()
	if !ok {
		pageSize = defaultPageSize
	}

	products := s.products

	if category, ok := params.Category.Get(); ok {
		if _, categoryExists := s.categories[category]; !categoryExists {
			return new(api.ProductsGetBadRequest), nil
		}

		products = s.productsPerCategory[category]

	}

	productsAmount := len(products)
	totalPages := int(math.Ceil(float64(productsAmount) / float64(pageSize)))

	paginationStart := (page - 1) * defaultPageSize

	if paginationStart >= productsAmount {
		return &api.ProductsGetOK{
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
	result := make([]api.ProductPreview, 0, listLen)

	for i := paginationStart; i < paginationEnd; i++ {
		product := products[i]
		result = append(result, api.ProductPreview{
			ID:          product.ID,
			Image:       product.Image,
			Name:        product.Name,
			Weight:      product.Weight,
			Price:       product.Price,
			Rating:      product.Rating,
			ReviewCount: len(product.Reviews),
			IsFavorite:  false, // todo: depends on user
			Discount:    product.Discount,
		})
	}

	return &api.ProductsGetOK{
		CurrentPage: page,
		TotalPages:  totalPages,
		Data:        result,
	}, nil
}
