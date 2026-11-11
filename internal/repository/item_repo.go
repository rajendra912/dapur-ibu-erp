package repository

import (
	"database/sql"

	"github.com/rajendra912/dapur-ibu-erp/internal/models"
)

func ListUnits(db *sql.DB) ([]models.Unit, error) {
	rows, err := db.Query(`SELECT id, code, name FROM units ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Unit{}
	for rows.Next() {
		var u models.Unit
		if err := rows.Scan(&u.ID, &u.Code, &u.Name); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func CreateUnit(db *sql.DB, u models.Unit) (int64, error) {
	res, err := db.Exec(`INSERT INTO units (code, name) VALUES (?, ?)`, u.Code, u.Name)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func ListItems(db *sql.DB) ([]models.Item, error) {
	rows, err := db.Query(`
		SELECT i.id, COALESCE(i.sku,''), i.name, i.item_type,
		       COALESCE(i.usage_unit_id,0), COALESCE(i.purchase_unit_id,0),
		       COALESCE(uu.code,''), COALESCE(pu.code,''),
		       COALESCE(i.conversion_ratio,0), COALESCE(i.stock_qty,0), COALESCE(i.moving_avg_cost,0)
		FROM items i
		LEFT JOIN units uu ON uu.id = i.usage_unit_id
		LEFT JOIN units pu ON pu.id = i.purchase_unit_id
		WHERE i.is_active = TRUE
		ORDER BY i.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Item{}
	for rows.Next() {
		var it models.Item
		if err := rows.Scan(&it.ID, &it.SKU, &it.Name, &it.ItemType,
			&it.UsageUnitID, &it.PurchaseUnitID, &it.UsageUnit, &it.PurchaseUnit,
			&it.ConversionRatio, &it.StockQty, &it.MovingAvgCost); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func CreateItem(db *sql.DB, it models.Item) (int64, error) {
	res, err := db.Exec(`
		INSERT INTO items (sku, name, item_type, usage_unit_id, purchase_unit_id, conversion_ratio)
		VALUES (NULLIF(?,''), ?, ?, NULLIF(?,0), NULLIF(?,0), NULLIF(?,0))`,
		it.SKU, it.Name, it.ItemType, it.UsageUnitID, it.PurchaseUnitID, it.ConversionRatio)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func ItemExists(db *sql.DB, id int64) (bool, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM items WHERE id = ? AND is_active = TRUE`, id).Scan(&n)
	return n > 0, err
}

// Sengaja TIDAK mengubah stock_qty dan moving_avg_cost.
func UpdateItem(db *sql.DB, id int64, it models.Item) error {
	_, err := db.Exec(`
		UPDATE items SET sku = NULLIF(?,''), name = ?, item_type = ?,
		       usage_unit_id = NULLIF(?,0), purchase_unit_id = NULLIF(?,0), conversion_ratio = NULLIF(?,0)
		WHERE id = ? AND is_active = TRUE`,
		it.SKU, it.Name, it.ItemType, it.UsageUnitID, it.PurchaseUnitID, it.ConversionRatio, id)
	return err
}

// Soft delete, JANGAN DELETE FROM.
func DeactivateItem(db *sql.DB, id int64) (int64, error) {
	res, err := db.Exec(`UPDATE items SET is_active = FALSE WHERE id = ? AND is_active = TRUE`, id)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
