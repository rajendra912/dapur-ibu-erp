package service

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/rajendra912/dapur-ibu-erp/internal/models"
	"github.com/rajendra912/dapur-ibu-erp/internal/repository"
)

// CalcMovingAverage: HPP baru = ((stok lama x harga lama) + (stok baru x harga baru)) / (stok lama + stok baru)
func CalcMovingAverage(oldQty, oldCost, inQty, inCost float64) float64 {
	if oldQty <= 0 {
		return inCost
	}
	total := oldQty + inQty
	if total <= 0 {
		return inCost
	}
	return (oldQty*oldCost + inQty*inCost) / total
}

type PurchaseService struct{ repo *repository.PurchaseRepo }

func NewPurchaseService(repo *repository.PurchaseRepo) *PurchaseService {
	return &PurchaseService{repo: repo}
}

func (s *PurchaseService) Create(ctx context.Context, req models.PurchaseRequest) (*models.Purchase, error) {
	req.InvoiceNo = strings.TrimSpace(req.InvoiceNo)
	if req.InvoiceNo == "" {
		return nil, validationErr("invoice_no wajib diisi")
	}
	if req.PurchaseDate == "" {
		req.PurchaseDate = time.Now().Format("2006-01-02")
	} else if _, err := time.Parse("2006-01-02", req.PurchaseDate); err != nil {
		return nil, validationErr("purchase_date harus berformat YYYY-MM-DD")
	}
	if len(req.Items) == 0 {
		return nil, validationErr("items tidak boleh kosong")
	}
	var total float64
	for i, l := range req.Items {
		if l.ItemID <= 0 || l.QtyPurchaseUnit <= 0 || l.UnitPrice < 0 {
			return nil, validationErr("items[%d]: item_id, qty_purchase_unit (>0), dan unit_price (>=0) harus valid", i)
		}
		total += l.QtyPurchaseUnit * l.UnitPrice
	}

	result := &models.Purchase{
		InvoiceNo: req.InvoiceNo, Supplier: req.Supplier, PurchaseDate: req.PurchaseDate, TotalAmount: total,
	}

	// Satu transaksi: nota + semua mutasi IN. Gagal di satu baris = seluruh nota dibatalkan.
	err := s.repo.WithTx(ctx, func(tx *sql.Tx) error {
		purchaseID, err := s.repo.InsertPurchase(ctx, tx, req.InvoiceNo, req.Supplier, req.PurchaseDate, total)
		if err != nil {
			return err
		}
		result.ID = purchaseID

		for _, l := range req.Items {
			st, err := s.repo.GetItemForUpdate(ctx, tx, l.ItemID)
			if err != nil {
				return err
			}
			if st.ConversionRatio <= 0 {
				return validationErr("item id %d memiliki conversion_ratio <= 0", l.ItemID)
			}
			qtyUsage := l.QtyPurchaseUnit * st.ConversionRatio // satuan beli -> satuan pakai
			costPerUsage := l.UnitPrice / st.ConversionRatio
			newCost := CalcMovingAverage(st.StockQty, st.MovingAvgCost, qtyUsage, costPerUsage)

			newQty, err := s.repo.ApplyInbound(ctx, tx, l.ItemID, purchaseID, qtyUsage, st.StockQty, st.MovingAvgCost, newCost)
			if err != nil {
				return err
			}
			result.Lines = append(result.Lines, models.PurchaseLineResult{
				ItemID: l.ItemID, QtyUsageUnit: qtyUsage, CostPerUsage: costPerUsage,
				OldAvgCost: st.MovingAvgCost, NewAvgCost: newCost, StockAfter: newQty,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *PurchaseService) List(ctx context.Context) ([]models.Purchase, error) {
	return s.repo.List(ctx)
}
