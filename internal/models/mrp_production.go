package models

// ===== Tabel production_runs & shopping_lists =====

type ProductionRun struct {
	ID             int    `json:"id"`
	ProductionDate string `json:"production_date"`
	Status         string `json:"status"` // PLANNED | DONE
}

type ShoppingListItem struct {
	ID                 int     `json:"id,omitempty"`
	ProductionRunID    int     `json:"production_run_id,omitempty"`
	RawItemID          int     `json:"raw_item_id"`
	RawItemName        string  `json:"raw_item_name,omitempty"`
	GrossReq           float64 `json:"gross_req"`            // satuan pakai (gr/ml)
	StockOnHand        float64 `json:"stock_on_hand"`        // satuan pakai
	NetReq             float64 `json:"net_req"`              // satuan pakai
	PurchaseQtyRounded int     `json:"purchase_qty_rounded"` // satuan beli, hasil CEIL
	PurchaseUnit       string  `json:"purchase_unit,omitempty"`
}

// ===== Struktur bantu untuk Exploding BOM (dibaca dari recipes, recipe_bom_items, items) =====

type Recipe struct {
	ID           int
	OutputItemID int
	BatchYield   float64
	Components   []RecipeComponent
}

type RecipeComponent struct {
	ComponentItemID int
	QtyReq          float64 // kebutuhan per 1 batch
}

type ItemInfo struct {
	ID              int
	Name            string
	ItemType        string // RAW | WIP | COMP | MENU
	StockQty        float64
	ConversionRatio float64 // 1 satuan beli = ConversionRatio satuan pakai
	PurchaseUnit    string
}

// ===== Tabel purchases (+ baris barang yang tersimpan sebagai stock_movements IN) =====

type PurchaseLine struct {
	ItemID          int     `json:"item_id"`
	QtyPurchaseUnit float64 `json:"qty_purchase_unit"` // jumlah dalam satuan beli (mis. 2 karung)
	UnitPrice       float64 `json:"unit_price"`        // harga riil per satuan beli
}

type PurchaseRequest struct {
	InvoiceNo    string         `json:"invoice_no"`
	Supplier     string         `json:"supplier"`
	PurchaseDate string         `json:"purchase_date"` // YYYY-MM-DD, default hari ini
	Items        []PurchaseLine `json:"items"`
}

type PurchaseLineResult struct {
	ItemID       int     `json:"item_id"`
	QtyUsageUnit float64 `json:"qty_usage_unit"` // sudah dikonversi ke satuan pakai
	CostPerUsage float64 `json:"cost_per_usage_unit"`
	OldAvgCost   float64 `json:"old_moving_avg_cost"`
	NewAvgCost   float64 `json:"new_moving_avg_cost"`
	StockAfter   float64 `json:"stock_after"`
}

type Purchase struct {
	ID           int                  `json:"id"`
	InvoiceNo    string               `json:"invoice_no"`
	Supplier     string               `json:"supplier"`
	PurchaseDate string               `json:"purchase_date"`
	TotalAmount  float64              `json:"total_amount"`
	Lines        []PurchaseLineResult `json:"lines,omitempty"`
}
