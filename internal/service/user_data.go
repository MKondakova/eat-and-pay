package service

import (
	"context"
	"eats-backend/internal/models"
	"sync"
)

type UserData struct {
	favourites map[string]map[string]struct{}

	mux sync.Mutex
}

func NewUserData() *UserData {
	return &UserData{
		favourites: make(map[string]map[string]struct{}),
	}
}

func (s *UserData) IsFavourite(ctx context.Context, id string) bool {
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

func (s *UserData) AddFavourite(ctx context.Context, id string) {
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

func (s *UserData) RemoveFavourite(ctx context.Context, id string) {
	userID := models.ClaimsFromContext(ctx).ID

	s.mux.Lock()
	defer s.mux.Unlock()

	if _, ok := s.favourites[userID]; !ok {
		return
	}

	delete(s.favourites[userID], id)
}
