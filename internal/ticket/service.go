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
	ok, err := service.repository.DecreaseStock(context, ticketID)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, ErrSoldOut
	}

	transactionID, err := service.repository.SaveTransaction(context, ticketID, buyerName)
	if err != nil {
		return 0, err
	}

	return transactionID, nil
}
