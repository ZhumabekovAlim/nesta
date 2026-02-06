package services

import (
	"context"
	"encoding/json"
	"errors"

	"nesta/internal/repositories"
)

type AddressService struct {
	Addresses *repositories.AddressRepository
	Complexes *repositories.ComplexRepository
}

type AddressInput struct {
	Name      string
	ComplexID string
	Address   map[string]any
}

func (s *AddressService) Create(ctx context.Context, userID string, input AddressInput) (repositories.Address, error) {
	if input.Name == "" {
		return repositories.Address{}, errors.New("address name required")
	}
	if input.ComplexID == "" {
		return repositories.Address{}, errors.New("complex required")
	}
	if input.Address == nil {
		return repositories.Address{}, errors.New("address required")
	}

	complex, err := s.Complexes.Get(ctx, input.ComplexID)
	if err != nil {
		return repositories.Address{}, err
	}
	if complex.Status != "ACTIVE" {
		return repositories.Address{}, errors.New("complex is not active")
	}

	raw, err := json.Marshal(input.Address)
	if err != nil {
		return repositories.Address{}, errors.New("invalid address")
	}

	id, err := NewID()
	if err != nil {
		return repositories.Address{}, err
	}

	address := repositories.Address{
		ID:          id,
		UserID:      userID,
		Name:        input.Name,
		ComplexID:   input.ComplexID,
		AddressJSON: raw,
	}

	if err := s.Addresses.Create(ctx, address); err != nil {
		return repositories.Address{}, err
	}

	return address, nil
}

func (s *AddressService) Update(ctx context.Context, userID, addressID string, input AddressInput) error {
	if input.Name == "" {
		return errors.New("address name required")
	}
	if input.ComplexID == "" {
		return errors.New("complex required")
	}
	if input.Address == nil {
		return errors.New("address required")
	}

	address, err := s.Addresses.Get(ctx, addressID)
	if err != nil {
		return err
	}
	if address.UserID != userID {
		return errors.New("address not found")
	}

	complex, err := s.Complexes.Get(ctx, input.ComplexID)
	if err != nil {
		return err
	}
	if complex.Status != "ACTIVE" {
		return errors.New("complex is not active")
	}

	raw, err := json.Marshal(input.Address)
	if err != nil {
		return errors.New("invalid address")
	}

	_, err = s.Addresses.Update(ctx, repositories.Address{
		ID:          addressID,
		UserID:      userID,
		Name:        input.Name,
		ComplexID:   input.ComplexID,
		AddressJSON: raw,
	})
	return err
}

func (s *AddressService) Delete(ctx context.Context, userID, addressID string) (bool, error) {
	affected, err := s.Addresses.Delete(ctx, addressID, userID)
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}
