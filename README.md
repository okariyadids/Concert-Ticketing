# Concert-Ticketing

Backend ticket booking system buat konser skala besar. Ini scaffold
awal (belum ada Section 1-5) -- go module + koneksi ke Postgres +
HTTP server kosong, jadi titik mulai buat tiap section nanti.

## Cara menjalankan

1. Nyalain Postgres lokal (butuh Docker Desktop jalan):
   ```
   docker compose up -d
   ```
   Postgres di-expose ke host port **5433** (bukan 5432 default) biar
   gak bentrok kalau di komputer kamu udah ada Postgres native yang
   jalan di 5432.

2. Jalankan server:
   ```
   go run ./cmd/server
   ```
   Server listen di `:8080`. Belum ada route terdaftar (scaffold
   kosong), jadi semua request bakal dapat `404`.
