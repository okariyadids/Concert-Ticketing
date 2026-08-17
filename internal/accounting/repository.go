package accounting

import (
	"context"
	"database/sql"
	"time"
)

type Repository struct {
	database *sql.DB
}

func NewRepository(database *sql.DB) *Repository {
	return &Repository{database: database}
}

func (repository *Repository) InsertPending(context context.Context, transaction *sql.Tx, transactionID int64) error {
	_, err := transaction.ExecContext(context,
		`INSERT INTO accounting_outbox (transaction_id) VALUES ($1)`,
		transactionID,
	)
	return err
}

func (repository *Repository) FetchDue(context context.Context, limit int) ([]OutboxEntry, error) {
	rows, err := repository.database.QueryContext(context, `
		SELECT o.id, o.transaction_id, o.attempt_count, t.ticket_id, t.buyer_name
		FROM accounting_outbox o
		JOIN transactions t ON t.id = o.transaction_id
		WHERE o.status = 'pending' AND o.next_attempt_at <= now()
		ORDER BY o.id
		LIMIT $1
		FOR UPDATE OF o SKIP LOCKED
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []OutboxEntry
	for rows.Next() {
		var entry OutboxEntry
		if err := rows.Scan(&entry.ID, &entry.TransactionID, &entry.AttemptCount, &entry.TicketID, &entry.BuyerName); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (repository *Repository) MarkSent(context context.Context, outboxID int64) error {
	_, err := repository.database.ExecContext(context,
		`UPDATE accounting_outbox SET status = 'sent', updated_at = now() WHERE id = $1`,
		outboxID,
	)
	return err
}

func (repository *Repository) MarkForRetry(context context.Context, outboxID int64, attemptCount int, lastError string, nextAttemptAt time.Time) error {
	_, err := repository.database.ExecContext(context, `
		UPDATE accounting_outbox
		SET attempt_count = $2, last_error = $3, next_attempt_at = $4, updated_at = now()
		WHERE id = $1
	`, outboxID, attemptCount, lastError, nextAttemptAt)
	return err
}

func (repository *Repository) MarkDead(context context.Context, outboxID int64, attemptCount int, lastError string) error {
	_, err := repository.database.ExecContext(context, `
		UPDATE accounting_outbox
		SET status = 'failed', attempt_count = $2, last_error = $3, updated_at = now()
		WHERE id = $1
	`, outboxID, attemptCount, lastError)
	return err
}
