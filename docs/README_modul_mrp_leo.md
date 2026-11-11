# Modul MRP Engine & Pembelian (Leo) — Tahap 1

Branch: `git checkout -b modul-mrp-leo`

## Yang sudah ada (tahap 1)
| File | Isi |
|---|---|
| `internal/models/mrp_production.go` | struct ProductionRun, ShoppingListItem, Recipe/Item bantu, Purchase |
| `internal/repository/mrp_repo.go` | query pesanan LOCKED, resep, item, simpan `production_runs` + `shopping_lists` |
| `internal/repository/purchase_repo.go` | transaksi nota beli + mutasi stok IN (sementara) |
| `internal/service/mrp_service.go` | **Exploding BOM rekursif**, net requirement, CEIL, buffer 5% opsional |
| `internal/service/purchase_service.go` | konversi satuan beli→pakai, moving average, 1 transaksi per nota |
| `internal/handler/mrp_handler.go` | 5 endpoint + mapping error ke status HTTP |
| `internal/service/mrp_service_test.go` | unit test logika inti (tanpa DB) |
| `docs/leo_dummy_seed.sql` | data dummy agar bisa dites tanpa menunggu modul teman |

Semua file baru (tidak mengedit file milik orang lain) supaya merge conflict minimal.

## Menyambungkan ke main.go (minimal)
```go
mrpSvc := service.NewMRPService(repository.NewMRPRepo(db))
purSvc := service.NewPurchaseService(repository.NewPurchaseRepo(db))
handler.NewMRPHandler(mrpSvc, purSvc).RegisterRoutes(mux, nil) // nil = tanpa RBAC dulu; isi dengan guard Rajen nanti
```
Syarat: `go.mod` memakai Go 1.22+ (pola route `"POST /api/..."`) dan driver `github.com/go-sql-driver/mysql`.

## Cara tes
1. Jalankan `migrations/001_init_12_tables.sql`, lalu `docs/leo_dummy_seed.sql`.
2. `go test ./internal/service/...` (tanpa DB).
3. Postman:
   - `POST /api/mrp/explode` body `{"production_date":"2026-11-20","apply_buffer":false}`
   - `GET /api/mrp/shopping-lists?production_run_id=<id dari respons>`
   - `POST /api/purchases`
     ```json
     {"invoice_no":"INV-001","supplier":"Pasar Johar","purchase_date":"2026-11-19",
      "items":[{"item_id":9104,"qty_purchase_unit":98,"unit_price":40000}]}
     ```
   Cek `items.stock_qty`, `items.moving_avg_cost`, `material_price_logs`, `stock_movements` ikut berubah.

## Asumsi (cek dengan tim)
1. Pesanan diambil dengan `orders.event_date = production_date` (ERD tidak punya kolom tanggal produksi terpisah).
2. Bahan yang stoknya cukup **tidak** dimasukkan ke `shopping_lists` (sesuai SRS). Jumlahnya ada di `skipped_stock_cukup`.
3. Menjalankan explode ulang pada tanggal yang sama menimpa daftar belanja sebelumnya; ditolak (409) bila `production_runs.status = 'DONE'`.
4. Nilai status `production_runs`: `PLANNED` / `DONE`.
5. Nama kolom sudah dicocokkan dengan `001_init_12_tables.sql`. Kolom `purchases.cost_applied` (NOT NULL, tidak ada di ERD) saya isi sama dengan `total_amount`; konfirmasi maknanya ke Rajen.

## Sementara / harus diganti saat modul teman masuk
- `PurchaseRepo.ApplyInbound` menduplikasi mutasi IN milik Dzaki → ganti dengan fungsi Dzaki setelah merge.
- RBAC: oper guard milik Rajen ke `RegisterRoutes`.
- Antarmuka HTML (`web/templates/mrp`) dikerjakan terakhir sesuai pedoman.
