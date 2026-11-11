package repository

import (
	"database/sql"

	"dapur-ibu-erp/internal/models"

	"github.com/shopspring/decimal"
)

func LockItem(tx *sql.Tx, id int64) (stock, avg decimal.Decimal, err error) {
	err = tx.QueryRow(`SELECT stock_qty, moving_avg_cost FROM items
		WHERE id = ? AND is_active = TRUE FOR UPDATE`, id).Scan(&stock, &avg)
	return
}

func SaveItemStock(tx *sql.Tx, id int64, stock decimal.Decimal) error {
	_, err := tx.Exec(`UPDATE items SET stock_qty = ? WHERE id = ?`, stock, id)
	return err
}

func SaveItemCost(tx *sql.Tx, id int64, cost decimal.Decimal) error {
	_, err := tx.Exec(`UPDATE items SET moving_avg_cost = ? WHERE id = ?`, cost, id)
	return err
}

func InsertPriceLog(tx *sql.Tx, id int64, oldCost, newCost decimal.Decimal) error {
	_, err := tx.Exec(`INSERT INTO material_price_logs (item_id, old_cost, new_cost)
		VALUES (?, ?, ?)`, id, oldCost, newCost)
	return err
}

func InsertMovement(tx *sql.Tx, m models.Movement, balance decimal.Decimal) error {
	_, err := tx.Exec(`INSERT INTO stock_movements
		(item_id, purchase_id, production_run_id, move_type, qty_change, balance_after)
		VALUES (?, ?, ?, ?, ?, ?)`,
		m.ItemID, m.PurchaseID, m.ProductionRunID, m.MoveType, m.QtyChange, balance)
	return err
}
