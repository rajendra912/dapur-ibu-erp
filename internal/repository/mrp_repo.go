package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/rajendra912/dapur-ibu-erp/internal/models"
)

var ErrRunClosed = errors.New("jadwal produksi sudah berstatus DONE, daftar belanja tidak dapat dibuat ulang")

type MRPRepo struct{ db *sql.DB }

func NewMRPRepo(db *sql.DB) *MRPRepo { return &MRPRepo{db: db} }

// LockedDemand menjumlahkan porsi per menu (Level 0) dari seluruh pesanan LOCKED pada tanggal produksi.
func (r *MRPRepo) LockedDemand(ctx context.Context, date string) (map[int]float64, int, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT oi.menu_item_id, SUM(oi.portions)
		FROM orders o
		JOIN order_items oi ON oi.order_id = o.id
		WHERE o.status = 'LOCKED' AND o.event_date = ?
		GROUP BY oi.menu_item_id`, date)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	demand := map[int]float64{}
	for rows.Next() {
		var menuID int
		var portions float64
		if err := rows.Scan(&menuID, &portions); err != nil {
			return nil, 0, err
		}
		demand[menuID] = portions
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	var orderCount int
	err = r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM orders WHERE status = 'LOCKED' AND event_date = ?`, date).Scan(&orderCount)
	return demand, orderCount, err
}

// LoadRecipes membaca semua resep, dikunci berdasarkan output_item_id.
func (r *MRPRepo) LoadRecipes(ctx context.Context) (map[int]models.Recipe, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT r.id, r.output_item_id, r.batch_yield, b.component_item_id, b.qty_req
		FROM recipes r
		LEFT JOIN recipe_bom_items b ON b.recipe_id = r.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	recipes := map[int]models.Recipe{}
	for rows.Next() {
		var id, out int
		var yield float64
		var comp sql.NullInt64
		var qty sql.NullFloat64
		if err := rows.Scan(&id, &out, &yield, &comp, &qty); err != nil {
			return nil, err
		}
		rc, ok := recipes[out]
		if !ok {
			rc = models.Recipe{ID: id, OutputItemID: out, BatchYield: yield}
		}
		if comp.Valid {
			rc.Components = append(rc.Components, models.RecipeComponent{
				ComponentItemID: int(comp.Int64), QtyReq: qty.Float64,
			})
		}
		recipes[out] = rc
	}
	return recipes, rows.Err()
}

// LoadItems membaca item aktif beserta stok dan satuan belinya.
func (r *MRPRepo) LoadItems(ctx context.Context) (map[int]models.ItemInfo, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT i.id, i.name, i.item_type, COALESCE(i.stock_qty, 0), COALESCE(i.conversion_ratio, 0), COALESCE(u.code, '')
		FROM items i
		LEFT JOIN units u ON u.id = i.purchase_unit_id
		WHERE i.is_active = TRUE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := map[int]models.ItemInfo{}
	for rows.Next() {
		var it models.ItemInfo
		if err := rows.Scan(&it.ID, &it.Name, &it.ItemType, &it.StockQty, &it.ConversionRatio, &it.PurchaseUnit); err != nil {
			return nil, err
		}
		items[it.ID] = it
	}
	return items, rows.Err()
}

// SaveShoppingList membuat/menemukan production_run pada tanggal tsb lalu mengganti isi shopping_lists-nya (idempotent).
func (r *MRPRepo) SaveShoppingList(ctx context.Context, date string, lines []models.ShoppingListItem) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var runID int
	var status string
	err = tx.QueryRowContext(ctx,
		`SELECT id, status FROM production_runs WHERE production_date = ? FOR UPDATE`, date).Scan(&runID, &status)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		res, e := tx.ExecContext(ctx, `INSERT INTO production_runs (production_date, status) VALUES (?, 'PLANNED')`, date)
		if e != nil {
			return 0, e
		}
		id, _ := res.LastInsertId()
		runID = int(id)
	case err != nil:
		return 0, err
	case status == "DONE":
		return 0, ErrRunClosed
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM shopping_lists WHERE production_run_id = ?`, runID); err != nil {
		return 0, err
	}
	for _, l := range lines {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO shopping_lists
				(production_run_id, raw_item_id, gross_req, stock_on_hand, net_req, purchase_qty_rounded)
			VALUES (?, ?, ?, ?, ?, ?)`,
			runID, l.RawItemID, l.GrossReq, l.StockOnHand, l.NetReq, l.PurchaseQtyRounded); err != nil {
			return 0, err
		}
	}
	return runID, tx.Commit()
}

func (r *MRPRepo) ListRuns(ctx context.Context) ([]models.ProductionRun, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, DATE_FORMAT(production_date, '%Y-%m-%d'), status FROM production_runs ORDER BY production_date DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	runs := []models.ProductionRun{}
	for rows.Next() {
		var p models.ProductionRun
		if err := rows.Scan(&p.ID, &p.ProductionDate, &p.Status); err != nil {
			return nil, err
		}
		runs = append(runs, p)
	}
	return runs, rows.Err()
}

func (r *MRPRepo) GetShoppingList(ctx context.Context, runID int) ([]models.ShoppingListItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT s.id, s.production_run_id, s.raw_item_id, i.name, s.gross_req, s.stock_on_hand,
		       s.net_req, s.purchase_qty_rounded, COALESCE(u.code, '')
		FROM shopping_lists s
		JOIN items i ON i.id = s.raw_item_id
		LEFT JOIN units u ON u.id = i.purchase_unit_id
		WHERE s.production_run_id = ?
		ORDER BY i.name`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []models.ShoppingListItem{}
	for rows.Next() {
		var l models.ShoppingListItem
		if err := rows.Scan(&l.ID, &l.ProductionRunID, &l.RawItemID, &l.RawItemName, &l.GrossReq,
			&l.StockOnHand, &l.NetReq, &l.PurchaseQtyRounded, &l.PurchaseUnit); err != nil {
			return nil, err
		}
		list = append(list, l)
	}
	return list, rows.Err()
}
