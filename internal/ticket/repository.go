package ticket

import (
	"context"
	"database/sql"
)

type OutboxWriter interface {
	InsertPending(context context.Context, transaction *sql.Tx, transactionID int64) error
}

type Repository struct {
	database *sql.DB
	outbox   OutboxWriter
}

func NewRepository(database *sql.DB, outbox OutboxWriter) *Repository {
	return &Repository{database: database, outbox: outbox}
}

func (repository *Repository) Purchase(context context.Context, ticketID int64, buyerName string) (int64, error) {
	transaction, err := repository.database.BeginTx(context, nil)
	if err != nil {
		return 0, err
	}
	defer transaction.Rollback()

	result, err := transaction.ExecContext(context,
		`UPDATE tickets SET stock = stock - 1 WHERE id = $1 AND stock > 0`,
		ticketID,
	)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if rowsAffected == 0 {
		return 0, ErrSoldOut
	}

	var transactionID int64
	err = transaction.QueryRowContext(context,
		`INSERT INTO transactions (ticket_id, buyer_name, status) VALUES ($1, $2, 'success') RETURNING id`,
		ticketID, buyerName,
	).Scan(&transactionID)
	if err != nil {
		return 0, err
	}

	if err := repository.outbox.InsertPending(context, transaction, transactionID); err != nil {
		return 0, err
	}

	if err := transaction.Commit(); err != nil {
		return 0, err
	}

	return transactionID, nil
}
