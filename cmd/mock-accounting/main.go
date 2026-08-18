package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"
)

func main() {
	var mutex sync.Mutex
	attemptsPerTransaction := map[int64]int{}

	http.HandleFunc("POST /transaction", func(responseWriter http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			http.Error(responseWriter, "bad request", http.StatusBadRequest)
			return
		}

		var payload struct {
			TransactionID int64 `json:"transaction_id"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			http.Error(responseWriter, "bad request", http.StatusBadRequest)
			return
		}

		mutex.Lock()
		attemptsPerTransaction[payload.TransactionID]++
		attempt := attemptsPerTransaction[payload.TransactionID]
		mutex.Unlock()

		if attempt <= 2 {
			log.Printf("transaction %d attempt %d -> 500 (simulated failure)", payload.TransactionID, attempt)
			http.Error(responseWriter, "internal server error", http.StatusInternalServerError)
			return
		}

		log.Printf("transaction %d attempt %d -> 200 (accepted)", payload.TransactionID, attempt)
		responseWriter.WriteHeader(http.StatusOK)
	})

	log.Println("mock accounting service listening on :9000")
	if err := http.ListenAndServe(":9000", nil); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
