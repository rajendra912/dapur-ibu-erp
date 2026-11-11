package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/rajendra912/dapur-ibu-erp/internal/models"
)

// ordQuerier membuat fungsi repository bisa dipakai untuk *sql.DB maupun *sql.Tx.
type ordQuerier interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

// OrderMenuPortion adalah potongan data menu + porsi untuk perhitungan HPP cutoff.
type OrderMenuPortion struct {
	MenuItemID int64
	Portions   int
}

const orderSelectBase = `
	SELECT o.id, o.order_num, o.customer, o.event_date, o.cutoff_date, o.status,
	       COALESCE(o.locked_hpp_per_portion, 0),
	       COALESCE(agg.total_portions, 0), COALESCE(agg.total_value, 0)
	FROM orders o
	LEFT JOIN (
		SELECT order_id, SUM(portions) AS total_portions,
		       SUM(portions * selling_price) AS total_value
		FROM order_items
		GROUP BY order_id
	) agg ON agg.order_id = o.id`

func scanOrder(sc interface{ Scan(dest ...any) error }) (models.Order, error) {
	var o models.Order
	var eventDate, cutoffDate time.Time
	err := sc.Scan(&o.ID, &o.OrderNum, &o.Customer, &eventDate, &cutoffDate,
		&o.Status, &o.LockedHPPPerPortion, &o.TotalPortions, &o.TotalValue)
	if err != nil {
		return o, err
	}
	o.EventDate = eventDate.Format("2006-01-02")
	o.CutoffDate = cutoffDate
	return o, nil
}

// ListOrders mengambil daftar pesanan beserta total porsi dan nilai omzetnya.
func ListOrders(db ordQuerier, f models.OrderFilter) ([]models.Order, error) {
	var where []string
	var args []any

	if f.Status != "" {
		where = append(where, "o.status = ?")
		args = append(args, f.Status)
	}
	if f.EventDate != "" {
		where = append(where, "o.event_date = ?")
		args = append(args, f.EventDate)
	}
	if f.Search != "" {
		where = append(where, "(o.customer LIKE ? OR o.order_num LIKE ?)")
		like := "%" + f.Search + "%"
		args = append(args, like, like)
	}

	q := orderSelectBase
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	q += " ORDER BY o.event_date ASC, o.id DESC LIMIT ?"
	args = append(args, f.Limit)

	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []models.Order{}
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// GetOrderByID mengambil satu header pesanan (tanpa detail item).
func GetOrderByID(db ordQuerier, id int64) (models.Order, error) {
	o, err := scanOrder(db.QueryRow(orderSelectBase+" WHERE o.id = ?", id))
	if err == sql.ErrNoRows {
		return o, ErrOrderNotFound
	}
	return o, err
}

// GetOrderByIdentifier mencari pesanan berdasarkan id numerik ATAU nomor pesanan.
func GetOrderByIdentifier(db ordQuerier, id int64, orderNum string) (models.Order, error) {
	if id > 0 {
		return GetOrderByID(db, id)
	}
	o, err := scanOrder(db.QueryRow(orderSelectBase+" WHERE o.order_num = ?", orderNum))
	if err == sql.ErrNoRows {
		return o, ErrOrderNotFound
	}
	return o, err
}

// CreateOrder menyimpan header pesanan lalu mengembalikan ID barunya.
func CreateOrder(q ordQuerier, o models.Order) (int64, error) {
	res, err := q.Exec(`
		INSERT INTO orders (order_num, customer, event_date, cutoff_date, status, locked_hpp_per_portion)
		VALUES (?, ?, ?, ?, ?, 0)`,
		o.OrderNum, o.Customer, o.EventDate, o.CutoffDate, o.Status)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// InsertOrderItems menyimpan seluruh detail menu milik satu pesanan.
func InsertOrderItems(q ordQuerier, orderID int64, items []models.OrderItemInput) error {
	for _, it := range items {
		if _, err := q.Exec(`
			INSERT INTO order_items (order_id, menu_item_id, portions, selling_price)
			VALUES (?, ?, ?, ?)`,
			orderID, it.MenuItemID, it.Portions, it.SellingPrice); err != nil {
			return err
		}
	}
	return nil
}

// ReplaceOrderItems menghapus detail lama lalu menulis ulang detail pesanan.
func ReplaceOrderItems(q ordQuerier, orderID int64, items []models.OrderItemInput) error {
	if _, err := q.Exec(`DELETE FROM order_items WHERE order_id = ?`, orderID); err != nil {
		return err
	}
	return InsertOrderItems(q, orderID, items)
}

// UpdateOrderHeader memperbarui data utama pesanan.
func UpdateOrderHeader(q ordQuerier, o models.Order) error {
	_, err := q.Exec(`
		UPDATE orders
		SET order_num = ?, customer = ?, event_date = ?, cutoff_date = ?, status = ?
		WHERE id = ?`,
		o.OrderNum, o.Customer, o.EventDate, o.CutoffDate, o.Status, o.ID)
	return err
}

// SetOrderStatus mengubah status pesanan (dipakai untuk menandai DONE).
func SetOrderStatus(q ordQuerier, id int64, status string) error {
	_, err := q.Exec(`UPDATE orders SET status = ? WHERE id = ?`, status, id)
	return err
}

// DeleteOrder menghapus header beserta detailnya (detail lebih dulu agar FK aman).
func DeleteOrder(q ordQuerier, id int64) error {
	if _, err := q.Exec(`DELETE FROM order_items WHERE order_id = ?`, id); err != nil {
		return err
	}
	_, err := q.Exec(`DELETE FROM orders WHERE id = ?`, id)
	return err
}

// ListOrderItems mengambil detail menu pesanan beserta nama menunya.
func ListOrderItems(db ordQuerier, orderID int64) ([]models.OrderItem, error) {
	rows, err := db.Query(`
		SELECT oi.id, oi.order_id, oi.menu_item_id, i.name, oi.portions, oi.selling_price
		FROM order_items oi
		JOIN items i ON i.id = oi.menu_item_id
		WHERE oi.order_id = ?
		ORDER BY oi.id`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []models.OrderItem{}
	for rows.Next() {
		var it models.OrderItem
		if err := rows.Scan(&it.ID, &it.OrderID, &it.MenuItemID, &it.MenuName,
			&it.Portions, &it.SellingPrice); err != nil {
			return nil, err
		}
		it.Subtotal = float64(it.Portions) * it.SellingPrice
		out = append(out, it)
	}
	return out, rows.Err()
}

// GetMenuItemInfo dipakai untuk memvalidasi bahwa menu benar-benar ada,
// masih aktif, dan bertipe MENU.
func GetMenuItemInfo(q ordQuerier, id int64) (name string, itemType string, active bool, err error) {
	err = q.QueryRow(`SELECT name, item_type, is_active FROM items WHERE id = ?`, id).
		Scan(&name, &itemType, &active)
	if err == sql.ErrNoRows {
		return "", "", false, ErrMenuItemNotFound
	}
	return name, itemType, active, err
}

// GetItemCost mengambil HPP satuan terkini (moving_avg_cost) satu item.
func GetItemCost(q ordQuerier, id int64) (cost float64, itemType string, err error) {
	err = q.QueryRow(`SELECT COALESCE(moving_avg_cost, 0), item_type FROM items WHERE id = ?`, id).
		Scan(&cost, &itemType)
	if err == sql.ErrNoRows {
		return 0, "", ErrMenuItemNotFound
	}
	return cost, itemType, err
}

// GetLatestRecipe mengambil resep terakhir milik sebuah output item.
func GetLatestRecipe(q ordQuerier, outputItemID int64) (recipeID int64, batchYield float64, stdHPP float64, err error) {
	err = q.QueryRow(`
		SELECT id, COALESCE(batch_yield, 0), COALESCE(std_hpp, 0)
		FROM recipes
		WHERE output_item_id = ?
		ORDER BY id DESC
		LIMIT 1`, outputItemID).Scan(&recipeID, &batchYield, &stdHPP)
	if err == sql.ErrNoRows {
		return 0, 0, 0, ErrRecipeNotFound
	}
	return recipeID, batchYield, stdHPP, err
}

// RecipeComponent adalah satu baris komponen penyusun resep.
type RecipeComponent struct {
	ComponentItemID int64
	QtyReq          float64
}

// ListRecipeComponents mengambil seluruh komponen dari satu resep.
func ListRecipeComponents(q ordQuerier, recipeID int64) ([]RecipeComponent, error) {
	rows, err := q.Query(`SELECT component_item_id, qty_req FROM recipe_bom_items WHERE recipe_id = ?`, recipeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []RecipeComponent{}
	for rows.Next() {
		var c RecipeComponent
		if err := rows.Scan(&c.ComponentItemID, &c.QtyReq); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ListOrderMenuPortions mengambil daftar menu + porsi milik satu pesanan.
func ListOrderMenuPortions(q ordQuerier, orderID int64) ([]OrderMenuPortion, error) {
	rows, err := q.Query(`SELECT menu_item_id, portions FROM order_items WHERE order_id = ?`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []OrderMenuPortion{}
	for rows.Next() {
		var p OrderMenuPortion
		if err := rows.Scan(&p.MenuItemID, &p.Portions); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ListDueOrders mengambil pesanan yang sudah melewati cutoff namun belum dikunci.
// FOR UPDATE dipakai agar worker dan endpoint manual tidak memproses baris yang sama.
func ListDueOrders(tx *sql.Tx, now time.Time) ([]models.Order, error) {
	rows, err := tx.Query(`
		SELECT id, order_num, customer, event_date, cutoff_date, status,
		       COALESCE(locked_hpp_per_portion, 0)
		FROM orders
		WHERE status IN ('DRAFT', 'CONFIRMED') AND cutoff_date <= ?
		ORDER BY event_date ASC, id ASC
		FOR UPDATE`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []models.Order{}
	for rows.Next() {
		var o models.Order
		var eventDate, cutoffDate time.Time
		if err := rows.Scan(&o.ID, &o.OrderNum, &o.Customer, &eventDate,
			&cutoffDate, &o.Status, &o.LockedHPPPerPortion); err != nil {
			return nil, err
		}
		o.EventDate = eventDate.Format("2006-01-02")
		o.CutoffDate = cutoffDate
		out = append(out, o)
	}
	return out, rows.Err()
}

// LockOrder mengunci pesanan sekaligus menyimpan snapshot HPP per porsi.
func LockOrder(tx *sql.Tx, id int64, hppPerPortion float64) error {
	_, err := tx.Exec(`
		UPDATE orders
		SET status = 'LOCKED', locked_hpp_per_portion = ?
		WHERE id = ? AND status IN ('DRAFT', 'CONFIRMED')`, hppPerPortion, id)
	return err
}

// SumOrderPortions menghitung total porsi satu pesanan.
func SumOrderPortions(q ordQuerier, orderID int64) (int, error) {
	var total sql.NullInt64
	err := q.QueryRow(`SELECT SUM(portions) FROM order_items WHERE order_id = ?`, orderID).Scan(&total)
	if err != nil {
		return 0, err
	}
	if !total.Valid {
		return 0, nil
	}
	return int(total.Int64), nil
}

// AggregateMenusByEventDate mengelompokkan total porsi per menu pada satu tanggal acara.
func AggregateMenusByEventDate(db ordQuerier, eventDate string, status string) ([]models.MenuAggregate, error) {
	q := `
		SELECT oi.menu_item_id, i.name, SUM(oi.portions) AS total_portions,
		       COUNT(DISTINCT o.id) AS order_count
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		JOIN items i ON i.id = oi.menu_item_id
		WHERE o.event_date = ?`
	args := []any{eventDate}
	if status != "" {
		q += " AND o.status = ?"
		args = append(args, status)
	}
	q += " GROUP BY oi.menu_item_id, i.name ORDER BY total_portions DESC"

	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []models.MenuAggregate{}
	for rows.Next() {
		var m models.MenuAggregate
		if err := rows.Scan(&m.MenuItemID, &m.MenuName, &m.TotalPortions, &m.OrderCount); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// SumOrdersByEventDate menghitung jumlah pesanan, total porsi, dan nilai omzet
// pada satu tanggal acara.
func SumOrdersByEventDate(db ordQuerier, eventDate string, status string) (orders int, portions int, value float64, err error) {
	q := `
		SELECT COUNT(DISTINCT o.id), COALESCE(SUM(oi.portions), 0),
		       COALESCE(SUM(oi.portions * oi.selling_price), 0)
		FROM orders o
		LEFT JOIN order_items oi ON oi.order_id = o.id
		WHERE o.event_date = ?`
	args := []any{eventDate}
	if status != "" {
		q += " AND o.status = ?"
		args = append(args, status)
	}
	err = db.QueryRow(q, args...).Scan(&orders, &portions, &value)
	return orders, portions, value, err
}

// ErrOrderNotFound dipakai bersama oleh repository dan service.
var ErrOrderNotFound = fmt.Errorf("pesanan tidak ditemukan")

// ErrMenuItemNotFound menandai menu yang tidak ada / sudah nonaktif.
var ErrMenuItemNotFound = fmt.Errorf("menu tidak ditemukan")

// ErrRecipeNotFound menandai item yang belum memiliki resep.
var ErrRecipeNotFound = fmt.Errorf("resep tidak ditemukan")
