package stocksync

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

func (repository *Repository) ApplyIfNewer(context context.Context, update StockUpdate) (applied bool, err error) {
	result, err := repository.database.ExecContext(context, `
		INSERT INTO ticket_stock_snapshot (ticket_id, quantity, version, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (ticket_id) DO UPDATE
			SET quantity = EXCLUDED.quantity,
			    version = EXCLUDED.version,
			    updated_at = now()
			WHERE ticket_stock_snapshot.version < EXCLUDED.version
	`, update.TicketID, update.Quantity, update.Version)
	if err != nil {
		return false, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return rowsAffected == 1, nil
}

func (repository *Repository) Get(context context.Context, ticketID int64) (StockUpdate, error) {
	var stockUpdate StockUpdate
	err := repository.database.QueryRowContext(context,
		`SELECT ticket_id, quantity, version FROM ticket_stock_snapshot WHERE ticket_id = $1`,
		ticketID,
	).Scan(&stockUpdate.TicketID, &stockUpdate.Quantity, &stockUpdate.Version)
	return stockUpdate, err
}
