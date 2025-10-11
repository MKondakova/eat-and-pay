package service

import (
	"context"
	"sync"

	"eats-backend/internal/models"
)

type Favourites struct {
	favourites map[string]map[string]struct{}

	mux sync.Mutex
}

func NewFavouritesService(favouritesData map[string][]string) *Favourites {
	result := &Favourites{favourites: make(map[string]map[string]struct{})}

	// Преобразуем данные из списка строк в map[string]struct{}
	for userID, favouriteList := range favouritesData {
		result.favourites[userID] = make(map[string]struct{})
		for _, productID := range favouriteList {
			result.favourites[userID][productID] = struct{}{}
		}
	}

	return result
}

func (s *Favourites) IsFavourite(ctx context.Context, id string) bool {
	userID := models.ClaimsFromContext(ctx).ID

	s.mux.Lock()
	defer s.mux.Unlock()

	if _, ok := s.favourites[userID]; !ok {
		s.favourites[userID] = make(map[string]struct{})

		return false
	}

	_, has := s.favourites[userID][id]

	return has
}

func (s *Favourites) AddFavourite(ctx context.Context, id string) {
	userID := models.ClaimsFromContext(ctx).ID

	s.mux.Lock()
	defer s.mux.Unlock()

	if _, ok := s.favourites[userID]; !ok {
		s.favourites[userID] = make(map[string]struct{})
	}

	_, has := s.favourites[userID][id]
	if has {
		return
	}

	s.favourites[userID][id] = struct{}{}
}

func (s *Favourites) RemoveFavourite(ctx context.Context, id string) {
	userID := models.ClaimsFromContext(ctx).ID

	s.mux.Lock()
	defer s.mux.Unlock()

	if _, ok := s.favourites[userID]; !ok {
		return
	}

	delete(s.favourites[userID], id)
}

// GetBackupData возвращает данные для бэкапа
func (s *Favourites) GetBackupData() interface{} {
	s.mux.Lock()
	defer s.mux.Unlock()

	// Создаем копию данных для бэкапа
	backupData := make(map[string][]string)
	for userID, favourites := range s.favourites {
		favouriteList := make([]string, 0, len(favourites))
		for productID := range favourites {
			favouriteList = append(favouriteList, productID)
		}
		backupData[userID] = favouriteList
	}

	return backupData
}

// GetBackupFileName возвращает имя файла для бэкапа
func (s *Favourites) GetBackupFileName() string {
	return "user_favourites"
}
