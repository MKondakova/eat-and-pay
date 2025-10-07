package service

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"eats-backend/internal/models"
)

type UserData struct {
	profileInfo map[string]*models.UserProfile

	mux sync.Mutex
}

func NewUserData() *UserData {
	result := &UserData{
		profileInfo: make(map[string]*models.UserProfile),
	}

	result.profileInfo["4479081e-fd93-499c-bf8b-1ad190b052e6"] = &models.UserProfile{Phone: "123123"}

	return result
}

func (s *UserData) GetProfile(ctx context.Context) (*models.UserProfile, error) {
	userID := models.ClaimsFromContext(ctx).ID

	s.mux.Lock()
	defer s.mux.Unlock()

	if profile, ok := s.profileInfo[userID]; ok {
		return profile, nil
	}

	return nil, fmt.Errorf("%w: profile not found", models.ErrForbidden)
}

func (s *UserData) UpdateProfile(ctx context.Context, data models.UpdateUserRequest) error {
	userID := models.ClaimsFromContext(ctx).ID

	name := strings.TrimSpace(data.Name)

	birthday, err := parseBirthday(data.Birthday)
	if err != nil {
		return err
	}

	if _, err = url.ParseRequestURI(data.Image); err != nil {
		return fmt.Errorf("%w: invalid image url: %w", models.ErrBadRequest, err)
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	s.profileInfo[userID].Name = name
	s.profileInfo[userID].Birthday = birthday
	s.profileInfo[userID].Image = data.Image

	return nil
}

func (s *UserData) DeleteProfile(ctx context.Context) error {
	userID := models.ClaimsFromContext(ctx).ID

	s.mux.Lock()
	defer s.mux.Unlock()

	s.profileInfo[userID].Name = ""
	s.profileInfo[userID].Birthday = ""
	s.profileInfo[userID].Image = ""

	return nil
}

func parseBirthday(birthday string) (string, error) {
	birthday = strings.TrimSpace(birthday)

	if birthday == "" {
		return "", nil
	}

	if _, err := time.Parse("02.01.2006", birthday); err != nil {
		return "", fmt.Errorf("%w: wrong birthday format, should be 02.01.2006", models.ErrBadRequest)
	}

	return birthday, nil
}
