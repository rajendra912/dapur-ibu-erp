package models

import "github.com/shopspring/decimal"

type Unit struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type Item struct {
	ID              int64           `json:"id"`
	SKU             string          `json:"sku"`
	Name            string          `json:"name"`
	ItemType        string          `json:"item_type"` // RAW | WIP | COMP | MENU
	UsageUnitID     int64           `json:"usage_unit_id"`
	PurchaseUnitID  int64           `json:"purchase_unit_id"`
	UsageUnit       string          `json:"usage_unit,omitempty"`
	PurchaseUnit    string          `json:"purchase_unit,omitempty"`
	ConversionRatio decimal.Decimal `json:"conversion_ratio"`
	StockQty        decimal.Decimal `json:"stock_qty"`
	MovingAvgCost   decimal.Decimal `json:"moving_avg_cost"`
}
