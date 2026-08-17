package ticket

import (
	"context"
	"database/sql"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (repository *Repository) DecreaseStock(context context.Context, ticketID int64) (bool, error) {
	result, err := repository.db.ExecContext(context,
		`UPDATE tickets SET stock = stock - 1 WHERE id = $1 AND stock > 0`,
		ticketID,
	)
	if err != nil {
		return false, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return rowsAffected == 1, nil
}

func (repository *Repository) SaveTransaction(context context.Context, ticketID int64, buyerName string) (int64, error) {
	var id int64
	err := repository.db.QueryRowContext(context,
		`INSERT INTO transactions (ticket_id, buyer_name, status) VALUES ($1, $2, 'success') RETURNING id`,
		ticketID, buyerName,
	).Scan(&id)
	return id, err
}

func (repository *Repository) GetTicket(context context.Context, ticketID int64) (*Ticket, error) {
	var t Ticket
	err := repository.db.QueryRowContext(context,
		`SELECT id, name, stock, price FROM tickets WHERE id = $1`,
		ticketID,
	).Scan(&t.ID, &t.Name, &t.Stock, &t.Price)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
