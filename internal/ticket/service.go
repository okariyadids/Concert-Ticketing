package ticket

import (
	"context"
	"errors"
)

var ErrSoldOut = errors.New("ticket sold out")

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (service *Service) Purchase(context context.Context, ticketID int64, buyerName string) (int64, error) {
	return service.repository.Purchase(context, ticketID, buyerName)
}
