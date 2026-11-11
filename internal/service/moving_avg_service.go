package service

import (
	"database/sql"

	"dapur-ibu-erp/internal/repository"

	"github.com/shopspring/decimal"
)

// Panggil SEBELUM stok ditambah (pakai stok lama). Mengembalikan rata-rata baru.
func ApplyMovingAverage(tx *sql.Tx, itemID int64, oldStock, oldAvg, qtyIn, unitCost decimal.Decimal) (decimal.Decimal, error) {
	newAvg := oldStock.Mul(oldAvg).Add(qtyIn.Mul(unitCost)).
		Div(oldStock.Add(qtyIn)).Round(4)
	if newAvg.Equal(oldAvg) {
		return oldAvg, nil
	}
	if err := repository.InsertPriceLog(tx, itemID, oldAvg, newAvg); err != nil {
		return decimal.Zero, err
	}
	if err := repository.SaveItemCost(tx, itemID, newAvg); err != nil {
		return decimal.Zero, err
	}
	return newAvg, nil
}
