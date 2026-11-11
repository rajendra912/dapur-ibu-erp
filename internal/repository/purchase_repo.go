package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"

	"github.com/rajendra912/dapur-ibu-erp/internal/models"
)

var ErrNotFound = errors.New("data tidak ditemukan")

type PurchaseRepo struct{ db *sql.DB }

func NewPurchaseRepo(db *sql.DB) *PurchaseRepo { return &PurchaseRepo{db: db} }

// WithTx menjalankan fn dalam satu transaksi; rollback otomatis bila fn mengembalikan error.
func (r *PurchaseRepo) WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

type ItemStock struct {
	StockQty        float64
	MovingAvgCost   float64
	ConversionRatio float64
}

// GetItemForUpdate mengunci baris item (FOR UPDATE) agar saldo stok aman dari race condition.
func (r *PurchaseRepo) GetItemForUpdate(ctx context.Context, tx *sql.Tx, itemID int) (ItemStock, error) {
	var s ItemStock
	err := tx.QueryRowContext(ctx, `
		SELECT COALESCE(stock_qty, 0), COALESCE(moving_avg_cost, 0), COALESCE(conversion_ratio, 0)
		FROM items WHERE id = ? AND is_active = TRUE FOR UPDATE`, itemID).
		Scan(&s.StockQty, &s.MovingAvgCost, &s.ConversionRatio)
	if errors.Is(err, sql.ErrNoRows) {
		return s, fmt.Errorf("item id %d tidak ditemukan atau nonaktif: %w", itemID, ErrNotFound)
	}
	return s, err
}

func (r *PurchaseRepo) InsertPurchase(ctx context.Context, tx *sql.Tx, invoice, supplier, date string, total float64) (int, error) {
	res, err := tx.ExecContext(ctx,
		`INSERT INTO purchases (invoice_no, supplier, purchase_date, total_amount, cost_applied) VALUES (?, ?, ?, ?, ?)`,
		invoice, supplier, date, total, total) // cost_applied (NOT NULL di skema) diisi = total_amount; konfirmasi maknanya ke tim
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

// ApplyInbound = mutasi stok IN. SEMENTARA dibuat di sini karena endpoint mutasi milik Dzaki
// (POST /api/inventory/movement) belum ada. Setelah modul Dzaki ter-merge, ganti pemanggilan
// ini dengan fungsi/service milik Dzaki agar logika stok tidak duplikat.
func (r *PurchaseRepo) ApplyInbound(ctx context.Context, tx *sql.Tx, itemID, purchaseID int, qtyIn, oldQty, oldCost, newCost float64) (newQty float64, err error) {
	newQty = oldQty + qtyIn
	if _, err = tx.ExecContext(ctx,
		`UPDATE items SET stock_qty = ?, moving_avg_cost = ? WHERE id = ?`, newQty, newCost, itemID); err != nil {
		return 0, err
	}
	// Log harga otomatis setiap moving_avg_cost berubah
	if math.Abs(newCost-oldCost) > 1e-9 {
		if _, err = tx.ExecContext(ctx,
			`INSERT INTO material_price_logs (item_id, old_cost, new_cost, effective_date) VALUES (?, ?, ?, NOW())`,
			itemID, oldCost, newCost); err != nil {
			return 0, err
		}
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO stock_movements
			(item_id, purchase_id, production_run_id, move_type, qty_change, balance_after)
		VALUES (?, ?, NULL, 'IN', ?, ?)`,
		itemID, purchaseID, qtyIn, newQty)
	return newQty, err
}

func (r *PurchaseRepo) List(ctx context.Context) ([]models.Purchase, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, invoice_no, COALESCE(supplier, ''), DATE_FORMAT(purchase_date, '%Y-%m-%d'), total_amount
		FROM purchases ORDER BY purchase_date DESC, id DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Purchase{}
	for rows.Next() {
		var p models.Purchase
		if err := rows.Scan(&p.ID, &p.InvoiceNo, &p.Supplier, &p.PurchaseDate, &p.TotalAmount); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
