Menjalankan server + tes manual:

Jalankan Postgres:

docker compose down -v
docker compose up -d

Jalankan server:

go run ./cmd/server
Listen di :8080. Seed data otomatis dari migrations/001_init.sql: 1 tiket VIP, stock 1.

Tes race condition manual — buka 2 terminal, jalankan bersamaan:

curl -X POST http://localhost:8080/tickets/1/purchase -H "Content-Type: application/json" -d "{\"buyer_name\":\"Bayu\"}"
curl -X POST http://localhost:8080/tickets/1/purchase -H "Content-Type: application/json" -d "{\"buyer_name\":\"Satria\"}"
1 harus 200 OK + transaction_id, 1 lagi 409 Conflict + {"error":"ticket sold out"}.

Tes otomatis:

go test ./... -v -run TestConcurrentPurchase -count=1