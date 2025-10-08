package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"

	"eats-backend/internal/models"
)

type AddressService struct {
	addresses map[string][]*models.Address

	mux sync.RWMutex
}

func NewAddressService() *AddressService {
	return &AddressService{
		addresses: make(map[string][]*models.Address),
	}
}

func (s *AddressService) GetAddresses(ctx context.Context) []*models.Address {
	userID := models.ClaimsFromContext(ctx).ID

	s.mux.RLock()
	defer s.mux.RUnlock()

	if addresses, ok := s.addresses[userID]; ok {
		return addresses
	}

	return []*models.Address{}
}

func (s *AddressService) AddAddress(ctx context.Context, address *models.Address) error {
	userID := models.ClaimsFromContext(ctx).ID

	if err := validateAddress(address); err != nil {
		return err
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	address.ID = uuid.NewString()

	if _, ok := s.addresses[userID]; !ok {
		s.addresses[userID] = make([]*models.Address, 0)
	}

	s.addresses[userID] = append(s.addresses[userID], address)

	return nil
}

func (s *AddressService) RemoveAddress(ctx context.Context, addressID string) error {
	userID := models.ClaimsFromContext(ctx).ID

	s.mux.Lock()
	defer s.mux.Unlock()

	if _, ok := s.addresses[userID]; !ok {
		return fmt.Errorf("%w: address not found", models.ErrNotFound)
	}

	for i, address := range s.addresses[userID] {
		if address.ID == addressID {
			s.addresses[userID] = append(s.addresses[userID][:i], s.addresses[userID][i+1:]...)

			return nil
		}
	}

	return fmt.Errorf("%w: address not found", models.ErrNotFound)
}

func (s *AddressService) UpdateAddress(ctx context.Context, newAddress *models.Address) error {
	userID := models.ClaimsFromContext(ctx).ID

	if err := validateAddress(newAddress); err != nil {
		return err
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	if _, ok := s.addresses[userID]; !ok {
		return fmt.Errorf("%w: address not found", models.ErrNotFound)
	}

	for i, address := range s.addresses[userID] {
		if address.ID == newAddress.ID {
			s.addresses[userID][i] = newAddress

			return nil
		}
	}

	return fmt.Errorf("%w: address not found", models.ErrNotFound)
}

func (s *AddressService) GetAddressByID(ctx context.Context, addressID string) (models.Address, error) {
	userID := models.ClaimsFromContext(ctx).ID

	s.mux.RLock()
	defer s.mux.RUnlock()

	if addresses, ok := s.addresses[userID]; ok {
		for _, address := range addresses {
			if address.ID == addressID {
				return *address, nil
			}
		}
	}

	return models.Address{}, fmt.Errorf("%w: address not found", models.ErrNotFound)
}

func validateCoordinates(coordinates []float64) error {
	if len(coordinates) != 2 {
		return fmt.Errorf("%w: invalid coordinates amount, should be two numbers", models.ErrBadRequest)
	}

	lon := coordinates[0]
	if lon < -180 || lon > 180 {
		return fmt.Errorf("%w: invalid coordinates, longitude should be between -180 and 180", models.ErrBadRequest)
	}

	lat := coordinates[1]
	if lat < -90 || lat > 90 {
		return fmt.Errorf("%w: invalid coordinates, latitude should be between -90 and 90", models.ErrBadRequest)
	}

	return nil
}

func validateAddress(address *models.Address) error {
	if address.AddressLine == "" {
		return fmt.Errorf("%w: address line required", models.ErrBadRequest)
	}

	if err := validateCoordinates(address.Coordinates); err != nil {
		return fmt.Errorf("%w: %w", models.ErrBadRequest, err)
	}

	return nil
}
