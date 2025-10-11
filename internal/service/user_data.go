package service

import (
	"context"
	"fmt"
	"math/rand"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"eats-backend/internal/models"
)

type UserData struct {
	profileInfo map[string]*models.UserProfile

	mux sync.Mutex
}

func NewUserData(profiles map[string]*models.UserProfile) *UserData {
	return &UserData{
		profileInfo: profiles,
	}
}

// generateRandomPhoneNumber генерирует случайный номер телефона, начинающийся с "79"
func generateRandomPhoneNumber() string {
	// Генерируем 9 случайных цифр (79 + 9 цифр = 11 цифр всего)
	var phoneNumber strings.Builder
	phoneNumber.WriteString("79")

	for i := 0; i < 9; i++ {
		phoneNumber.WriteString(fmt.Sprintf("%d", rand.Intn(10)))
	}

	return phoneNumber.String()
}

func (s *UserData) GetProfile(ctx context.Context) (*models.UserProfile, error) {
	userID := models.ClaimsFromContext(ctx).ID

	s.mux.Lock()
	defer s.mux.Unlock()

	if _, ok := s.profileInfo[userID]; !ok {
		s.profileInfo[userID] = &models.UserProfile{
			Phone:    generateRandomPhoneNumber(),
			Name:     "",
			Birthday: "",
			Image:    "",
		}
	}

	return s.profileInfo[userID], nil
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

	// Check if the URL points to a .jxl file
	parsedURL, err := url.Parse(data.Image)
	if err != nil {
		return fmt.Errorf("%w: invalid image url: %w", models.ErrBadRequest, err)
	}

	fileExt := strings.ToLower(filepath.Ext(parsedURL.Path))
	if fileExt != ".jxl" {
		return fmt.Errorf("%w: image must be a .jxl file", models.ErrBadRequest)
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

// GetBackupData возвращает данные для бэкапа
func (s *UserData) GetBackupData() interface{} {
	s.mux.Lock()
	defer s.mux.Unlock()

	// Создаем копию данных для бэкапа
	backupData := make(map[string]*models.UserProfile)
	for id, profile := range s.profileInfo {
		backupProfile := &models.UserProfile{
			Phone:    profile.Phone,
			Name:     profile.Name,
			Birthday: profile.Birthday,
			Image:    profile.Image,
		}
		backupData[id] = backupProfile
	}

	return backupData
}

// GetUserIDByPhone возвращает ID пользователя по номеру телефона
func (s *UserData) GetUserIDByPhone(phone string) (string, bool) {
	s.mux.Lock()
	defer s.mux.Unlock()

	for userID, profile := range s.profileInfo {
		if profile.Phone == phone {
			return userID, true
		}
	}
	return "", false
}

// GetBackupFileName возвращает имя файла для бэкапа
func (s *UserData) GetBackupFileName() string {
	return "user_profiles"
}
