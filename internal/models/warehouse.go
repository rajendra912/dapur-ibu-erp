package models

import "github.com/shopspring/decimal"

type Movement struct {
	ItemID          int64           `json:"item_id"`
	MoveType        string          `json:"move_type"`  // IN | OUT | WASTE
	QtyChange       decimal.Decimal `json:"qty_change"` // satuan pakai, positif
	UnitCost        decimal.Decimal `json:"unit_cost"`  // opsional, hanya IN, per satuan pakai
	PurchaseID      *int64          `json:"purchase_id,omitempty"`
	ProductionRunID *int64          `json:"production_run_id,omitempty"`
}
