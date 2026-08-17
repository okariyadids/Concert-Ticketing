package ticket

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestHighTraffic_AllTransactionsSaved(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	context := context.Background()

	const totalTransactions = 10000

	var ticketID int64
	err := database.QueryRowContext(context,
		`INSERT INTO tickets (name, stock, price) VALUES ('Load Test Ticket', $1, 50000) RETURNING id`,
		totalTransactions,
	).Scan(&ticketID)
	if err != nil {
		t.Fatalf("failed to seed ticket: %v", err)
	}
	ticketIDStr := strconv.FormatInt(ticketID, 10)

	service := NewService(NewRepository(database))
	queue := NewQueue(service, 50, 1000)
	handler := NewHandler(queue)

	var waitGroup sync.WaitGroup
	var successCount int64

	start := time.Now()

	for transactionIndex := 0; transactionIndex < totalTransactions; transactionIndex++ {
		waitGroup.Add(1)
		go func(buyerNumber int) {
			defer waitGroup.Done()

			body := strings.NewReader(`{"buyer_name": "buyer-` + strconv.Itoa(buyerNumber) + `"}`)
			httpRequest := httptest.NewRequest(http.MethodPost, "/tickets/"+ticketIDStr+"/purchase", body)
			httpRequest.SetPathValue("id", ticketIDStr)
			responseRecorder := httptest.NewRecorder()

			handler.HandlePurchase(responseRecorder, httpRequest)

			if responseRecorder.Code == http.StatusOK {
				atomic.AddInt64(&successCount, 1)
			} else {
				t.Errorf("buyer %d got status %d: %s", buyerNumber, responseRecorder.Code, responseRecorder.Body.String())
			}
		}(transactionIndex)
	}

	waitGroup.Wait()
	elapsed := time.Since(start)

	if successCount != totalTransactions {
		t.Errorf("expected all %d transactions to succeed, got %d", totalTransactions, successCount)
	}
	if elapsed > time.Minute {
		t.Errorf("processing took too long: %s (harus di bawah 1 menit)", elapsed)
	}

	var savedCount int
	err = database.QueryRowContext(context,
		`SELECT COUNT(*) FROM transactions WHERE ticket_id = $1`, ticketID,
	).Scan(&savedCount)
	if err != nil {
		t.Fatalf("failed to count saved transactions: %v", err)
	}
	if savedCount != totalTransactions {
		t.Errorf("expected %d rows saved in transactions table, got %d", totalTransactions, savedCount)
	}

	t.Logf("processed %d transactions in %s", totalTransactions, elapsed)
}
