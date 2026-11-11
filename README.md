# 🍲 Dapur Ibu ERP - Sistem Perencanaan Produksi Katering (Kelompok 7)

Selamat datang di repositori resmi proyek **ERP Katering "Dapur Ibu"**. Sistem ini dirancang untuk menangani operasional katering berskala besar (ratusan hingga ribuan porsi), mulai dari pencatatan pesanan pelanggan, manajemen resep bertingkat (*Multi-level Bill of Materials*), penguncian pesanan otomatis (*Cutoff H-2*), perhitungan daftar belanja pasar (*MRP Exploding BOM*), hingga evaluasi Harga Pokok Produksi (HPP) menggunakan metode *Weighted Moving Average*.

Dokumen ini adalah **"Kitab Suci" (Single Source of Truth)** bagi seluruh anggota Kelompok 7. Baca dokumen ini dari atas hingga bawah sebelum menulis satu baris kode pun!

---

## 🛠️ 1. Tech Stack & Arsitektur
*   **Bahasa Pemrograman (Backend):** Go (Golang) `v1.20+`
*   **Router HTTP:** `gorilla/mux`
*   **Basis Data (Database):** MySQL 8.0 (Storage Engine: `InnoDB` - Mendukung transaksi ACID)
*   **Driver Database:** `go-sql-driver/mysql`
*   **Frontend (Nanti):** HTML5, CSS (Bootstrap 5 / Tailwind), JavaScript Vanilla (Fetch API)
*   **Pola Arsitektur:** Modular Clean Architecture (Handler $\rightarrow$ Service $\rightarrow$ Repository $\rightarrow$ Database)

---

## 💻 2. Persyaratan Sistem (Prerequisites)
Sebelum melakukan *clone* proyek ini, pastikan di laptop masing-masing sudah terinstal aplikasi berikut:
1.  **Go (Golang):** Cek di terminal dengan mengetik `go version`.
2.  **Git:** Cek di terminal dengan mengetik `git --version`.
3.  **Database Server:** **Laragon** (Sangat disarankan) atau **XAMPP**.
4.  **Database Client:** **HeidiSQL** (bawaan Laragon), **DBeaver**, atau **phpMyAdmin**.
5.  **API Testing Tool:** **Postman** atau ekstensi **Thunder Client** di VS Code (Wajib untuk mengetes API buatanmu sebelum ada tampilan HTML).
6.  **Code Editor:** Visual Studio Code (VS Code) dengan ekstensi resmi **Go**.

---

## 🚀 3. Panduan Setup Lokal (Langkah demi Langkah)

Ikuti langkah ini secara berurutan agar proyek bisa berjalan di laptopmu tanpa *error*:

### Langkah 3.1: Clone Repository
Buka terminal (CMD / Git Bash) di folder tempat kamu biasa menyimpan tugas kuliah, lalu jalankan:
```bash
git clone https://github.com/rajendra912/dapur-ibu-erp.git
cd dapur-ibu-erp
```

### Langkah 3.2: Buat Struktur Folder Lokal (PENTING!)
Karena Git tidak melacak folder kosong, beberapa folder arsitektur kita mungkin belum terlihat saat pertama kali di-*clone*. Jalankan perintah ini di terminal (di dalam folder `dapur-ibu-erp`) untuk memastikan semua folder tersedia di laptopmu:

*   **Jika menggunakan CMD Windows:**
    ```cmd
    mkdir internal\middleware internal\models internal\repository internal\service internal\handler internal\worker web\templates web\static
    ```
*   **Jika menggunakan Git Bash / Mac / Linux:**
    ```bash
    mkdir -p internal/{middleware,models,repository,service,handler,worker} web/{templates,static}
    ```

### Langkah 3.3: Setup Database MySQL Lokal
Setiap anggota **WAJIB** memiliki database lokal di laptop masing-masing. Jangan mengubah struktur tabel tanpa berdiskusi di grup!
1. Buka **Laragon** atau **XAMPP**, lalu nyalakan layanan **MySQL**.
2. Buka aplikasi **HeidiSQL** atau **phpMyAdmin** (`http://localhost/phpmyadmin`).
3. Buka file `migrations/001_init_12_tables.sql` yang ada di dalam folder proyek VS Code-mu.
4. **Copy seluruh isi file SQL tersebut**, lalu **Paste** ke dalam tab *Query* / *SQL* di HeidiSQL/phpMyAdmin, dan klik **Run / Eksekusi (F9)**.
5. Jika berhasil, akan muncul database baru bernama **`dapur_ibu_db`** yang berisi tepat **12 tabel**.

### Langkah 3.4: Sesuaikan Konfigurasi Database (Jika Perlu)
Buka file `config/database.go`. Secara *default*, konfigurasi menggunakan pengaturan standar Laragon/XAMPP (`username: root`, tanpa *password*, port `3306`):
```go
dsn := "root:@tcp(127.0.0.1:3306)/dapur_ibu_db?parseTime=true"
```
*(Catatan: Jika MySQL di laptopmu menggunakan password, tambahkan password setelah `root:`, misalnya `root:passwordmu@tcp...`, namun **jangan ikut di-commit** perubahan password pribadimu ke GitHub).*

### Langkah 3.5: Unduh Library & Jalankan Server
Buka terminal di VS Code (`` Ctrl + ` ``), lalu jalankan:
```bash
go mod tidy
go run cmd/server/main.go
```
Jika di terminal muncul pesan:
`✅ Koneksi ke database dapur_ibu_db berhasil!`
`Server siap menerima pesanan di http://localhost:8080`

Buka browser atau Postman, lalu akses `http://localhost:8080/api/health`. Jika muncul tulisan *"🚀 Server ERP Katering Dapur Ibu API berjalan lancar!"*, selamat! Setup lokalmu 100% berhasil.

---

## 🌿 4. Aturan Main Tim & Git Workflow (WAJIB PATUH!)

Agar kode kita tidak saling menimpa atau rusak (*Merge Conflict* parah), patuhi aturan kerja paralel berikut:

### 4.1. HARAM Push Langsung ke Branch `main`!
Branch `main` adalah cabang suci yang hanya boleh berisi kode yang sudah stabil. Setiap anggota **WAJIB** membuat cabang (*branch*) masing-masing sebelum mulai bekerja:

```bash
# 1. Pastikan kamu berada di branch main dan ambil update terbaru
git checkout main
git pull origin main

# 2. Buat dan pindah ke branch khusus milikmu (Cukup dilakukan 1x di awal)
git checkout -b modul-[nama-fitur]-[nama-kamu]
```
**Daftar Nama Branch Resmi Kelompok 7:**
*   **Rajen:** `git checkout -b modul-auth-rajen`
*   **Dzaki:** `git checkout -b modul-master-dzaki`
*   **Syarif:** `git checkout -b modul-recipe-syarif`
*   **Fiko:** `git checkout -b modul-order-fiko`
*   **Leo:** `git checkout -b modul-mrp-leo`

### 4.2. Cara Menyimpan Pekerjaan (Commit & Push)
Setelah selesai menulis kode dan mengetesnya di laptopmu:
```bash
git add .
git commit -m "feat(nama-modul): deskripsi singkat apa yang dibuat"
git push -u origin nama-branch-kamu
```
Setelah di-*push*, buka GitHub, lalu klik tombol **"Compare & pull request"** dan kabari **Rajen** di grup WhatsApp untuk di-*review* dan digabungkan (*merge*) ke `main`.

### 4.3. Strategi Kerja Paralel (Tanpa Saling Tunggu!)
Jangan pernah beralasan *"Aku belum bisa ngoding karena fitur si A belum selesai"*. Struktur 12 tabel kita sudah ada di laptopmu!
*   **Contoh Kasus:** Syarif mau membuat fitur Resep, tapi butuh data Bahan Baku di tabel `items` (yang merupakan tugas Dzaki, dan Dzaki belum selesai).
*   **Solusi:** Syarif cukup membuka HeidiSQL/phpMyAdmin, lalu **input manual 2-3 baris data bahan baku palsu (*dummy*)** langsung ke tabel `items` dan `units`. Dengan begitu, Syarif bisa langsung mengetes kode Resep-nya detik itu juga!

---

## 📂 5. Pola Penulisan Kode (Arsitektur Modular)

Saat membuat fitur, bagi kodemu ke dalam 4 lapisan folder di dalam `internal/` agar rapi dan seragam:
1.  **`internal/models/`**: Berisi `struct` Go yang merepresentasikan tabel database dan format JSON (`json:"nama_field"`).
2.  **`internal/repository/`**: Berisi fungsi yang langsung berbicara dengan database MySQL (tempat kamu menulis kueri `SELECT`, `INSERT`, `UPDATE`).
3.  **`internal/service/`**: Berisi logika bisnis dan perhitungan matematika (contoh: rumus konversi satuan, rumus HPP *Moving Average*, pengecekan tanggal *Cutoff*).
4.  **`internal/handler/`**: Berisi fungsi HTTP (`w http.ResponseWriter, r *http.Request`) yang menerima *request* JSON dari *user*, memanggil `service`, lalu mengembalikan *response* JSON.

---

## 👨‍💻 6. Detail Pembagian Tugas (Job Description Super Lengkap)

Berikut adalah spesifikasi teknis mendalam untuk setiap anggota tim.

---

### 👑 ORANG 1: RAJEN (Lead, Core System & Authentication)
*   **Branch:** `modul-auth-rajen`
*   **Tabel Database:** `users`
*   **File yang Harus Dibuat/Diedit:**
    *   `internal/models/user.go`
    *   `internal/repository/user_repo.go`
    *   `internal/service/auth_service.go`
    *   `internal/handler/auth_master_handler.go`
    *   `internal/middleware/auth_middleware.go`
    *   `cmd/server/main.go` (Mengatur *routing* seluruh tim)

#### Rincian Tugas Teknis Rajen:
1.  **Manajemen Pengguna & Enkripsi (`POST /api/auth/register` & `GET /api/users`):**
    *   Saat mendaftarkan *user* baru (`OWNER`, `ADMIN`, `KITCHEN`, `WAREHOUSE`), *password* **dilarang keras** disimpan sebagai teks biasa. Gunakan pustaka `golang.org/x/crypto/bcrypt` (`bcrypt.GenerateFromPassword`) untuk mengubah *password* menjadi `password_hash` sebelum di-*INSERT* ke tabel `users`.
2.  **Sistem Login (`POST /api/auth/login`):**
    *   Menerima JSON: `{"username": "admin1", "password": "123"}`.
    *   Mencocokkan *hash* dengan `bcrypt.CompareHashAndPassword`.
    *   Jika cocok, kembalikan *Session Cookie* atau *JWT Token* (atau minimal kembalikan data `role` dan simpan di *session/header* untuk divalidasi oleh Middleware).
3.  **Middleware RBAC (`internal/middleware/auth_middleware.go`):**
    *   Buat fungsi pembungkus (*wrapper*) HTTP untuk mengecek *Role* pengguna yang sedang mengakses API.
    *   Aturan Akses:
        *   `ADMIN`: Hanya boleh akses modul Pesanan (`/api/orders`).
        *   `KITCHEN`: Hanya boleh akses modul Resep (`/api/recipes`) dan jadwal produksi.
        *   `WAREHOUSE`: Hanya boleh akses modul Master Bahan, Inventori, Pembelian, dan MRP.
        *   `OWNER`: Boleh mengakses semua endpoint `GET` (Super-Read) dan laporan analisis.
    *   Jika melanggar, kembalikan HTTP Status `403 Forbidden` dengan pesan JSON `{"error": "Akses ditolak untuk role Anda"}`.
4.  **Tugas Integrator (Tech Lead):**
    *   Mendaftarkan semua rute URL dari Handler milik Dzaki, Syarif, Fiko, dan Leo ke dalam `cmd/server/main.go`.
    *   Me-*review* dan menggabungkan (*Merge*) *Pull Request* dari anggota tim di GitHub.

---

### 📦 ORANG 2: DZAKI (Master Data, Inventory & Log Harga)
*   **Branch:** `modul-master-dzaki`
*   **Tabel Database:** `units`, `items`, `stock_movements`, `material_price_logs`
*   **File yang Harus Dibuat:**
    *   `internal/models/item_unit.go` & `internal/models/warehouse.go`
    *   `internal/repository/item_repo.go` & `internal/repository/warehouse_repo.go`
    *   `internal/service/moving_avg_service.go` & `internal/service/warehouse_service.go`
    *   `internal/handler/warehouse_handler.go`

#### Rincian Tugas Teknis Dzaki:
1.  **CRUD Master Satuan (`GET`, `POST` di `/api/units`):**
    *   Mengelola satuan seperti `gr` (Gram), `ml` (Mililiter), `krg` (Karung), `pcs` (Pieces).
2.  **CRUD Master Barang & Konversi (`GET`, `POST`, `PUT`, `DELETE` di `/api/items`):**
    *   Mencatat item dengan tipe `RAW` (Bahan Mentah), `WIP` (Bahan Olahan/Bumbu), `COMP` (Komponen Masakan), dan `MENU` (Paket Katering).
    *   Menyimpan `usage_unit_id` (satuan pakai resep), `purchase_unit_id` (satuan beli pasar), dan `conversion_ratio` (misal: 1 Karung = `5000` gram).
    *   **ATURAN WAJIB (Soft Delete):** Saat endpoint `DELETE /api/items/{id}` dipanggil, **JANGAN** gunakan `DELETE FROM items`! Gunakan kueri:
        ```sql
        UPDATE items SET is_active = FALSE WHERE id = ?
        ```
    *   Saat mengambil daftar barang (`GET /api/items`), selalu gunakan filter `WHERE is_active = TRUE`.
3.  **Pencatatan Mutasi Stok (`POST /api/inventory/movements`):**
    *   Menerima input mutasi: `item_id`, `move_type` (`IN`, `OUT`, atau `WASTE`), dan `qty_change` (dalam satuan pakai).
    *   **Wajib menggunakan Database Transaction (`tx, err := config.DB.Begin()`)**:
        1. Ambil `stock_qty` saat ini dari tabel `items`.
        2. Jika `move_type` adalah `OUT` atau `WASTE`, validasi apakah `stock_qty >= qty_change`. Jika kurang, *Rollback* dan kembalikan error `"Stok gudang tidak mencukupi"`.
        3. Hitung `balance_after` (`stock_qty + qty_change` untuk `IN`, atau `stock_qty - qty_change` untuk `OUT`/`WASTE`).
        4. `UPDATE items SET stock_qty = ? WHERE id = ?`.
        5. `INSERT INTO stock_movements (...)`.
        6. Lakukan `tx.Commit()`.
4.  **Fungsi Kalkulasi HPP Moving Average & Log Harga (`moving_avg_service.go`):**
    *   Buat fungsi khusus yang nantinya juga bisa dipanggil saat ada pembelian barang baru.
    *   Rumus HPP Rata-Rata Tertimbang:
        $$\text{New Moving Avg Cost} = \frac{(\text{Stok Lama} \times \text{Harga Lama}) + (\text{Stok Masuk} \times \text{Harga Beli Baru})}{\text{Stok Lama} + \text{Stok Masuk}}$$
    *   Jika `New Moving Avg Cost` berbeda dengan `moving_avg_cost` lama di tabel `items`, otomatis lakukan `INSERT INTO material_price_logs (item_id, old_cost, new_cost, effective_date)` sebelum meng-*update* kolom `moving_avg_cost` di tabel `items`.

---

### 🍳 ORANG 3: SYARIF (Recipe Builder & Kitchen Engine)
*   **Branch:** `modul-recipe-syarif`
*   **Tabel Database:** `recipes`, `recipe_bom_items`, `items` (Read/Update HPP)
*   **File yang Harus Dibuat:**
    *   `internal/models/recipe_bom.go`
    *   `internal/repository/bom_repo.go`
    *   `internal/service/bom_service.go`
    *   `internal/handler/recipe_handler.go`

#### Rincian Tugas Teknis Syarif:
1.  **Membuat Master Resep & Komponennya (`POST /api/recipes`):**
    *   Menerima JSON bersarang (*nested*) untuk menyimpan 1 baris ke tabel `recipes` (Header) dan banyak baris ke tabel `recipe_bom_items` ( Bahan penyusunnya) dalam satu transaksi database.
    *   Contoh Payload Request JSON:
        ```json
        {
          "output_item_id": 10,
          "recipe_name": "Resep Ayam Goreng Lengkuas (Batch 20 Porsi)",
          "batch_yield": 20.00,
          "components": [
            {"component_item_id": 1, "qty_req": 5000.00},
            {"component_item_id": 5, "qty_req": 250.00}
          ]
        }
        ```
2.  **Melihat Struktur Pohon Resep (`GET /api/recipes/{id}`):**
    *   Menampilkan detail resep beserta nama item penyusunnya (lakukan `JOIN` antara `recipe_bom_items` dengan tabel `items` dan `units` agar keluar nama bahan dan satuannya, bukan cuma angka ID).
3.  **Kalkulator HPP Standar Bertingkat / Roll-up (`GET /api/recipes/{id}/calculate-hpp`):**
    *   Ini adalah logika inti bagian Dapur. Resep di Dapur Ibu memiliki hierarki:
        *   **Level 0:** `MENU` (Nasi Kotak) $\rightarrow$ disusun oleh `COMP` (Ayam Goreng, Sambal).
        *   **Level 1:** `COMP` (Ayam Goreng) $\rightarrow$ disusun oleh `WIP` (Bumbu Ungkep) dan `RAW` (Daging Ayam).
        *   **Level 2:** `WIP` (Bumbu Ungkep) $\rightarrow$ disusun oleh `RAW` (Bawang, Lengkuas, Kunyit).
    *   **Algoritma Perhitungan:**
        1. Ambil semua baris di `recipe_bom_items` untuk `recipe_id` tersebut.
        2. Kalikan `qty_req` setiap komponen dengan `moving_avg_cost` milik komponen tersebut di tabel `items`. Jumlahkan semuanya menjadi `Total Biaya Batch`.
        3. Bagi `Total Biaya Batch` dengan `batch_yield` resep tersebut untuk mendapatkan **HPP per 1 porsi/satuan output**:
           $$\text{Std HPP per Output} = \frac{\sum (\text{qty\_req} \times \text{moving\_avg\_cost})}{\text{batch\_yield}}$$
        4. Simpan hasilnya ke kolom `std_hpp` di tabel `recipes`, DAN *update* juga kolom `moving_avg_cost` milik `output_item_id` di tabel `items` (agar jika komponen ini dipakai oleh resep di level atasnya, harganya sudah terbarui!).

---

### 📅 ORANG 4: FIKO (Order Management & Automasi Cutoff H-2)
*   **Branch:** `modul-order-fiko`
*   **Tabel Database:** `orders`, `order_items`
*   **File yang Harus Dibuat:**
    *   `internal/models/order.go`
    *   `internal/repository/order_repo.go`
    *   `internal/service/order_service.go`
    *   `internal/handler/order_handler.go`
    *   `internal/middleware/cutoff_guard.go`
    *   `internal/worker/cutoff_cron.go`

#### Rincian Tugas Teknis Fiko:
1.  **Input Pesanan Pelanggan (`POST /api/orders`):**
    *   Menerima input: `order_num`, `customer`, `event_date` (Tanggal Acara, misal: `2026-11-20`), dan daftar menu yang dipesan (`order_items`: `menu_item_id`, `portions`, `selling_price`).
    *   **Otomatisasi Tanggal Cutoff:** Saat menyimpan pesanan, sistem di Go otomatis menghitung `cutoff_date`, yaitu **H-2 pukul 23:59:59** sebelum `event_date` (atau tepat H-1 pukul `00:00:00`).
        *   *Contoh:* Jika `event_date` adalah `2026-11-20`, maka `cutoff_date` otomatis diisi `2026-11-19 00:00:00`.
    *   Status awal pesanan diset menjadi `DRAFT` atau `CONFIRMED`.
2.  **Update/Edit Pesanan dengan Proteksi Cutoff (`PUT /api/orders/{id}`):**
    *   Admin boleh mengubah jumlah porsi pesanan (`portions`) **HANYA JIKA**:
        1. Status pesanan belum `LOCKED` atau `DONE`.
        2. Waktu sekarang (`time.Now()`) masih lebih kecil dari `cutoff_date`.
    *   Jika sudah melewati batas, tolak *request* dengan HTTP Status `400 Bad Request` dan pesan: `{"error": "Pesanan sudah terkunci (Melewati batas Cutoff H-2) dan tidak dapat diubah!"}`.
3.  **Background Worker / Cron-Job Penguncian Otomatis (`internal/worker/cutoff_cron.go` & `POST /api/cron/cutoff`):**
    *   Buat dua cara pemicu: (1) Goroutine dengan `time.NewTicker` yang mengecek database secara berkala, dan (2) Endpoint manual `POST /api/cron/cutoff` (sangat penting agar saat **Demo ke Dosen**, kita bisa langsung memicu penguncian tanpa harus menunggu jam 12 malam!).
    *   **Logika Penguncian & Snapshot HPP:**
        1. Cari semua pesanan di tabel `orders` dengan kondisi: `status IN ('DRAFT', 'CONFIRMED') AND cutoff_date <= NOW()`.
        2. Untuk setiap pesanan yang ditemukan, hitung total HPP per porsi dari menu yang dipesan (ambil nilai `moving_avg_cost` / `std_hpp` dari item menu terkait saat detik itu juga).
        3. Lakukan `UPDATE`:
           ```sql
           UPDATE orders 
           SET status = 'LOCKED', locked_hpp_per_portion = ? 
           WHERE id = ?
           ```
        *Tujuan:* Menyimpan *snapshot* HPP agar laporan laba kotor pesanan ini tidak berubah meskipun bulan depan harga cabai di pasar naik.

---

### 🛒 ORANG 5: LEO (MRP Engine / Exploding BOM, Daftar Belanja & Pembelian)
*   **Branch:** `modul-mrp-leo`
*   **Tabel Database:** `production_runs`, `shopping_lists`, `purchases`
*   **File yang Harus Dibuat:**
    *   `internal/models/mrp_production.go`
    *   `internal/repository/mrp_repo.go`
    *   `internal/service/mrp_service.go`
    *   `internal/handler/mrp_handler.go`

#### Rincian Tugas Teknis Leo:
1.  **Engine Utama: Exploding BOM & Generate Daftar Belanja (`POST /api/mrp/explode`):**
    *   Ini adalah fitur bintang utama dari ERP kita! Menerima input tanggal produksi, misal: `{"production_date": "2026-11-20"}`.
    *   **Langkah Algoritma secara berurutan:**
        1. **Buat Jadwal Produksi:** Buat atau ambil ID dari tabel `production_runs` untuk tanggal tersebut.
        2. **Agregasi Pesanan (Level 0):** Cari semua pesanan di tabel `orders` pada tanggal tersebut yang berstatus **`LOCKED`**. Jumlahkan total `portions` per `menu_item_id` dari tabel `order_items`. *(Contoh: Total 500 porsi Nasi Kotak A)*.
        3. **Pecah Resep Secara Rekursif (Level 1 $\rightarrow$ Level 2 $\rightarrow$ Level 3):**
           * Telusuri tabel `recipes` dan `recipe_bom_items` dari setiap menu.
           * Hitung pengali batch: $\text{Multiplier} = \frac{\text{Kebutuhan Output}}{\text{batch\_yield}}$.
           * Kalikan $\text{Multiplier}$ dengan `qty_req` setiap komponen di bawahnya.
           * Jika komponen tersebut bertipe `COMP` atau `WIP` (masih punya sub-resep lagi), pecah lagi ke bawah sampai semuanya murni menjadi item bertipe **`RAW` (Bahan Mentah Dasar)**.
           * Jumlahkan seluruh kebutuhan bahan mentah yang sama menjadi **`gross_req`** (Kebutuhan Kotor dalam satuan pakai: gram/ml).
        4. **Potong Stok Gudang (Net Requirement):**
           * Ambil `stock_qty` terkini dari tabel `items` sebagai **`stock_on_hand`**.
           * Hitung kebutuhan bersih: $\text{net\_req} = \text{gross\_req} - \text{stock\_on\_hand}$. (Jika hasilnya $\le 0$, berarti stok gudang cukup dan `net_req = 0`).
        5. **Konversi Satuan Beli & Pembulatan Ke Atas (`CEIL`):**
           * Bagi `net_req` dengan `conversion_ratio` bahan baku tersebut, lalu bulatkan ke atas menggunakan fungsi `math.Ceil()` di Go!
           * *Contoh:* Butuh bersih `4.500` gram beras. Rasio konversi karung = `5.000` gram. Maka $\lceil 4500 / 5000 \rceil = \lceil 0.9 \rceil = 1$ Karung (`purchase_qty_rounded = 1`).
        6. **Simpan ke Database:** Lakukan `INSERT` hasil perhitungannya ke tabel **`shopping_lists`**, dan kembalikan hasilnya sebagai JSON agar bisa dicetak oleh orang gudang.
2.  **Melihat Daftar Belanja (`GET /api/mrp/shopping-lists?date=YYYY-MM-DD`):**
    *   Menampilkan daftar belanja yang sudah di-*generate* lengkap dengan nama bahan baku, kebutuhan kotor (gram), stok gudang (gram), kebutuhan bersih (gram), dan jumlah yang harus dibeli di pasar beserta satuan belinya (Karung/Pouch/Kg).
3.  **Pencatatan Nota Pembelian Pasar (`POST /api/purchases`):**
    *   Ketika orang gudang selesai belanja dari pasar, mereka menginput nota: `invoice_no`, `supplier`, `purchase_date`, serta rincian barang yang dibeli (`item_id`, jumlah dalam satuan beli, dan harga beli aktual).
    *   **Integrasi dengan Modul Dzaki:**
        1. Simpan nota ke tabel `purchases`.
        2. Konversikan jumlah beli ke satuan pakai ($\text{Qty Pakai} = \text{Qty Beli} \times \text{conversion\_ratio}$).
        3. Panggil logika *Moving Average* untuk memperbarui HPP bahan baku di tabel `items` dan mencatat log di `material_price_logs`.
        4. Tambah stok gudang (`stock_qty`) dan catat riwayat barang masuk (`move_type = 'IN'`) di tabel `stock_movements` dengan menyertakan `purchase_id`.

---

## 📊 7. Urutan Prioritas Pengerjaan (Timeline Sprint)

Agar kita bisa mendemokan progres dengan tenang, ikuti urutan pengerjaan ini di cabang (*branch*) masing-masing:
1.  **Hari 1 - 2:** Semua anggota wajib sudah selesai *clone* repo, import 12 tabel SQL di lokal, dan berhasil menyalakan `go run cmd/server/main.go`.
2.  **Hari 3 - 5:** Masing-masing anggota membuat file di folder `models/`, `repository/`, `service/`, dan `handler/` sesuai pembagian di atas. Fokus sampai endpoint bisa dites pakai **Postman**!
3.  **Hari 6:** *Pull Request* ke GitHub. **Rajen** melakukan penggabungan (*Merge*) seluruh endpoint API ke branch `main`.
4.  **Hari 7 dst:** Pembuatan halaman antarmuka HTML/CSS/JS di dalam folder `web/templates/` dan menyambungkannya ke API yang sudah matang.

---
*Dibuat dengan penuh dedikasi oleh Kelompok 7. Jika ada kendala error saat setup atau bingung mengenai kueri SQL, segera tanyakan di grup WhatsApp!*
