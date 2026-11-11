package service

import (
	"database/sql"
	"errors"

	"dapur-ibu-erp/internal/models"
	"dapur-ibu-erp/internal/repository"

	"github.com/shopspring/decimal"
)

var (
	ErrItemNotFound = errors.New("bahan tidak ditemukan atau nonaktif")
	ErrStokKurang   = errors.New("Stok gudang tidak mencukupi")
	ErrJenisMutasi  = errors.New("move_type harus IN, OUT, atau WASTE")
	ErrQtyInvalid   = errors.New("qty_change harus lebih dari 0")
)

// Dipanggil di dalam transaksi milik pemanggil. Mengembalikan saldo terbaru.
func RecordMovement(tx *sql.Tx, m models.Movement) (decimal.Decimal, error) {
	if !m.QtyChange.IsPositive() {
		return decimal.Zero, ErrQtyInvalid
	}
	stock, avg, err := repository.LockItem(tx, m.ItemID)
	if err == sql.ErrNoRows {
		return decimal.Zero, ErrItemNotFound
	}
	if err != nil {
		return decimal.Zero, err
	}

	var balance decimal.Decimal
	switch m.MoveType {
	case "IN":
		if m.UnitCost.IsPositive() {
			if _, err = ApplyMovingAverage(tx, m.ItemID, stock, avg, m.QtyChange, m.UnitCost); err != nil {
				return decimal.Zero, err
			}
		}
		balance = stock.Add(m.QtyChange)
	case "OUT", "WASTE":
		if stock.LessThan(m.QtyChange) {
			return decimal.Zero, ErrStokKurang
		}
		balance = stock.Sub(m.QtyChange)
	default:
		return decimal.Zero, ErrJenisMutasi
	}
	balance = balance.Round(2)

	if err = repository.SaveItemStock(tx, m.ItemID, balance); err != nil {
		return decimal.Zero, err
	}
	if err = repository.InsertMovement(tx, m, balance); err != nil {
		return decimal.Zero, err
	}
	return balance, nil
}
