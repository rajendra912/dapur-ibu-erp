-- Data dummy untuk menguji modul MRP & Pembelian (Leo) TANPA menunggu modul teman.
-- Dijalankan SETELAH migrations/001_init_12_tables.sql. Memakai id 9001+ agar tidak bentrok dengan data lain.
-- Sudah disesuaikan dengan 001_init_12_tables.sql. Aman dijalankan berulang (bagian pertama membersihkan data dummy lama).
USE dapur_ibu_db;

-- Bersihkan data dummy lama (urutan: tabel anak dulu)
DELETE FROM stock_movements    WHERE item_id BETWEEN 9101 AND 9106;
DELETE FROM material_price_logs WHERE item_id BETWEEN 9101 AND 9106;
DELETE FROM shopping_lists     WHERE raw_item_id BETWEEN 9101 AND 9106;
DELETE FROM production_runs    WHERE production_date = '2026-11-20' AND id NOT IN (SELECT production_run_id FROM stock_movements WHERE production_run_id IS NOT NULL);
DELETE FROM order_items        WHERE order_id BETWEEN 9301 AND 9303;
DELETE FROM orders             WHERE id BETWEEN 9301 AND 9303;
DELETE FROM recipe_bom_items   WHERE recipe_id BETWEEN 9201 AND 9203;
DELETE FROM recipes            WHERE id BETWEEN 9201 AND 9203;
DELETE FROM items              WHERE id BETWEEN 9101 AND 9106;
DELETE FROM units              WHERE id BETWEEN 9001 AND 9004;

INSERT INTO units (id, code, name) VALUES
 (9001, 'gr',  'Gram'),
 (9002, 'kg',  'Kilogram'),
 (9003, 'krg', 'Karung 5kg'),
 (9004, 'pcs', 'Porsi');

-- item_type: MENU / COMP / WIP / RAW
INSERT INTO items (id, sku, name, item_type, usage_unit_id, purchase_unit_id, conversion_ratio, stock_qty, moving_avg_cost, is_active) VALUES
 (9101, 'LEO-MENU-01', 'Nasi Kotak A',  'MENU', 9004, 9004, 1,    0,     0,   TRUE),
 (9102, 'LEO-COMP-01', 'Ayam Goreng',   'COMP', 9004, 9004, 1,    0,     0,   TRUE),
 (9103, 'LEO-WIP-01',  'Bumbu Dasar',   'WIP',  9001, 9001, 1,    0,     0,   TRUE),
 (9104, 'LEO-RAW-01',  'Ayam Mentah',   'RAW',  9001, 9002, 1000, 2000,  38,  TRUE),
 (9105, 'LEO-RAW-02',  'Bawang Merah',  'RAW',  9001, 9002, 1000, 0,     30,  TRUE),
 (9106, 'LEO-RAW-03',  'Beras',         'RAW',  9001, 9003, 5000, 99999, 13,  TRUE);

INSERT INTO recipes (id, output_item_id, recipe_name, batch_yield, std_hpp) VALUES
 (9201, 9101, 'Resep Nasi Kotak A', 1,  0),
 (9202, 9102, 'Resep Ayam Goreng',  20, 0),
 (9203, 9103, 'Resep Bumbu Dasar',  1,  0);

INSERT INTO recipe_bom_items (recipe_id, component_item_id, qty_req) VALUES
 (9201, 9102, 1),     -- 1 porsi menu = 1 porsi ayam goreng
 (9201, 9106, 150),   -- + 150 gr beras
 (9202, 9104, 4000),  -- 1 batch (20 porsi) = 4000 gr ayam mentah
 (9202, 9103, 1),     -- + 1 batch bumbu dasar
 (9203, 9105, 500);   -- 1 batch bumbu dasar = 500 gr bawang merah

-- Pesanan LOCKED untuk tanggal produksi 2026-11-20 (300 + 200 = 500 porsi)
INSERT INTO orders (id, order_num, customer, event_date, cutoff_date, status) VALUES
 (9301, 'LEO-ORD-001', 'Pak Budi',  '2026-11-20', '2026-11-19 00:00:00', 'LOCKED'),
 (9302, 'LEO-ORD-002', 'Bu Sari',   '2026-11-20', '2026-11-19 00:00:00', 'LOCKED'),
 (9303, 'LEO-ORD-003', 'Pak Andi',  '2026-11-20', '2026-11-19 00:00:00', 'DRAFT');  -- harus diabaikan MRP

INSERT INTO order_items (order_id, menu_item_id, portions, selling_price) VALUES
 (9301, 9101, 300, 25000),
 (9302, 9101, 200, 25000),
 (9303, 9101, 100, 25000);

-- Hasil yang diharapkan dari POST /api/mrp/explode {"production_date":"2026-11-20"}:
--   Ayam Mentah : gross 100000, stok 2000, net 98000  -> 98 kg
--   Bawang Merah: gross  12500, stok 0,    net 12500  -> 13 kg (12,5 dibulatkan ke atas)
--   Beras       : stok cukup -> tidak masuk daftar belanja
