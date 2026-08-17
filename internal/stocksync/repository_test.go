package stocksync

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://concert:concert@localhost:5433/concert_bnl?sslmode=disable"
	}

	database, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := database.Ping(); err != nil {
		t.Skipf("postgres tidak bisa dijangkau (%v). Jalankan `docker compose up -d` dulu.", err)
	}
	return database
}

func seedTicket(t *testing.T, database *sql.DB) int64 {
	t.Helper()

	var ticketID int64
	err := database.QueryRow(
		`INSERT INTO tickets (name, stock, price) VALUES ('Sync Test Ticket', 10, 100000) RETURNING id`,
	).Scan(&ticketID)
	if err != nil {
		t.Fatalf("failed to seed ticket: %v", err)
	}
	return ticketID
}

func TestApplyIfNewer_OutOfOrderUpdateIsIgnored(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	context := context.Background()
	ticketID := seedTicket(t, database)
	repository := NewRepository(database)

	applied, err := repository.ApplyIfNewer(context, StockUpdate{TicketID: ticketID, Quantity: 2, Version: 2})
	if err != nil {
		t.Fatalf("failed to apply version 2: %v", err)
	}
	if !applied {
		t.Fatalf("expected version 2 (arriving first, no existing data) to be applied")
	}

	applied, err = repository.ApplyIfNewer(context, StockUpdate{TicketID: ticketID, Quantity: 5, Version: 1})
	if err != nil {
		t.Fatalf("failed to apply version 1: %v", err)
	}
	if applied {
		t.Errorf("expected stale version 1 to be REJECTED, but it was applied")
	}

	snapshot, err := repository.Get(context, ticketID)
	if err != nil {
		t.Fatalf("failed to read final snapshot: %v", err)
	}
	if snapshot.Quantity != 2 || snapshot.Version != 2 {
		t.Errorf("expected final state {quantity:2 version:2}, got {quantity:%d version:%d}", snapshot.Quantity, snapshot.Version)
	}
}

func TestApplyIfNewer_NewerUpdateStillApplied(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	context := context.Background()
	ticketID := seedTicket(t, database)
	repository := NewRepository(database)

	if _, err := repository.ApplyIfNewer(context, StockUpdate{TicketID: ticketID, Quantity: 5, Version: 1}); err != nil {
		t.Fatalf("failed to apply version 1: %v", err)
	}

	applied, err := repository.ApplyIfNewer(context, StockUpdate{TicketID: ticketID, Quantity: 2, Version: 2})
	if err != nil {
		t.Fatalf("failed to apply version 2: %v", err)
	}
	if !applied {
		t.Errorf("expected newer version 2 to be applied")
	}

	snapshot, err := repository.Get(context, ticketID)
	if err != nil {
		t.Fatalf("failed to read final snapshot: %v", err)
	}
	if snapshot.Quantity != 2 || snapshot.Version != 2 {
		t.Errorf("expected final state {quantity:2 version:2}, got {quantity:%d version:%d}", snapshot.Quantity, snapshot.Version)
	}
}
