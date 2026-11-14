package models

// RecipeHeader merepresentasikan satu baris tabel `recipes` (header resep).
type RecipeHeader struct {
	ID             int64           `json:"id"`
	OutputItemID   int64           `json:"output_item_id"`
	OutputItemName string          `json:"output_item_name,omitempty"`
	RecipeName     string          `json:"recipe_name"`
	BatchYield     float64         `json:"batch_yield"`
	StdHPP         float64         `json:"std_hpp"`
	Components     []RecipeBOMItem `json:"components,omitempty"`
}

// RecipeBOMItem merepresentasikan satu baris tabel `recipe_bom_items` (detail komponen resep).
type RecipeBOMItem struct {
	ID              int64   `json:"id"`
	RecipeID        int64   `json:"recipe_id"`
	ComponentItemID int64   `json:"component_item_id"`
	ComponentName   string  `json:"component_name,omitempty"`
	ComponentType   string  `json:"component_type,omitempty"`
	UsageUnit       string  `json:"usage_unit,omitempty"`
	QtyReq          float64 `json:"qty_req"`
	MovingAvgCost   float64 `json:"moving_avg_cost,omitempty"`
}

// CreateRecipeRequest adalah payload untuk POST /api/recipes.
type CreateRecipeRequest struct {
	OutputItemID int64   `json:"output_item_id"`
	RecipeName   string  `json:"recipe_name"`
	BatchYield   float64 `json:"batch_yield"`
}

// RecipeComponentInput adalah payload satu komponen penyusun resep.
type RecipeComponentInput struct {
	ComponentItemID int64   `json:"component_item_id"`
	QtyReq          float64 `json:"qty_req"`
}

// AddRecipeComponentsRequest adalah payload untuk POST /api/recipes/{id}/components.
// Mendukung input satu komponen langsung maupun array `components`.
type AddRecipeComponentsRequest struct {
	ComponentItemID int64                  `json:"component_item_id"`
	QtyReq          float64                `json:"qty_req"`
	Components      []RecipeComponentInput `json:"components"`
}

// RecipeHPPComponent adalah rincian biaya satu komponen dalam perhitungan HPP resep.
type RecipeHPPComponent struct {
	ComponentItemID int64   `json:"component_item_id"`
	ComponentName   string  `json:"component_name"`
	ComponentType   string  `json:"component_type"`
	UsageUnit       string  `json:"usage_unit,omitempty"`
	QtyReq          float64 `json:"qty_req"`
	UnitCost        float64 `json:"unit_cost"`
	SubtotalCost    float64 `json:"subtotal_cost"`
}

// RecipeHPPResponse adalah hasil kalkulasi GET /api/recipes/{id}/hpp.
type RecipeHPPResponse struct {
	RecipeID       int64                `json:"recipe_id"`
	OutputItemID   int64                `json:"output_item_id"`
	RecipeName     string               `json:"recipe_name"`
	BatchYield     float64              `json:"batch_yield"`
	TotalBatchCost float64              `json:"total_batch_cost"`
	StdHPP         float64              `json:"std_hpp"`
	Components     []RecipeHPPComponent `json:"components"`
}
