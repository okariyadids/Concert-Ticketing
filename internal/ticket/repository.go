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

func (r *Repository) DecreaseStock(ctx context.Context, ticketID int64) (bool, error) {
	result, err := r.db.ExecContext(ctx,
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

func (r *Repository) SaveTransaction(ctx context.Context, ticketID int64, buyerName string) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO transactions (ticket_id, buyer_name, status) VALUES ($1, $2, 'success') RETURNING id`,
		ticketID, buyerName,
	).Scan(&id)
	return id, err
}

func (r *Repository) GetTicket(ctx context.Context, ticketID int64) (*Ticket, error) {
	var t Ticket
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, stock, price FROM tickets WHERE id = $1`,
		ticketID,
	).Scan(&t.ID, &t.Name, &t.Stock, &t.Price)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
