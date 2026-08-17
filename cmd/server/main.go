package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/okariyadids/concert-bnl/internal/accounting"
	"github.com/okariyadids/concert-bnl/internal/ticket"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://concert:concert@localhost:5433/concert_bnl?sslmode=disable"
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("failed to open db connection: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}

	const workerCount = 50
	const queueSize = 1000
	db.SetMaxOpenConns(workerCount)
	db.SetMaxIdleConns(workerCount)

	accountingURL := os.Getenv("ACCOUNTING_API_URL")
	if accountingURL == "" {
		accountingURL = "http://localhost:9000"
	}

	accountingRepository := accounting.NewRepository(db)
	accountingClient := accounting.NewClient(accountingURL, 5*time.Second)
	accountingWorker := accounting.NewWorker(
		accountingRepository,
		accountingClient,
		2*time.Second,
		20,
		5,
		2*time.Second,
		time.Minute,
	)

	accountingWorkerContext, cancelAccountingWorker := context.WithCancel(context.Background())
	defer cancelAccountingWorker()
	go accountingWorker.Start(accountingWorkerContext)

	repository := ticket.NewRepository(db, accountingRepository)
	service := ticket.NewService(repository)
	queue := ticket.NewQueue(service, workerCount, queueSize)
	handler := ticket.NewHandler(queue)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /tickets/{id}/purchase", handler.HandlePurchase)

	log.Println("server listening on :8080")
	log.Printf("sending transactions to accounting service at %s", accountingURL)
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
