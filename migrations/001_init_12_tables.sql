-- Membuat Database
CREATE DATABASE IF NOT EXISTS dapur_ibu_db;
USE dapur_ibu_db;

-- 1. TABEL USERS (Untuk Login & RBAC)
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(100) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role ENUM('OWNER', 'ADMIN', 'KITCHEN', 'WAREHOUSE') NOT NULL
) ENGINE=InnoDB;

-- 2. TABEL UNITS (Master Satuan)
CREATE TABLE units (
    id INT AUTO_INCREMENT PRIMARY KEY,
    code VARCHAR(20) NOT NULL UNIQUE, -- cth: ml, gr, pcs, krg
    name VARCHAR(50) NOT NULL
) ENGINE=InnoDB;

-- 3. TABEL ITEMS (Master Bahan Baku & Menu - Soft Delete)
CREATE TABLE items (
    id INT AUTO_INCREMENT PRIMARY KEY,
    sku VARCHAR(50) UNIQUE,
    name VARCHAR(150) NOT NULL,
    item_type ENUM('RAW', 'WIP', 'COMP', 'MENU') NOT NULL,
    usage_unit_id INT,
    purchase_unit_id INT,
    conversion_ratio DECIMAL(12,4),
    stock_qty DECIMAL(14,2) DEFAULT 0.00,
    moving_avg_cost DECIMAL(14,4) DEFAULT 0.0000,
    is_active BOOLEAN DEFAULT TRUE,
    FOREIGN KEY (usage_unit_id) REFERENCES units(id),
    FOREIGN KEY (purchase_unit_id) REFERENCES units(id)
) ENGINE=InnoDB;

-- 4. TABEL MATERIAL PRICE LOGS (Riwayat Perubahan Harga)
CREATE TABLE material_price_logs (
    id INT AUTO_INCREMENT PRIMARY KEY,
    item_id INT NOT NULL,
    old_cost DECIMAL(14,4) NOT NULL,
    new_cost DECIMAL(14,4) NOT NULL,
    effective_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (item_id) REFERENCES items(id)
) ENGINE=InnoDB;

-- 5. TABEL RECIPES (Master Resep)
CREATE TABLE recipes (
    id INT AUTO_INCREMENT PRIMARY KEY,
    output_item_id INT NOT NULL,
    recipe_name VARCHAR(150) NOT NULL,
    batch_yield DECIMAL(10,2) NOT NULL,
    std_hpp DECIMAL(14,2) DEFAULT 0.00,
    FOREIGN KEY (output_item_id) REFERENCES items(id)
) ENGINE=InnoDB;

-- 6. TABEL RECIPE BOM ITEMS (Struktur Komponen Resep Bertingkat)
CREATE TABLE recipe_bom_items (
    id INT AUTO_INCREMENT PRIMARY KEY,
    recipe_id INT NOT NULL,
    component_item_id INT NOT NULL,
    qty_req DECIMAL(12,4) NOT NULL,
    FOREIGN KEY (recipe_id) REFERENCES recipes(id),
    FOREIGN KEY (component_item_id) REFERENCES items(id)
) ENGINE=InnoDB;

-- 7. TABEL ORDERS (Pesanan Pelanggan)
CREATE TABLE orders (
    id INT AUTO_INCREMENT PRIMARY KEY,
    order_num VARCHAR(50) NOT NULL UNIQUE,
    customer VARCHAR(150) NOT NULL,
    event_date DATE NOT NULL,
    cutoff_date DATETIME NOT NULL,
    status ENUM('DRAFT', 'CONFIRMED', 'LOCKED', 'DONE') DEFAULT 'DRAFT',
    locked_hpp_per_portion DECIMAL(14,2) DEFAULT 0.00
) ENGINE=InnoDB;

-- 8. TABEL ORDER ITEMS (Rincian Menu yang Dipesan)
CREATE TABLE order_items (
    id INT AUTO_INCREMENT PRIMARY KEY,
    order_id INT NOT NULL,
    menu_item_id INT NOT NULL,
    portions INT NOT NULL,
    selling_price DECIMAL(14,2) NOT NULL,
    FOREIGN KEY (order_id) REFERENCES orders(id),
    FOREIGN KEY (menu_item_id) REFERENCES items(id)
) ENGINE=InnoDB;

-- 9. TABEL PRODUCTION RUNS (Jadwal Produksi Harian)
CREATE TABLE production_runs (
    id INT AUTO_INCREMENT PRIMARY KEY,
    production_date DATE NOT NULL UNIQUE,
    status VARCHAR(50) DEFAULT 'PLANNED'
) ENGINE=InnoDB;

-- 10. TABEL PURCHASES (Penerimaan Barang/Belanja)
CREATE TABLE purchases (
    id INT AUTO_INCREMENT PRIMARY KEY,
    invoice_no VARCHAR(100) NOT NULL,
    supplier VARCHAR(150) NOT NULL,
    purchase_date DATE NOT NULL,
    total_amount DECIMAL(14,4) NOT NULL,
    cost_applied DECIMAL(14,4) NOT NULL
) ENGINE=InnoDB;

-- 11. TABEL SHOPPING LISTS (Daftar Belanja MRP Engine)
CREATE TABLE shopping_lists (
    id INT AUTO_INCREMENT PRIMARY KEY,
    production_run_id INT NOT NULL,
    raw_item_id INT NOT NULL,
    gross_req DECIMAL(14,2) NOT NULL,
    stock_on_hand DECIMAL(14,2) NOT NULL,
    net_req DECIMAL(14,2) NOT NULL,
    purchase_qty_rounded INT NOT NULL,
    FOREIGN KEY (production_run_id) REFERENCES production_runs(id),
    FOREIGN KEY (raw_item_id) REFERENCES items(id)
) ENGINE=InnoDB;

-- 12. TABEL STOCK MOVEMENTS (Kartu Stok & Mutasi)
CREATE TABLE stock_movements (
    id INT AUTO_INCREMENT PRIMARY KEY,
    item_id INT NOT NULL,
    purchase_id INT DEFAULT NULL,
    production_run_id INT DEFAULT NULL,
    move_type ENUM('IN', 'OUT', 'WASTE') NOT NULL,
    qty_change DECIMAL(14,2) NOT NULL,
    balance_after DECIMAL(14,2) NOT NULL,
    FOREIGN KEY (item_id) REFERENCES items(id),
    FOREIGN KEY (purchase_id) REFERENCES purchases(id),
    FOREIGN KEY (production_run_id) REFERENCES production_runs(id)
) ENGINE=InnoDB;