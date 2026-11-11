package models

import "time"

// Status pesanan sesuai ENUM pada tabel `orders`.
const (
	OrderStatusDraft     = "DRAFT"
	OrderStatusConfirmed = "CONFIRMED"
	OrderStatusLocked    = "LOCKED"
	OrderStatusDone      = "DONE"
)

// Order merepresentasikan satu baris tabel `orders` (header pesanan).
type Order struct {
	ID       int64  `json:"id"`
	OrderNum string `json:"order_num"`
	Customer string `json:"customer"`
	// EventDate selalu berformat YYYY-MM-DD di JSON.
	EventDate string `json:"event_date"`
	// CutoffDate dihitung otomatis oleh sistem (H-1 pukul 00:00:00,
	// yang artinya batas akhir perubahan adalah H-2 pukul 23:59:59).
	CutoffDate time.Time `json:"cutoff_date"`
	Status     string    `json:"status"`
	// Snapshot HPP per porsi saat pesanan dikunci (rata-rata tertimbang
	// dari seluruh menu yang dipesan).
	LockedHPPPerPortion float64 `json:"locked_hpp_per_portion"`
	// Field ringkasan (hasil agregasi tabel order_items), bukan kolom database.
	TotalPortions int     `json:"total_portions"`
	TotalValue    float64 `json:"total_value"`
	// Editable = masih boleh diubah admin (belum LOCKED/DONE dan belum lewat cutoff).
	Editable bool        `json:"editable"`
	Items    []OrderItem `json:"items,omitempty"`
}

// OrderItem merepresentasikan satu baris tabel `order_items` (detail menu).
type OrderItem struct {
	ID           int64   `json:"id"`
	OrderID      int64   `json:"order_id,omitempty"`
	MenuItemID   int64   `json:"menu_item_id"`
	MenuName     string  `json:"menu_name,omitempty"`
	Portions     int     `json:"portions"`
	SellingPrice float64 `json:"selling_price"`
	Subtotal     float64 `json:"subtotal,omitempty"`
}

// OrderItemInput adalah payload satu detail menu saat membuat/mengubah pesanan.
type OrderItemInput struct {
	MenuItemID   int64   `json:"menu_item_id"`
	Portions     int     `json:"portions"`
	SellingPrice float64 `json:"selling_price"`
}

// OrderRequest adalah payload untuk POST /api/orders dan PUT /api/orders/{id}.
// Pada PUT, field yang dikosongkan (string kosong / items null) tidak diubah.
type OrderRequest struct {
	OrderNum  string           `json:"order_num"`
	Customer  string           `json:"customer"`
	EventDate string           `json:"event_date"`
	Status    string           `json:"status"`
	Items     []OrderItemInput `json:"items"`
}

// OrderFilter dipakai untuk query GET /api/orders.
type OrderFilter struct {
	Status    string
	EventDate string
	Search    string
	Limit     int
}

// MenuAggregate adalah rekap total porsi per menu pada satu tanggal acara.
type MenuAggregate struct {
	MenuItemID    int64  `json:"menu_item_id"`
	MenuName      string `json:"menu_name"`
	TotalPortions int    `json:"total_portions"`
	OrderCount    int    `json:"order_count"`
}

// OrderSummary adalah rekap pesanan per tanggal acara (dipakai dashboard
// dan menjadi bahan agregasi Level 0 untuk modul MRP).
type OrderSummary struct {
	EventDate     string          `json:"event_date"`
	TotalOrders   int             `json:"total_orders"`
	TotalPortions int             `json:"total_portions"`
	TotalValue    float64         `json:"total_value"`
	Menus         []MenuAggregate `json:"menus"`
}

// LockedOrderInfo adalah ringkasan satu pesanan yang baru saja dikunci worker.
type LockedOrderInfo struct {
	OrderID             int64   `json:"order_id"`
	OrderNum            string  `json:"order_num"`
	EventDate           string  `json:"event_date"`
	TotalPortions       int     `json:"total_portions"`
	LockedHPPPerPortion float64 `json:"locked_hpp_per_portion"`
}

// CutoffResult adalah hasil eksekusi cutoff (worker maupun endpoint manual).
type CutoffResult struct {
	LockedCount int               `json:"locked_count"`
	Orders      []LockedOrderInfo `json:"orders"`
	RunAt       time.Time         `json:"run_at"`
}
