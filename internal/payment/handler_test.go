package payment

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
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

	database, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := database.Ping(); err != nil {
		t.Skipf("postgres tidak bisa dijangkau (%v). Jalankan `docker compose up -d` dulu.", err)
	}
	return database
}

func seedTransaction(t *testing.T, database *sql.DB) int64 {
	t.Helper()

	var ticketID int64
	err := database.QueryRow(
		`INSERT INTO tickets (name, stock, price) VALUES ('Payment Test Ticket', 10, 100000) RETURNING id`,
	).Scan(&ticketID)
	if err != nil {
		t.Fatalf("failed to seed ticket: %v", err)
	}

	var transactionID int64
	err = database.QueryRow(
		`INSERT INTO transactions (ticket_id, buyer_name, status) VALUES ($1, 'buyer-payment-test', 'success') RETURNING id`,
		ticketID,
	).Scan(&transactionID)
	if err != nil {
		t.Fatalf("failed to seed transaction: %v", err)
	}
	return transactionID
}

func TestHandleWebhook_DuplicateRequestsNotDuplicated(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	transactionID := seedTransaction(t, database)

	handler := NewHandler(NewRepository(database))

	const totalRequests = 10
	const externalPaymentID = "PAY-DUPLICATE-TEST-001"
	body := `{"external_payment_id": "` + externalPaymentID + `", "transaction_id": ` +
		strconv.FormatInt(transactionID, 10) + `, "amount": 150000, "status": "paid"}`

	var waitGroup sync.WaitGroup
	var okCount int64

	for requestIndex := 0; requestIndex < totalRequests; requestIndex++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()

			httpRequest := httptest.NewRequest(http.MethodPost, "/webhooks/payment", strings.NewReader(body))
			responseRecorder := httptest.NewRecorder()

			handler.HandleWebhook(responseRecorder, httpRequest)

			if responseRecorder.Code == http.StatusOK {
				atomic.AddInt64(&okCount, 1)
			} else {
				t.Errorf("unexpected status %d: %s", responseRecorder.Code, responseRecorder.Body.String())
			}
		}()
	}

	waitGroup.Wait()

	if okCount != totalRequests {
		t.Errorf("expected all %d requests to return 200 OK, got %d", totalRequests, okCount)
	}

	var savedCount int
	err := database.QueryRow(
		`SELECT COUNT(*) FROM transaction_payments WHERE external_payment_id = $1`, externalPaymentID,
	).Scan(&savedCount)
	if err != nil {
		t.Fatalf("failed to count saved payments: %v", err)
	}
	if savedCount != 1 {
		t.Errorf("expected exactly 1 saved payment row, got %d", savedCount)
	}
}
