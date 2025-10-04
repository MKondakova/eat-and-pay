package service

import (
	"context"

	api "eats-backend/api/generated"
)

type ProductsService struct {
	api.UnimplementedHandler
}

func NewProductsService() *ProductsService {
	return &ProductsService{
		api.UnimplementedHandler{},
	}
}

func (s *ProductsService) CategoriesGet(ctx context.Context) (api.CategoriesGetRes, error) {
	result := &api.CategoriesGetOKApplicationJSON{{
		ID:       "1",
		Name:     "Любимые",
		ImageURL: "https://basket-01.wbbasket.ru/vol100/part10039/10039442/images/big/1.webp",
	}}

	return result, nil
}
