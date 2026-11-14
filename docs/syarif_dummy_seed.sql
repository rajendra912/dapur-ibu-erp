-- Data dummy untuk menguji modul Recipe & Kitchen Engine (Syarif).
-- Dijalankan SETELAH migrations/001_init_12_tables.sql.
-- Memakai ID 8001+ agar tidak bentrok dengan data dummy modul lain (misal leo_dummy_seed.sql 9001+).
-- Aman dijalankan berulang kali (bagian awal membersihkan data dummy 8001+ terlebih dahulu).

USE dapur_ibu_db;

-- Bersihkan data dummy lama (urutan: tabel anak terlebih dahulu)
DELETE FROM recipe_bom_items WHERE recipe_id BETWEEN 8201 AND 8209;
DELETE FROM recipes          WHERE id BETWEEN 8201 AND 8209;
DELETE FROM items            WHERE id BETWEEN 8101 AND 8112;
DELETE FROM units            WHERE id BETWEEN 8001 AND 8004;

-- 1. Master Satuan (Units)
INSERT INTO units (id, code, name) VALUES
 (8001, 'gr-syr',  'Gram (Syarif)'),
 (8002, 'ml-syr',  'Mililiter (Syarif)'),
 (8003, 'kg-syr',  'Kilogram (Syarif)'),
 (8004, 'pcs-syr', 'Porsi (Syarif)');

-- 2. Master Items (RAW, WIP, COMP, MENU + 1 item nonaktif untuk uji validasi)
INSERT INTO items (id, sku, name, item_type, usage_unit_id, purchase_unit_id, conversion_ratio, stock_qty, moving_avg_cost, is_active) VALUES
 (8101, 'SYR-MENU-01', 'Paket Nasi Rendang',    'MENU', 8004, 8004, 1,    0,     0.0000, TRUE),
 (8102, 'SYR-COMP-01', 'Rendang Daging',        'COMP', 8004, 8004, 1,    0,     0.0000, TRUE),
 (8103, 'SYR-WIP-01',  'Bumbu Halus Rendang',   'WIP',  8001, 8001, 1,    0,     0.0000, TRUE),
 (8104, 'SYR-RAW-01',  'Daging Sapi',           'RAW',  8001, 8003, 1000, 5000,  140.0000, TRUE), -- Rp 140/gr (Rp 140.000/kg)
 (8105, 'SYR-RAW-02',  'Santan Kental',         'RAW',  8002, 8002, 1,    3000,  25.0000, TRUE),  -- Rp 25/ml
 (8106, 'SYR-RAW-03',  'Cabai Merah',           'RAW',  8001, 8003, 1000, 2000,  60.0000, TRUE),  -- Rp 60/gr (Rp 60.000/kg)
 (8107, 'SYR-RAW-04',  'Bawang Merah',          'RAW',  8001, 8003, 1000, 2000,  40.0000, TRUE),  -- Rp 40/gr (Rp 40.000/kg)
 (8108, 'SYR-RAW-05',  'Beras Putih',           'RAW',  8001, 8003, 1000, 10000, 15.0000, TRUE),  -- Rp 15/gr (Rp 15.000/kg)
 (8109, 'SYR-RAW-06',  'Daun Singkong',         'RAW',  8001, 8003, 1000, 1500,  20.0000, TRUE),  -- Rp 20/gr (Rp 20.000/kg)
 (8110, 'SYR-RAW-99',  'Bahan Nonaktif (Uji)',  'RAW',  8001, 8003, 1000, 0,     10.0000, FALSE); -- is_active = FALSE

-- 3. Master Resep Bertingkat (BOM Multi-level)
--    Level 2 (WIP) : 8203 Bumbu Halus Rendang (batch_yield = 1000 gr)
--    Level 1 (COMP): 8202 Rendang Daging (batch_yield = 20 porsi)
--    Level 0 (MENU): 8201 Paket Nasi Rendang (batch_yield = 1 porsi)
INSERT INTO recipes (id, output_item_id, recipe_name, batch_yield, std_hpp) VALUES
 (8203, 8103, 'Resep Bumbu Halus Rendang', 1000.00, 0.00),
 (8202, 8102, 'Resep Rendang Daging',      20.00,   0.00),
 (8201, 8101, 'Resep Paket Nasi Rendang',  1.00,    0.00);

-- 4. Komponen BOM (recipe_bom_items)
INSERT INTO recipe_bom_items (recipe_id, component_item_id, qty_req) VALUES
 -- Bumbu Halus Rendang (1 batch = 1000 gr):
 --   500 gr Cabai Merah (500 * 60 = 30.000) + 500 gr Bawang Merah (500 * 40 = 20.000) = 50.000 / 1000 = Rp 50/gr
 (8203, 8106, 500.0000),
 (8203, 8107, 500.0000),

 -- Rendang Daging (1 batch = 20 porsi):
 --   2000 gr Daging Sapi (280.000) + 1000 ml Santan (25.000) + 500 gr Bumbu Halus (25.000) = 330.000 / 20 = Rp 16.500/porsi
 (8202, 8104, 2000.0000),
 (8202, 8105, 1000.0000),
 (8202, 8103, 500.0000),

 -- Paket Nasi Rendang (1 batch = 1 porsi):
 --   1 porsi Rendang (16.500) + 150 gr Beras (2.250) + 50 gr Daun Singkong (1.000) = Rp 19.750/porsi
 (8201, 8102, 1.0000),
 (8201, 8108, 150.0000),
 (8201, 8109, 50.0000);
