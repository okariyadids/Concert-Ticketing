package accounting

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/okariyadids/concert-bnl/internal/ticket"
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

func waitUntilStatus(t *testing.T, database *sql.DB, outboxID int64, expectedStatus string, timeout time.Duration) string {
	t.Helper()

	deadline := time.Now().Add(timeout)
	var currentStatus string
	for time.Now().Before(deadline) {
		err := database.QueryRow(`SELECT status FROM accounting_outbox WHERE id = $1`, outboxID).Scan(&currentStatus)
		if err != nil {
			t.Fatalf("failed to read outbox status: %v", err)
		}
		if currentStatus == expectedStatus {
			return currentStatus
		}
		time.Sleep(20 * time.Millisecond)
	}
	return currentStatus
}

func seedPurchase(t *testing.T, database *sql.DB) (transactionID int64, outboxID int64) {
	t.Helper()

	repository := ticket.NewRepository(database, NewRepository(database))
	service := ticket.NewService(repository)

	var ticketID int64
	err := database.QueryRow(
		`INSERT INTO tickets (name, stock, price) VALUES ('Accounting Test Ticket', 10, 100000) RETURNING id`,
	).Scan(&ticketID)
	if err != nil {
		t.Fatalf("failed to seed ticket: %v", err)
	}

	transactionID, err = service.Purchase(context.Background(), ticketID, "buyer-accounting-test")
	if err != nil {
		t.Fatalf("failed to seed purchase: %v", err)
	}

	err = database.QueryRow(
		`SELECT id FROM accounting_outbox WHERE transaction_id = $1`, transactionID,
	).Scan(&outboxID)
	if err != nil {
		t.Fatalf("failed to find seeded outbox row: %v", err)
	}
	return transactionID, outboxID
}

func TestWorker_RetriesUntilThirdPartyRecovers(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	var mutex sync.Mutex
	requestCount := 0

	fakeThirdParty := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		mutex.Lock()
		requestCount++
		currentCount := requestCount
		mutex.Unlock()

		if currentCount <= 2 {
			responseWriter.WriteHeader(http.StatusInternalServerError)
			return
		}
		responseWriter.WriteHeader(http.StatusOK)
	}))
	defer fakeThirdParty.Close()

	_, outboxID := seedPurchase(t, database)

	client := NewClient(fakeThirdParty.URL, time.Second)
	worker := NewWorker(NewRepository(database), client,
		20*time.Millisecond,
		10,
		5,
		10*time.Millisecond,
		200*time.Millisecond,
	)

	context, cancel := context.WithCancel(context.Background())
	defer cancel()
	go worker.Start(context)

	status := waitUntilStatus(t, database, outboxID, "sent", 3*time.Second)
	if status != "sent" {
		t.Fatalf("expected outbox status 'sent', got %q", status)
	}

	mutex.Lock()
	finalCount := requestCount
	mutex.Unlock()
	if finalCount != 3 {
		t.Errorf("expected exactly 3 requests to third party (2 failures + 1 success), got %d", finalCount)
	}
}

func TestWorker_GivesUpAfterMaxAttempts(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	var requestCount int64
	fakeThirdParty := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		atomic.AddInt64(&requestCount, 1)
		responseWriter.WriteHeader(http.StatusInternalServerError)
	}))
	defer fakeThirdParty.Close()

	_, outboxID := seedPurchase(t, database)

	const maxAttempts = 3
	client := NewClient(fakeThirdParty.URL, time.Second)
	worker := NewWorker(NewRepository(database), client,
		20*time.Millisecond,
		10,
		maxAttempts,
		10*time.Millisecond,
		50*time.Millisecond,
	)

	context, cancel := context.WithCancel(context.Background())
	defer cancel()
	go worker.Start(context)

	status := waitUntilStatus(t, database, outboxID, "failed", 3*time.Second)
	if status != "failed" {
		t.Fatalf("expected outbox status 'failed' after giving up, got %q", status)
	}

	time.Sleep(100 * time.Millisecond)
	countAfterDead := atomic.LoadInt64(&requestCount)
	time.Sleep(200 * time.Millisecond)
	countLater := atomic.LoadInt64(&requestCount)

	if countAfterDead != countLater {
		t.Errorf("worker kept retrying after being marked dead: %d requests before, %d after", countAfterDead, countLater)
	}
	if countLater != maxAttempts {
		t.Errorf("expected exactly %d requests total, got %d", maxAttempts, countLater)
	}
}
