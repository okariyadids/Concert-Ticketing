package ticket

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://concert:concert@localhost:5433/concert_bnl?sslmode=disable"
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Skipf("postgres tidak bisa dijangkau (%v). Jalankan `docker compose up -d` dulu.", err)
	}
	return db
}

func TestConcurrentPurchase_OnlyOneSucceeds(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	var ticketID int64
	err := db.QueryRowContext(ctx,
		`INSERT INTO tickets (name, stock, price) VALUES ('Test VIP Ticket', 1, 100000) RETURNING id`,
	).Scan(&ticketID)
	if err != nil {
		t.Fatalf("failed to seed ticket: %v", err)
	}

	service := NewService(NewRepository(db))

	const totalBuyers = 2
	var successCount int64
	var soldOutCount int64
	var wg sync.WaitGroup

	for i := 0; i < totalBuyers; i++ {
		wg.Add(1)
		go func(buyerNumber int) {
			defer wg.Done()

			_, err := service.Purchase(ctx, ticketID, "buyer-"+strconv.Itoa(buyerNumber))
			switch {
			case err == nil:
				atomic.AddInt64(&successCount, 1)
			case errors.Is(err, ErrSoldOut):
				atomic.AddInt64(&soldOutCount, 1)
			default:
				t.Errorf("unexpected error: %v", err)
			}
		}(i)
	}

	wg.Wait()

	if successCount != 1 {
		t.Errorf("expected exactly 1 successful purchase, got %d", successCount)
	}
	if soldOutCount != totalBuyers-1 {
		t.Errorf("expected %d sold-out responses, got %d", totalBuyers-1, soldOutCount)
	}
}
