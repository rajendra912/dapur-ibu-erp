package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/rajendra912/dapur-ibu-erp/internal/models"
	"github.com/rajendra912/dapur-ibu-erp/internal/repository"
)

var (
	ErrValidation     = errors.New("input tidak valid")
	ErrNoLockedOrders = errors.New("tidak ada pesanan berstatus LOCKED pada tanggal produksi tersebut")
	ErrBOMCycle       = errors.New("resep bertingkat membentuk siklus (loop)")
)

const (
	maxBOMDepth   = 10   // MENU -> COMP -> WIP -> RAW = 4 tingkat; 10 sebagai pengaman siklus
	bufferPercent = 5.0  // buffer produksi opsional (toggle) sesuai spesifikasi final
	epsilon       = 1e-9 // toleransi error floating point sebelum CEIL
)

func validationErr(format string, a ...any) error {
	return fmt.Errorf("%w: %s", ErrValidation, fmt.Sprintf(format, a...))
}

// ---------- Fungsi murni (tanpa DB, mudah dites) ----------

// ExplodeBOM memecah demand menu (item_id -> porsi) menjadi kebutuhan bahan mentah (RAW) kotor.
// recipes dikunci berdasarkan output_item_id. Satuan hasil = satuan pakai (gr/ml).
func ExplodeBOM(demand map[int]float64, recipes map[int]models.Recipe, items map[int]models.ItemInfo) (map[int]float64, error) {
	gross := map[int]float64{}

	var walk func(itemID int, qty float64, depth int) error
	walk = func(itemID int, qty float64, depth int) error {
		if depth > maxBOMDepth {
			return ErrBOMCycle
		}
		it, ok := items[itemID]
		if !ok {
			return fmt.Errorf("item id %d tidak ditemukan atau sudah nonaktif", itemID)
		}
		if it.ItemType == "RAW" {
			gross[itemID] += qty
			return nil
		}
		rc, ok := recipes[itemID]
		if !ok {
			return fmt.Errorf("item %q (%s) belum memiliki resep", it.Name, it.ItemType)
		}
		if rc.BatchYield <= 0 {
			return fmt.Errorf("resep %q memiliki batch_yield <= 0", it.Name)
		}
		batches := qty / rc.BatchYield // pengali resep, mis. 500 porsi / 20 = 25 batch
		for _, c := range rc.Components {
			if err := walk(c.ComponentItemID, batches*c.QtyReq, depth+1); err != nil {
				return err
			}
		}
		return nil
	}

	for menuID, portions := range demand {
		if err := walk(menuID, portions, 0); err != nil {
			return nil, err
		}
	}
	return gross, nil
}

// BuildShoppingLines: gross -> net (dikurangi stok) -> CEIL ke satuan beli. Bahan yang stoknya cukup dilewati.
func BuildShoppingLines(gross map[int]float64, items map[int]models.ItemInfo, applyBuffer bool) (lines []models.ShoppingListItem, skipped int, err error) {
	ids := make([]int, 0, len(gross))
	for id := range gross {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	for _, id := range ids {
		it := items[id]
		g := gross[id]
		if applyBuffer {
			g *= 1 + bufferPercent/100
		}
		net := math.Max(g-it.StockQty, 0)
		if net == 0 {
			skipped++
			continue
		}
		if it.ConversionRatio <= 0 {
			return nil, 0, fmt.Errorf("bahan %q memiliki conversion_ratio <= 0", it.Name)
		}
		lines = append(lines, models.ShoppingListItem{
			RawItemID:          id,
			RawItemName:        it.Name,
			GrossReq:           round2(g),
			StockOnHand:        round2(it.StockQty),
			NetReq:             round2(net),
			PurchaseQtyRounded: int(math.Ceil(net/it.ConversionRatio - epsilon)),
			PurchaseUnit:       it.PurchaseUnit,
		})
	}
	return lines, skipped, nil
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

// ---------- Service ----------

type ExplodeRequest struct {
	ProductionDate string `json:"production_date"` // YYYY-MM-DD
	ApplyBuffer    bool   `json:"apply_buffer"`    // true = tambah buffer 5%
}

type ExplodeResult struct {
	ProductionRunID   int                       `json:"production_run_id"`
	ProductionDate    string                    `json:"production_date"`
	LockedOrderCount  int                       `json:"locked_order_count"`
	MenuDemand        map[int]float64           `json:"menu_demand"`
	BufferApplied     bool                      `json:"buffer_applied"`
	SkippedStockCukup int                       `json:"skipped_stock_cukup"`
	ShoppingList      []models.ShoppingListItem `json:"shopping_list"`
}

type MRPService struct{ repo *repository.MRPRepo }

func NewMRPService(repo *repository.MRPRepo) *MRPService { return &MRPService{repo: repo} }

func (s *MRPService) Explode(ctx context.Context, req ExplodeRequest) (*ExplodeResult, error) {
	if _, err := time.Parse("2006-01-02", req.ProductionDate); err != nil {
		return nil, validationErr("production_date harus berformat YYYY-MM-DD")
	}

	demand, orderCount, err := s.repo.LockedDemand(ctx, req.ProductionDate)
	if err != nil {
		return nil, err
	}
	if len(demand) == 0 {
		return nil, ErrNoLockedOrders
	}
	recipes, err := s.repo.LoadRecipes(ctx)
	if err != nil {
		return nil, err
	}
	items, err := s.repo.LoadItems(ctx)
	if err != nil {
		return nil, err
	}

	gross, err := ExplodeBOM(demand, recipes, items)
	if err != nil {
		return nil, err
	}
	lines, skipped, err := BuildShoppingLines(gross, items, req.ApplyBuffer)
	if err != nil {
		return nil, err
	}
	runID, err := s.repo.SaveShoppingList(ctx, req.ProductionDate, lines)
	if err != nil {
		return nil, err
	}
	for i := range lines {
		lines[i].ProductionRunID = runID
	}
	return &ExplodeResult{
		ProductionRunID: runID, ProductionDate: req.ProductionDate, LockedOrderCount: orderCount,
		MenuDemand: demand, BufferApplied: req.ApplyBuffer, SkippedStockCukup: skipped, ShoppingList: lines,
	}, nil
}

func (s *MRPService) ListRuns(ctx context.Context) ([]models.ProductionRun, error) {
	return s.repo.ListRuns(ctx)
}

func (s *MRPService) ShoppingList(ctx context.Context, runID int) ([]models.ShoppingListItem, error) {
	return s.repo.GetShoppingList(ctx, runID)
}
