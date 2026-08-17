package ticket

import (
	"context"
	"errors"
)

var ErrSoldOut = errors.New("ticket sold out")

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Purchase(ctx context.Context, ticketID int64, buyerName string) (int64, error) {
	ok, err := s.repo.DecreaseStock(ctx, ticketID)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, ErrSoldOut
	}

	transactionID, err := s.repo.SaveTransaction(ctx, ticketID, buyerName)
	if err != nil {
		return 0, err
	}

	return transactionID, nil
}
