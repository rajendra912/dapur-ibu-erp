package service

import (
	"math"
	"testing"

	"github.com/rajendra912/dapur-ibu-erp/internal/models"
)

func fixtures() (map[int]models.Recipe, map[int]models.ItemInfo) {
	items := map[int]models.ItemInfo{
		1: {ID: 1, Name: "Nasi Kotak A", ItemType: "MENU"},
		2: {ID: 2, Name: "Ayam Goreng", ItemType: "COMP"},
		3: {ID: 3, Name: "Bumbu Dasar", ItemType: "WIP"},
		4: {ID: 4, Name: "Ayam Mentah", ItemType: "RAW", StockQty: 2000, ConversionRatio: 1000, PurchaseUnit: "kg"},
		5: {ID: 5, Name: "Bawang Merah", ItemType: "RAW", StockQty: 0, ConversionRatio: 1000, PurchaseUnit: "kg"},
		6: {ID: 6, Name: "Beras", ItemType: "RAW", StockQty: 99999, ConversionRatio: 5000, PurchaseUnit: "krg"},
	}
	recipes := map[int]models.Recipe{
		1: {ID: 1, OutputItemID: 1, BatchYield: 1, Components: []models.RecipeComponent{{ComponentItemID: 2, QtyReq: 1}, {ComponentItemID: 6, QtyReq: 150}}},
		2: {ID: 2, OutputItemID: 2, BatchYield: 20, Components: []models.RecipeComponent{{ComponentItemID: 4, QtyReq: 4000}, {ComponentItemID: 3, QtyReq: 1}}},
		3: {ID: 3, OutputItemID: 3, BatchYield: 1, Components: []models.RecipeComponent{{ComponentItemID: 5, QtyReq: 500}}},
	}
	return recipes, items
}

func TestExplodeBOM_MultiLevel(t *testing.T) {
	recipes, items := fixtures()
	gross, err := ExplodeBOM(map[int]float64{1: 500}, recipes, items)
	if err != nil {
		t.Fatal(err)
	}
	// 500 porsi / 20 = 25 batch ayam -> 100.000 gr ayam, 25 x 500 = 12.500 gr bawang, 500 x 150 = 75.000 gr beras
	want := map[int]float64{4: 100000, 5: 12500, 6: 75000}
	for id, w := range want {
		if math.Abs(gross[id]-w) > 1e-6 {
			t.Errorf("item %d: dapat %v, seharusnya %v", id, gross[id], w)
		}
	}
}

func TestBuildShoppingLines_NetAndCeil(t *testing.T) {
	recipes, items := fixtures()
	gross, _ := ExplodeBOM(map[int]float64{1: 500}, recipes, items)
	lines, skipped, err := BuildShoppingLines(gross, items, false)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 1 { // beras: stok 99.999 > kebutuhan 75.000
		t.Errorf("skipped = %d, seharusnya 1", skipped)
	}
	got := map[int]int{}
	for _, l := range lines {
		got[l.RawItemID] = l.PurchaseQtyRounded
	}
	if got[4] != 98 { // (100.000 - 2.000) / 1.000 = 98 pas
		t.Errorf("ayam = %d, seharusnya 98", got[4])
	}
	if got[5] != 13 { // 12.500 / 1.000 = 12,5 -> CEIL 13
		t.Errorf("bawang = %d, seharusnya 13", got[5])
	}
}

func TestBuildShoppingLines_Buffer5Persen(t *testing.T) {
	recipes, items := fixtures()
	gross, _ := ExplodeBOM(map[int]float64{1: 500}, recipes, items)
	lines, _, _ := BuildShoppingLines(gross, items, true)
	for _, l := range lines {
		if l.RawItemID == 4 && l.PurchaseQtyRounded != 103 { // 105.000 - 2.000 = 103.000 gr
			t.Errorf("ayam + buffer = %d, seharusnya 103", l.PurchaseQtyRounded)
		}
	}
}

func TestExplodeBOM_Siklus(t *testing.T) {
	items := map[int]models.ItemInfo{1: {ID: 1, Name: "A", ItemType: "WIP"}, 2: {ID: 2, Name: "B", ItemType: "WIP"}}
	recipes := map[int]models.Recipe{
		1: {OutputItemID: 1, BatchYield: 1, Components: []models.RecipeComponent{{ComponentItemID: 2, QtyReq: 1}}},
		2: {OutputItemID: 2, BatchYield: 1, Components: []models.RecipeComponent{{ComponentItemID: 1, QtyReq: 1}}},
	}
	if _, err := ExplodeBOM(map[int]float64{1: 1}, recipes, items); err != ErrBOMCycle {
		t.Errorf("seharusnya ErrBOMCycle, dapat %v", err)
	}
}

func TestCalcMovingAverage(t *testing.T) {
	if got := CalcMovingAverage(10, 1000, 10, 2000); math.Abs(got-1500) > 1e-9 {
		t.Errorf("dapat %v, seharusnya 1500", got)
	}
	if got := CalcMovingAverage(0, 0, 5, 700); got != 700 {
		t.Errorf("stok kosong: dapat %v, seharusnya 700", got)
	}
}
