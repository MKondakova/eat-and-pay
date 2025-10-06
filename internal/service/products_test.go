package service_test

import (
	"eats-backend/internal/models"
	"eats-backend/internal/service"
	"fmt"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
)

func TestProductsService_GetProductByID(t *testing.T) {
	ctrl := gomock.NewController(t)

	id := "ff25265d-9dfc-49c3-bd01-678c6baa001f"

	userService := service.NewMockUserService(ctrl)
	service := service.NewProductsService(userService, []*models.Product{
		{
			ID:          id,
			Image:       "https://basket-01.wbbasket.ru/vol100/part10039/10039442/images/big/1.webp",
			Name:        "Мука",
			Weight:      123,
			Price:       1000,
			Rating:      5.6,
			Description: "Норм",
			Discount:    0,
			Reviews:     []models.Review{{Rating: 5, Author: "sdsadas", CreatedAt: time.Now(), Content: "ssdsdfsdfa"}},
		},
	}, map[string][]string{
		"favourite": {"ff25265d-9dfc-49c3-bd01-678c6baa001f"},
	}, map[string]models.Category{
		"favourite": {
			ID:    "favourite",
			Name:  "Любимое",
			Image: "https://basket-01.wbbasket.ru/vol100/part10039/10039442/images/big/1.webp",
		},
	})

	userService.EXPECT().IsFavourite(t.Context(), id).Return(true)
	userService.EXPECT().IsFavourite(t.Context(), id).Return(false)
	fmt.Println(service.GetProductByID(t.Context(), id))
	fmt.Println(service.GetProductByID(t.Context(), id))
}
