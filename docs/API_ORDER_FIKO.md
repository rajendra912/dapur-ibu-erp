# 📅 Modul Order Management & Automasi Cutoff H-2 (Fiko)

Dokumentasi teknis untuk bagian **Fiko Hilmy Dzaky** — penerimaan pesanan pelanggan dan
penguncian otomatis H-2 pada ERP Katering "Dapur Ibu".

Branch: `modul-order-fiko`

---

## 1. File yang Dibuat

| File | Isi |
| --- | --- |
| `internal/models/order.go` | Struct `Order`, `OrderItem`, payload request, filter, rekap, dan hasil cutoff |
| `internal/repository/order_repo.go` | Seluruh query MySQL untuk `orders` & `order_items` |
| `internal/service/order_service.go` | Validasi, hitung `cutoff_date`, proteksi edit, snapshot HPP, rekap |
| `internal/handler/order_handler.go` | Endpoint HTTP + `RegisterOrderRoutes()` |
| `internal/middleware/cutoff_guard.go` | Middleware penolak edit pesanan LOCKED/DONE/lewat cutoff |
| `internal/worker/cutoff_cron.go` | Goroutine penjadwal penguncian tiap pukul 00:00 |
| `internal/service/order_service_test.go` | Unit test cutoff, status, dan parser tanggal |
| `internal/worker/cutoff_cron_test.go` | Unit test jadwal tengah malam |
| `cmd/server/main.go` | 2 baris tambahan: daftarkan rute + jalankan worker |

Struktur mengikuti pola tim: **Handler → Service → Repository → Database**.

---

## 2. Endpoint

| Method | Endpoint | Fungsi |
| --- | --- | --- |
| `POST` | `/api/orders` | Input pesanan baru (header + detail menu) |
| `GET` | `/api/orders` | Daftar pesanan (`?status=&event_date=&q=&limit=`) |
| `GET` | `/api/orders/{id}` | Detail pesanan + rincian menu |
| `PUT` / `PATCH` | `/api/orders/{id}` | Ubah pesanan (ditolak jika LOCKED/DONE/lewat cutoff) |
| `DELETE` | `/api/orders/{id}` | Hapus pesanan (ditolak jika LOCKED/DONE/lewat cutoff) |
| `GET` | `/api/orders/summary?event_date=YYYY-MM-DD` | Rekap total porsi per menu pada satu tanggal acara |
| `POST` | `/api/cron/cutoff` | Picu penguncian manual (untuk demo ke dosen) |

Semua respons error berformat `{"error": "..."}`.

---

## 3. Contoh Request

### 3.1 Buat Pesanan

`POST /api/orders`

```json
{
  "order_num": "ORD-20261120-001",
  "customer": "Bu Sri",
  "event_date": "2026-11-20",
  "status": "CONFIRMED",
  "items": [
    { "menu_item_id": 6, "portions": 100, "selling_price": 35000 },
    { "menu_item_id": 7, "portions": 50,  "selling_price": 40000 }
  ]
}
```

Respons `201 Created`:

```json
{
  "id": 1,
  "order_num": "ORD-20261120-001",
  "customer": "Bu Sri",
  "event_date": "2026-11-20",
  "cutoff_date": "2026-11-19T00:00:00+07:00",
  "status": "CONFIRMED",
  "locked_hpp_per_portion": 0,
  "total_portions": 150,
  "total_value": 5500000,
  "editable": true,
  "items": [
    { "id": 1, "order_id": 1, "menu_item_id": 6, "menu_name": "Nasi Kotak A", "portions": 100, "selling_price": 35000, "subtotal": 3500000 },
    { "id": 2, "order_id": 1, "menu_item_id": 7, "menu_name": "Nasi Kotak B", "portions": 50, "selling_price": 40000, "subtotal": 2000000 }
  ]
}
```

Catatan:
* `order_num` opsional; jika kosong sistem membuat `ORD-YYYYMMDD-HHMMSS-mmm`.
* `status` awal hanya boleh `DRAFT` (default) atau `CONFIRMED`.
* Hanya item bertipe `MENU` dan masih aktif yang boleh dipesan.
* `cutoff_date` **tidak boleh diisi manual** — dihitung otomatis.

### 3.2 Ubah Pesanan (partial update)

`PUT /api/orders/1`

```json
{
  "items": [
    { "menu_item_id": 6, "portions": 120, "selling_price": 35000 }
  ]
}
```

Field yang tidak dikirim (`customer`, `event_date`, `status`) akan dipertahankan.
Mengirim `items` akan **mengganti seluruh** detail menu pesanan tersebut.

### 3.3 Rekap porsi per tanggal acara

`GET /api/orders/summary?event_date=2026-11-20&status=LOCKED`

```json
{
  "event_date": "2026-11-20",
  "total_orders": 3,
  "total_portions": 520,
  "total_value": 18200000,
  "menus": [
    { "menu_item_id": 6, "menu_name": "Nasi Kotak A", "total_portions": 400, "order_count": 2 },
    { "menu_item_id": 7, "menu_name": "Nasi Kotak B", "total_portions": 120, "order_count": 1 }
  ]
}
```

Endpoint ini bisa dipakai Leo sebagai bahan **Agregasi Level 0** modul MRP dan oleh
dashboard untuk menampilkan total porsi produksi.

### 3.4 Picu Cutoff Manual

`POST /api/cron/cutoff`

```json
{
  "message": "cutoff selesai dijalankan",
  "result": {
    "locked_count": 1,
    "orders": [
      {
        "order_id": 1,
        "order_num": "ORD-20261120-001",
        "event_date": "2026-11-20",
        "total_portions": 150,
        "locked_hpp_per_portion": 14.70
      }
    ],
    "run_at": "2026-11-19T00:00:00+07:00"
  }
}
```

Aman dipanggil berulang kali (idempotent): pesanan yang sudah `LOCKED` tidak diproses lagi.

---

## 4. Aturan Bisnis yang Diimplementasikan

### 4.1 Cutoff otomatis H-2

```
cutoff_date = event_date - 1 hari, pukul 00:00:00
```

* Acara **2026-11-20** → `cutoff_date` **2026-11-19 00:00:00**.
* Artinya admin bebas mengubah pesanan sampai **H-2 pukul 23:59:59**.
* `editable` pada JSON dihitung real-time:
  `status != LOCKED/DONE && now < cutoff_date`.

### 4.2 Proteksi edit (berlapis)

1. **Middleware** `GuardOrderEditable` pada rute `PUT/PATCH/DELETE` → tolak permintaan
   sebelum masuk handler.
2. **Service** mengecek ulang status + cutoff di dalam proses simpan (double guard).

Pesan error saat terkunci (sesuai jobdesc):

```json
{ "error": "Pesanan sudah terkunci (Melewati batas Cutoff H-2) dan tidak dapat diubah!" }
```

HTTP Status: **400 Bad Request**. Pesanan berstatus `DONE` juga tidak bisa diubah
dengan pesan `"Pesanan sudah berstatus DONE dan tidak dapat diubah!"`.

### 4.3 Worker Cron-Job

`internal/worker/cutoff_cron.go` berjalan sebagai **goroutine** di latar belakang:

* Saat server dinyalakan → langsung *catch-up* mengunci pesanan yang terlewat
  (misalnya server sempat mati melewati tengah malam).
* Setiap **pukul 00:00 waktu laptop** → memanggil `service.LockDueOrders()`.
* Query: `status IN ('DRAFT','CONFIRMED') AND cutoff_date <= NOW()` dengan `FOR UPDATE`
  agar worker dan endpoint manual tidak memproses baris yang sama.
* Matikan worker saat demo manual dengan environment variable `CUTOFF_WORKER=off`.

### 4.4 Snapshot HPP (`locked_hpp_per_portion`)

Saat pesanan dikunci, sistem menghitung HPP per porsi dari menu yang dipesan dengan
**rata-rata tertimbang porsi**:

```
HPP pesanan = Σ (HPP per porsi menu × porsi menu) / total porsi
```

Sumber HPP tiap menu, berurutan:

1. `items.moving_avg_cost` (hasil roll-up modul Resep / master bahan).
2. `recipes.std_hpp` terakhir bila `moving_avg_cost` masih 0.
3. Hitung rekursif `recipe_bom_items` → komponen → bahan mentah (`RAW`) memakai
   `moving_avg_cost` bahan, dibagi `batch_yield` (maksimal 6 level, aman dari resep melingkar).

Contoh pada data uji: menu Nasi Kotak A = 12.05 (beras 150 gr × 0.012 + Ayam Goreng
10.25 hasil hitung rekursif), Nasi Kotak B = 20.00, dengan 100 + 50 porsi →
snapshot **14.70**.

---

## 5. Integrasi dengan Modul Lain

| Modul | Kebutuhan |
| --- | --- |
| **Rajen (Auth/RBAC)** | Bungkus `RegisterOrderRoutes` dengan middleware role: `ADMIN` untuk create/update/delete, `OWNER` untuk semua `GET`, `KITCHEN`/`WAREHOUSE` minimal `GET` rekap. Endpoint `POST /api/cron/cutoff` sebaiknya hanya untuk `OWNER`/internal. |
| **Syarif (Resep)** | Snapshot HPP memakai `items.moving_avg_cost` / `recipes.std_hpp`; jika belum ada, service menghitung sendiri dari BOM. Tidak ada ketergantungan kompilasi. |
| **Leo (MRP)** | Gunakan `GET /api/orders/summary?event_date=...&status=LOCKED` atau query `orders` + `order_items` langsung. |
| **Dzaki (Inventory)** | Tidak ada pemanggilan stok dari modul pesanan; stok baru bergerak di modul gudang. |

### Catatan `config/database.go`

DSN ditambah `&loc=Local`:

```go
dsn := "root:@tcp(127.0.0.1:3306)/dapur_ibu_db?parseTime=true&loc=Local"
```

Tanpa `loc=Local`, driver MySQL menulis `cutoff_date` dalam UTC ke kolom `DATETIME`,
sehingga perbandingan dengan `NOW()` (waktu lokal MySQL) bisa meleset beberapa jam dan
cutoff tidak terkunci tepat waktu. Perubahan ini hanya 1 baris dan tidak memengaruhi
modul lain.

---

## 6. Cara Menjalankan & Mengetes

```bash
# 1. Pastikan MySQL menyala dan database sudah di-import
#    (migrations/001_init_12_tables.sql)

# 2. Jalankan server
go run cmd/server/main.go

# 3. Tes cepat
curl http://localhost:8080/api/health
```

* Import `docs/postman_order_fiko.json` ke Postman untuk mencoba semua endpoint.
* Demo cutoff tanpa menunggu tengah malam:
  1. Buat pesanan.
  2. Ubah `cutoff_date` menjadi waktu lampau (HeidiSQL):
     `UPDATE orders SET cutoff_date = DATE_SUB(NOW(), INTERVAL 1 MINUTE) WHERE id = 1;`
  3. `POST /api/cron/cutoff` → pesanan menjadi `LOCKED` + snapshot HPP terisi.
  4. Coba `PUT /api/orders/1` → harus `400 Bad Request`.

### Unit test

```bash
go test ./internal/service ./internal/worker -v
```

---

## 7. Hasil Pengujian (E2E)

Diuji dengan MySQL 8.4 + data dummy (bahan mentah, bumbu, komponen, 2 menu, 3 resep):
22 skenario lulus, meliputi:

* CRUD pesanan, validasi tanggal/menu/porsi, nomor pesanan duplikat.
* Perhitungan `cutoff_date` otomatis (termasuk pergantian bulan/tahun).
* Penolakan edit setelah cutoff, setelah `LOCKED`, dan setelah `DONE`.
* Cutoff manual + snapshot HPP rata-rata tertimbang (14.70 sesuai perhitungan manual).
* Cutoff idempotent (dipanggil dua kali → 0 pesanan).
* Worker *catch-up* saat server restart: 1 pesanan otomatis terkunci + HPP tersimpan.
