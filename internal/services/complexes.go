package services

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"nesta/internal/repositories"
)

type ComplexService struct {
	DB              *sql.DB
	Complexes       *repositories.ComplexRepository
	Requests        *repositories.ComplexRequestRepository
	ThresholdStatus string
}

func (s *ComplexService) CreateRequest(ctx context.Context, complexID, phone string) (repositories.ComplexRequest, repositories.ResidentialComplex, error) {
	request, err := s.Requests.FindByComplexAndPhone(ctx, complexID, phone)
	if err == nil {
		if request.Verified {
			return request, repositories.ResidentialComplex{}, errors.New("already verified")
		}
		return request, repositories.ResidentialComplex{}, errors.New("pending verification")
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return repositories.ComplexRequest{}, repositories.ResidentialComplex{}, err
	}

	id, err := NewID()
	if err != nil {
		return repositories.ComplexRequest{}, repositories.ResidentialComplex{}, err
	}

	request = repositories.ComplexRequest{
		ID:        id,
		ComplexID: complexID,
		Phone:     phone,
		Verified:  false,
	}

	if err := s.Requests.Create(ctx, request); err != nil {
		return repositories.ComplexRequest{}, repositories.ResidentialComplex{}, err
	}

	return request, repositories.ResidentialComplex{}, nil
}

func (s *ComplexService) VerifyRequest(ctx context.Context, request repositories.ComplexRequest) (repositories.ResidentialComplex, error) {
	if request.Verified {
		return repositories.ResidentialComplex{}, errors.New("already verified")
	}

	return s.applyVerification(ctx, request)
}

func (s *ComplexService) applyVerification(ctx context.Context, request repositories.ComplexRequest) (repositories.ResidentialComplex, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return repositories.ResidentialComplex{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	complex, err := s.Complexes.Get(ctx, request.ComplexID)
	if err != nil {
		return repositories.ResidentialComplex{}, err
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE complex_requests SET verified = TRUE, verified_at = $2 WHERE id = $1
	`, request.ID, time.Now())
	if err != nil {
		return repositories.ResidentialComplex{}, err
	}

	newCount := complex.CurrentRequests + 1
	newStatus := complex.Status
	if complex.Threshold > 0 && newCount >= complex.Threshold {
		newStatus = s.ThresholdStatus
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE residential_complexes SET current_requests = $2, status = $3 WHERE id = $1
	`, complex.ID, newCount, newStatus)
	if err != nil {
		return repositories.ResidentialComplex{}, err
	}

	if err := tx.Commit(); err != nil {
		return repositories.ResidentialComplex{}, err
	}

	complex.CurrentRequests = newCount
	complex.Status = newStatus
	return complex, nil
}
