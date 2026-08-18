# Concert-Ticketing

## Flow Diagrams

- 1. Race condition -- atomic `UPDATE ... WHERE stock > 0`
- 2. High traffic -- buffered queue + fixed worker pool
- 3. External API integration -- outbox pattern + retry dengan backoff
- 4. Duplicate request -- `UNIQUE` + `ON CONFLICT DO NOTHING`
- 5. Data synchronization -- version-gated upsert (`WHERE version < EXCLUDED.version`)

## Cara Menjalankan

1. Nyalakan Postgres, pastikan Docker Desktop sudah berjalan:

   --------------------
   docker compose up -d
   --------------------

2. (Opsional, untuk mengamati mekanisme retry pada Section 3) jalankan
   mock accounting service pada terminal terpisah:

   --------------------
   go run ./cmd/mock-accounting
   --------------------

   Service ini berjalan pada port `:9000` dan dikonfigurasi untuk
   mengembalikan error 500 pada dua percobaan pertama setiap transaksi.
   Jika tidak dijalankan, worker Section 3 tetap berjalan normal namun
   akan terus mengalami kegagalan koneksi.

3. Jalankan server:

   --------------------
   go run ./cmd/server
   --------------------

   Server berjalan pada port `:8080` dengan tiga endpoint:
   - `POST /tickets/{id}/purchase` -- melakukan pembelian tiket, dengan
     body `{"buyer_name": "..."}`.
   - `POST /webhooks/payment` -- mensimulasikan notifikasi pembayaran
     dari pihak ketiga, dengan body
     `{"external_payment_id": "...", "transaction_id": ..., "amount": ..., "status": "..."}`.
   - `POST /sync/stock` -- mensimulasikan pembaruan stok dari sistem
     lain, dengan body `{"ticket_id": ..., "quantity": ..., "version": ...}`.

   Contoh pembelian tiket (ganti nilai `1` dengan id tiket dari seed data):

   --------------------
   curl -X POST http://localhost:8080/tickets/1/purchase \
     -H "Content-Type: application/json" \
     -d "{\"buyer_name\": \"Budi\"}"
   --------------------

   Respons berhasil: `200 OK` disertai `transaction_id`. Apabila stok
   habis: `409 Conflict` disertai `{"error": "ticket sold out"}`.

## Cara Testing

--------------------
docker compose up -d
go test ./... -v -count=1
--------------------

- `TestConcurrentPurchase_OnlyOneSucceeds` (Section 1) -- dua
  pembelian tiket dilakukan secara bersamaan dengan stok=1, hanya satu
  yang boleh berhasil.
- `TestHighTraffic_AllTransactionsSaved` (Section 2) -- 10.000
  pembelian dilakukan secara bersamaan, seluruhnya harus berhasil dan
  tersimpan.
- `TestWorker_RetriesUntilThirdPartyRecovers` (Section 3) -- accounting
  service gagal dua kali sebelum akhirnya berhasil, worker harus tetap
  mengirimkan data melalui mekanisme retry.
- `TestWorker_GivesUpAfterMaxAttempts` (Section 3) -- accounting
  service gagal secara terus-menerus, worker harus berhenti mencoba
  setelah mencapai `maxAttempts` (bukan retry tanpa batas).
- `TestHandleWebhook_DuplicateRequestsNotDuplicated` (Section 4) --
  sepuluh webhook identik dikirim secara bersamaan, hanya satu baris
  yang boleh tersimpan.
- `TestApplyIfNewer_OutOfOrderUpdateIsIgnored` (Section 5) -- update
  dengan versi lebih lama yang tiba belakangan harus ditolak.
- `TestApplyIfNewer_NewerUpdateStillApplied` (Section 5) -- update
  dengan versi lebih baru yang tiba secara normal tetap diterapkan.

Apabila Postgres belum berjalan, test akan otomatis di-skip dengan
pesan yang jelas (bukan gagal/error).

## Desain

Setiap section merupakan satu package pada `internal/`, dengan pola
yang konsisten: `model.go` (struct data) + `repository.go` (query SQL)
+ `handler.go` (HTTP layer), ditambah `service.go` khusus pada
package `ticket` untuk keperluan orkestrasi. Mekanisme inti dari
masing-masing section adalah sebagai berikut:

- **`internal/ticket`** (Section 1 & 2) -- proses pembelian dilakukan
  melalui atomic conditional UPDATE (`stock = stock - 1 WHERE stock > 0`,
  satu perintah SQL, sehingga tidak terdapat celah race condition),
  dibungkus dalam satu DB transaction bersama insert transaksi dan
  outbox row. Request masuk melalui worker pool (buffered channel +
  goroutine tetap) agar jumlah koneksi database tidak membengkak saat
  traffic tinggi.
- **`internal/accounting`** (Section 3) -- menerapkan outbox pattern:
  outbox row disimpan pada transaksi yang sama dengan pembelian,
  kemudian background worker melakukan polling dan mengirimkan data ke
  accounting service dengan mekanisme retry serta exponential backoff,
  dan masuk ke status dead-letter setelah mencapai `maxAttempts` agar
  tidak melakukan retry tanpa batas waktu.
- **`internal/payment`** (Section 4) -- menerapkan idempotency key
  (`external_payment_id` dari pihak ketiga) dengan UNIQUE constraint
  serta `INSERT ... ON CONFLICT DO NOTHING`, yang ditangani secara
  atomic oleh database sehingga tetap aman meskipun request identik
  diterima secara bersamaan.
- **`internal/stocksync`** (Section 5) -- menerapkan version-gated
  upsert (`INSERT ... ON CONFLICT DO UPDATE ... WHERE version < EXCLUDED.version`),
  dimana update hanya diterapkan apabila versinya lebih baru
  dibandingkan data yang tersimpan, sehingga urutan kedatangan melalui
  jaringan tidak memengaruhi hasil akhir.
