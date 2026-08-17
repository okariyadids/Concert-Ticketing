package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"

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

	repository := ticket.NewRepository(db)
	service := ticket.NewService(repository)
	handler := ticket.NewHandler(service)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /tickets/{id}/purchase", handler.HandlePurchase)

	log.Println("server listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
