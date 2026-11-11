package service

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"dapur-ibu-erp/internal/models"
	"dapur-ibu-erp/internal/repository"

	"github.com/go-sql-driver/mysql"
)

// Error bisnis modul pesanan. Pesan ErrOrderLocked sengaja disamakan dengan
// spesifikasi jobdesc agar bisa langsung ditampilkan di frontend.
var (
	ErrOrderLocked = errors.New("Pesanan sudah terkunci (Melewati batas Cutoff H-2) dan tidak dapat diubah!")
	ErrOrderDone   = errors.New("Pesanan sudah berstatus DONE dan tidak dapat diubah!")
)

// ValidationError menandai kesalahan input dari pengguna (HTTP 400).
type ValidationError struct {
	Message string
}

func (e ValidationError) Error() string { return e.Message }

func invalid(format string, args ...any) error {
	return ValidationError{Message: fmt.Sprintf(format, args...)}
}

const (
	dateLayout     = "2006-01-02"
	maxBOMDepth    = 6
	defaultLimit   = 200
	maxLimit       = 1000
	dateTimeLayout = "2006-01-02 15:04:05"
)

// ---------- Util tanggal ----------

// ParseDate menerima "YYYY-MM-DD" (juga "YYYY-MM-DD HH:MM:SS" / RFC3339)
// dan mengembalikan tanggal pada zona waktu lokal.
func ParseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, invalid("tanggal wajib diisi (format YYYY-MM-DD)")
	}
	// Ambil bagian tanggal saja bila yang dikirim datetime.
	if len(s) >= 10 {
		if t, err := time.ParseInLocation(dateLayout, s[:10], time.Local); err == nil {
			return t, nil
		}
	}
	if t, err := time.ParseInLocation(dateTimeLayout, s, time.Local); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Time{}, invalid("format tanggal tidak valid: %q (gunakan YYYY-MM-DD)", s)
}

// ComputeCutoffDate menghitung batas cutoff otomatis: H-1 pukul 00:00:00,
// yang berarti admin masih bisa mengubah pesanan sampai H-2 pukul 23:59:59.
// Contoh: event_date 2026-11-20 -> cutoff_date 2026-11-19 00:00:00.
func ComputeCutoffDate(eventDate time.Time) time.Time {
	y, m, d := eventDate.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, eventDate.Location()).AddDate(0, 0, -1)
}

// IsEditable menentukan apakah pesanan masih boleh diubah pada waktu `now`.
func IsEditable(o models.Order, now time.Time) bool {
	if o.Status == models.OrderStatusLocked || o.Status == models.OrderStatusDone {
		return false
	}
	return now.Before(o.CutoffDate)
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func dateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

// ---------- Validasi ----------

func normalizeStatus(s string, allowed ...string) (string, error) {
	s = strings.ToUpper(strings.TrimSpace(s))
	for _, a := range allowed {
		if s == a {
			return s, nil
		}
	}
	return "", invalid("status %q tidak valid (pilihan: %s)", s, strings.Join(allowed, ", "))
}

func validateItems(db *sql.DB, items []models.OrderItemInput) error {
	if len(items) == 0 {
		return invalid("pesanan minimal memiliki 1 menu")
	}
	seen := map[int64]bool{}
	for i, it := range items {
		if it.MenuItemID <= 0 {
			return invalid("items[%d]: menu_item_id wajib diisi", i)
		}
		if seen[it.MenuItemID] {
			return invalid("items[%d]: menu yang sama dimasukkan lebih dari sekali, gabungkan porsinya", i)
		}
		seen[it.MenuItemID] = true

		if it.Portions <= 0 {
			return invalid("items[%d]: portions harus lebih dari 0", i)
		}
		if it.SellingPrice < 0 {
			return invalid("items[%d]: selling_price tidak boleh negatif", i)
		}

		name, itemType, active, err := repository.GetMenuItemInfo(db, it.MenuItemID)
		if errors.Is(err, repository.ErrMenuItemNotFound) {
			return invalid("items[%d]: menu dengan id %d tidak ditemukan", i, it.MenuItemID)
		}
		if err != nil {
			return err
		}
		if !active {
			return invalid("items[%d]: menu %q sudah nonaktif", i, name)
		}
		if itemType != "MENU" {
			return invalid("items[%d]: item %q bertipe %s, pesanan hanya boleh berisi item bertipe MENU", i, name, itemType)
		}
	}
	return nil
}

// generateOrderNum membuat nomor pesanan otomatis jika admin tidak mengisinya.
func generateOrderNum(now time.Time) string {
	return fmt.Sprintf("ORD-%s-%03d", now.Format("20060102-150405"), now.Nanosecond()/1_000_000)
}

// mapWriteError menerjemahkan error MySQL menjadi pesan yang ramah pengguna.
func mapWriteError(err error) error {
	var me *mysql.MySQLError
	if errors.As(err, &me) {
		switch me.Number {
		case 1062: // duplicate entry
			return invalid("order_num sudah dipakai, gunakan nomor lain")
		case 1452: // foreign key gagal
			return invalid("data referensi tidak valid (menu tidak ditemukan)")
		}
	}
	return err
}

// ---------- CRUD Pesanan ----------

// CreateOrder memvalidasi input, menghitung cutoff_date otomatis, lalu
// menyimpan header + detail menu dalam satu transaksi.
func CreateOrder(db *sql.DB, req models.OrderRequest) (*models.Order, error) {
	customer := strings.TrimSpace(req.Customer)
	if customer == "" {
		return nil, invalid("nama pemesan (customer) wajib diisi")
	}

	eventDate, err := ParseDate(req.EventDate)
	if err != nil {
		return nil, err
	}
	if eventDate.Before(dateOnly(time.Now())) {
		return nil, invalid("tanggal acara tidak boleh di masa lalu")
	}

	status := models.OrderStatusDraft
	if strings.TrimSpace(req.Status) != "" {
		status, err = normalizeStatus(req.Status, models.OrderStatusDraft, models.OrderStatusConfirmed)
		if err != nil {
			return nil, err
		}
	}

	if err := validateItems(db, req.Items); err != nil {
		return nil, err
	}

	orderNum := strings.TrimSpace(req.OrderNum)
	if orderNum == "" {
		orderNum = generateOrderNum(time.Now())
	}

	order := models.Order{
		OrderNum:   orderNum,
		Customer:   customer,
		EventDate:  eventDate.Format(dateLayout),
		CutoffDate: ComputeCutoffDate(eventDate),
		Status:     status,
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	id, err := repository.CreateOrder(tx, order)
	if err != nil {
		return nil, mapWriteError(err)
	}
	if err := repository.InsertOrderItems(tx, id, req.Items); err != nil {
		return nil, mapWriteError(err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return GetOrderDetail(db, id)
}

// GetOrderDetail mengambil header + detail menu satu pesanan.
func GetOrderDetail(db *sql.DB, id int64) (*models.Order, error) {
	o, err := repository.GetOrderByID(db, id)
	if err != nil {
		return nil, err
	}
	items, err := repository.ListOrderItems(db, id)
	if err != nil {
		return nil, err
	}
	o.Items = items
	o.Editable = IsEditable(o, time.Now())
	return &o, nil
}

// ListOrders mengambil daftar pesanan dengan filter opsional.
func ListOrders(db *sql.DB, f models.OrderFilter) ([]models.Order, error) {
	if f.Status != "" {
		s, err := normalizeStatus(f.Status,
			models.OrderStatusDraft, models.OrderStatusConfirmed,
			models.OrderStatusLocked, models.OrderStatusDone)
		if err != nil {
			return nil, err
		}
		f.Status = s
	}
	if f.EventDate != "" {
		d, err := ParseDate(f.EventDate)
		if err != nil {
			return nil, err
		}
		f.EventDate = d.Format(dateLayout)
	}
	if f.Limit <= 0 {
		f.Limit = defaultLimit
	}
	if f.Limit > maxLimit {
		f.Limit = maxLimit
	}

	list, err := repository.ListOrders(db, f)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	for i := range list {
		list[i].Editable = IsEditable(list[i], now)
	}
	return list, nil
}

// UpdateOrder mengubah pesanan selama belum LOCKED/DONE dan belum melewati cutoff.
// Field yang dikirim kosong akan dipertahankan (partial update).
func UpdateOrder(db *sql.DB, id int64, req models.OrderRequest) (*models.Order, error) {
	current, err := repository.GetOrderByID(db, id)
	if err != nil {
		return nil, err
	}

	// PROTEKSI KRUSIAL: pesanan LOCKED/DONE tidak boleh diubah.
	if current.Status == models.OrderStatusLocked {
		return nil, ErrOrderLocked
	}
	if current.Status == models.OrderStatusDone {
		return nil, ErrOrderDone
	}
	// PROTEKSI CUTOFF: tolak jika waktu sekarang sudah melewati cutoff_date.
	if !time.Now().Before(current.CutoffDate) {
		return nil, ErrOrderLocked
	}

	if c := strings.TrimSpace(req.Customer); c != "" {
		current.Customer = c
	}

	if strings.TrimSpace(req.EventDate) != "" {
		eventDate, err := ParseDate(req.EventDate)
		if err != nil {
			return nil, err
		}
		if eventDate.Before(dateOnly(time.Now())) {
			return nil, invalid("tanggal acara tidak boleh di masa lalu")
		}
		current.EventDate = eventDate.Format(dateLayout)
		current.CutoffDate = ComputeCutoffDate(eventDate)
	}

	if strings.TrimSpace(req.Status) != "" {
		status, err := normalizeStatus(req.Status,
			models.OrderStatusDraft, models.OrderStatusConfirmed, models.OrderStatusDone)
		if err != nil {
			return nil, err
		}
		current.Status = status
	}

	replaceItems := req.Items != nil
	if replaceItems {
		if err := validateItems(db, req.Items); err != nil {
			return nil, err
		}
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if err := repository.UpdateOrderHeader(tx, current); err != nil {
		return nil, mapWriteError(err)
	}
	if replaceItems {
		if err := repository.ReplaceOrderItems(tx, current.ID, req.Items); err != nil {
			return nil, mapWriteError(err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return GetOrderDetail(db, id)
}

// DeleteOrder menghapus pesanan (hanya jika masih boleh diubah).
func DeleteOrder(db *sql.DB, id int64) error {
	current, err := repository.GetOrderByID(db, id)
	if err != nil {
		return err
	}
	if current.Status == models.OrderStatusLocked {
		return ErrOrderLocked
	}
	if current.Status == models.OrderStatusDone {
		return ErrOrderDone
	}
	if !time.Now().Before(current.CutoffDate) {
		return ErrOrderLocked
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := repository.DeleteOrder(tx, current.ID); err != nil {
		return err
	}
	return tx.Commit()
}

// ---------- Rekap ----------

// GetOrderSummary menghitung total porsi per menu pada satu tanggal acara.
func GetOrderSummary(db *sql.DB, dateStr string, status string) (*models.OrderSummary, error) {
	eventDate, err := ParseDate(dateStr)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(status) != "" {
		status, err = normalizeStatus(status,
			models.OrderStatusDraft, models.OrderStatusConfirmed,
			models.OrderStatusLocked, models.OrderStatusDone)
		if err != nil {
			return nil, err
		}
	}
	key := eventDate.Format(dateLayout)

	menus, err := repository.AggregateMenusByEventDate(db, key, status)
	if err != nil {
		return nil, err
	}
	orders, portions, value, err := repository.SumOrdersByEventDate(db, key, status)
	if err != nil {
		return nil, err
	}

	return &models.OrderSummary{
		EventDate:     key,
		TotalOrders:   orders,
		TotalPortions: portions,
		TotalValue:    round2(value),
		Menus:         menus,
	}, nil
}

// ---------- Automasi Cutoff H-2 ----------

// LockDueOrders mencari semua pesanan DRAFT/CONFIRMED yang cutoff_date-nya
// sudah terlewati, mengunci statusnya, dan menyimpan snapshot HPP per porsi.
func LockDueOrders(db *sql.DB, now time.Time) (*models.CutoffResult, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	due, err := repository.ListDueOrders(tx, now)
	if err != nil {
		return nil, err
	}

	result := &models.CutoffResult{
		Orders: []models.LockedOrderInfo{},
		RunAt:  now,
	}

	for _, o := range due {
		totalPortions, err := repository.SumOrderPortions(tx, o.ID)
		if err != nil {
			return nil, err
		}

		hppPerPortion := 0.0
		if totalPortions > 0 {
			menus, err := repository.ListOrderMenuPortions(tx, o.ID)
			if err != nil {
				return nil, err
			}
			totalCost := 0.0
			for _, m := range menus {
				hpp, err := calcMenuHPP(tx, m.MenuItemID)
				if err != nil {
					return nil, err
				}
				totalCost += hpp * float64(m.Portions)
			}
			hppPerPortion = round2(totalCost / float64(totalPortions))
		}

		if err := repository.LockOrder(tx, o.ID, hppPerPortion); err != nil {
			return nil, err
		}

		result.Orders = append(result.Orders, models.LockedOrderInfo{
			OrderID:             o.ID,
			OrderNum:            o.OrderNum,
			EventDate:           o.EventDate,
			TotalPortions:       totalPortions,
			LockedHPPPerPortion: hppPerPortion,
		})
	}
	result.LockedCount = len(result.Orders)

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

// calcMenuHPP menghitung HPP per porsi sebuah item. Urutan sumber harga:
// 1) moving_avg_cost item (hasil roll-up modul resep / master bahan),
// 2) std_hpp resep terakhir,
// 3) hitung rekursif dari komponen resep hingga bahan mentah.
func calcMenuHPP(q *sql.Tx, itemID int64) (float64, error) {
	return resolveItemCost(q, itemID, map[int64]bool{}, 0)
}

func resolveItemCost(q *sql.Tx, itemID int64, visited map[int64]bool, depth int) (float64, error) {
	if depth > maxBOMDepth || visited[itemID] {
		return 0, nil // jaga-jaga terhadap resep melingkar
	}
	visited[itemID] = true
	defer delete(visited, itemID)

	cost, _, err := repository.GetItemCost(q, itemID)
	if errors.Is(err, repository.ErrMenuItemNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if cost > 0 {
		return cost, nil
	}

	recipeID, batchYield, stdHPP, err := repository.GetLatestRecipe(q, itemID)
	if errors.Is(err, repository.ErrRecipeNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if stdHPP > 0 {
		return stdHPP, nil
	}
	if batchYield <= 0 {
		return 0, nil
	}

	components, err := repository.ListRecipeComponents(q, recipeID)
	if err != nil {
		return 0, err
	}
	total := 0.0
	for _, c := range components {
		compCost, err := resolveItemCost(q, c.ComponentItemID, visited, depth+1)
		if err != nil {
			return 0, err
		}
		total += compCost * c.QtyReq
	}
	return total / batchYield, nil
}
