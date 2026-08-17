package payment

import (
	"context"
	"database/sql"
)

type Repository struct {
	database *sql.DB
}

func NewRepository(database *sql.DB) *Repository {
	return &Repository{database: database}
}

func (repository *Repository) SaveIdempotent(context context.Context, payment Payment) (inserted bool, err error) {
	result, err := repository.database.ExecContext(context, `
		INSERT INTO transaction_payments (transaction_id, external_payment_id, amount, status)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (external_payment_id) DO NOTHING
	`, payment.TransactionID, payment.ExternalPaymentID, payment.Amount, payment.Status)
	if err != nil {
		return false, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return rowsAffected == 1, nil
}
