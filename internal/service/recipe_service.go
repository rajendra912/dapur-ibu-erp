package service

import (
	"context"
	"errors"
	"math"
	"strings"

	"github.com/rajendra912/dapur-ibu-erp/internal/models"
	"github.com/rajendra912/dapur-ibu-erp/internal/repository"
)

const maxRecipeBOMDepth = 10

// RecipeService menangani logika bisnis pembuatan resep bertingkat (BOM) dan kalkulasi HPP standar.
type RecipeService struct {
	repo *repository.RecipeRepo
}

// NewRecipeService membuat instance RecipeService baru.
func NewRecipeService(repo *repository.RecipeRepo) *RecipeService {
	return &RecipeService{repo: repo}
}

// CreateRecipe memvalidasi dan menyimpan header resep baru.
func (s *RecipeService) CreateRecipe(ctx context.Context, req models.CreateRecipeRequest) (*models.RecipeHeader, error) {
	req.RecipeName = strings.TrimSpace(req.RecipeName)
	if req.OutputItemID <= 0 {
		return nil, ValidationError{Message: "output_item_id wajib diisi"}
	}
	if req.RecipeName == "" {
		return nil, ValidationError{Message: "recipe_name wajib diisi"}
	}
	if req.BatchYield <= 0 {
		return nil, ValidationError{Message: "batch_yield harus lebih dari 0"}
	}

	active, err := s.repo.IsActiveItem(ctx, req.OutputItemID)
	if err != nil {
		return nil, err
	}
	if !active {
		return nil, ValidationError{Message: "output_item_id tidak ditemukan atau tidak aktif"}
	}

	id, err := s.repo.CreateRecipe(ctx, req)
	if err != nil {
		return nil, err
	}
	return s.repo.GetRecipeByID(ctx, id)
}

// ListRecipes mengambil daftar seluruh resep.
func (s *RecipeService) ListRecipes(ctx context.Context) ([]models.RecipeHeader, error) {
	return s.repo.ListRecipes(ctx)
}

// GetRecipe mengambil detail resep beserta seluruh komponen penyusunnya.
func (s *RecipeService) GetRecipe(ctx context.Context, recipeID int64) (*models.RecipeHeader, error) {
	rec, err := s.repo.GetRecipeByID(ctx, recipeID)
	if err != nil {
		return nil, err
	}
	components, err := s.repo.ListBOMItems(ctx, recipeID)
	if err != nil {
		return nil, err
	}
	rec.Components = components
	return rec, nil
}

// AddComponents menambahkan satu atau lebih komponen ke resep tertentu dan memperbarui std_hpp.
func (s *RecipeService) AddComponents(ctx context.Context, recipeID int64, req models.AddRecipeComponentsRequest) ([]models.RecipeBOMItem, error) {
	rec, err := s.repo.GetRecipeByID(ctx, recipeID)
	if err != nil {
		return nil, err
	}

	items := req.Components
	if len(items) == 0 {
		if req.ComponentItemID == 0 && req.QtyReq == 0 {
			return nil, ValidationError{Message: "komponen resep wajib diisi"}
		}
		items = []models.RecipeComponentInput{
			{
				ComponentItemID: req.ComponentItemID,
				QtyReq:          req.QtyReq,
			},
		}
	}

	for _, it := range items {
		if it.ComponentItemID <= 0 {
			return nil, ValidationError{Message: "component_item_id wajib diisi"}
		}
		if it.QtyReq <= 0 {
			return nil, ValidationError{Message: "qty_req harus lebih dari 0"}
		}
		if it.ComponentItemID == rec.OutputItemID {
			return nil, ValidationError{Message: "component_item_id tidak boleh sama dengan output_item_id"}
		}
		active, err := s.repo.IsActiveItem(ctx, it.ComponentItemID)
		if err != nil {
			return nil, err
		}
		if !active {
			return nil, ValidationError{Message: "component_item_id tidak ditemukan atau tidak aktif"}
		}
	}

	if err := s.repo.AddBOMItems(ctx, recipeID, items); err != nil {
		return nil, err
	}

	// Sinkronkan std_hpp setelah komponen bertambah agar data di tabel recipes tetap akurat.
	if _, err := s.CalculateHPP(ctx, recipeID); err != nil {
		return nil, err
	}

	return s.repo.ListBOMItems(ctx, recipeID)
}

// CalculateHPP menghitung Harga Pokok Standar per porsi (std_hpp) dari seluruh komponen resep
// dan menyimpannya kembali ke kolom `std_hpp` pada tabel `recipes`.
func (s *RecipeService) CalculateHPP(ctx context.Context, recipeID int64) (*models.RecipeHPPResponse, error) {
	rec, err := s.repo.GetRecipeByID(ctx, recipeID)
	if err != nil {
		return nil, err
	}
	if rec.BatchYield <= 0 {
		return nil, ValidationError{Message: "batch_yield resep tidak valid"}
	}

	bomItems, err := s.repo.ListBOMItems(ctx, recipeID)
	if err != nil {
		return nil, err
	}

	visited := map[int64]bool{rec.OutputItemID: true}
	hppComponents := make([]models.RecipeHPPComponent, 0, len(bomItems))
	totalBatchCost := 0.0

	for _, item := range bomItems {
		unitCost := item.MovingAvgCost
		if unitCost <= 0 {
			resolvedCost, err := s.resolveSubRecipeCost(ctx, item.ComponentItemID, visited, 1)
			if err != nil {
				return nil, err
			}
			unitCost = resolvedCost
		}

		subtotal := item.QtyReq * unitCost
		totalBatchCost += subtotal

		hppComponents = append(hppComponents, models.RecipeHPPComponent{
			ComponentItemID: item.ComponentItemID,
			ComponentName:   item.ComponentName,
			ComponentType:   item.ComponentType,
			UsageUnit:       item.UsageUnit,
			QtyReq:          item.QtyReq,
			UnitCost:        roundCurrency(unitCost),
			SubtotalCost:    roundCurrency(subtotal),
		})
	}

	stdHPP := roundCurrency(totalBatchCost / rec.BatchYield)
	if err := s.repo.UpdateRecipeStdHPP(ctx, recipeID, stdHPP); err != nil {
		return nil, err
	}

	return &models.RecipeHPPResponse{
		RecipeID:       rec.ID,
		OutputItemID:   rec.OutputItemID,
		RecipeName:     rec.RecipeName,
		BatchYield:     rec.BatchYield,
		TotalBatchCost: roundCurrency(totalBatchCost),
		StdHPP:         stdHPP,
		Components:     hppComponents,
	}, nil
}

// resolveSubRecipeCost menghitung HPP komponen bertingkat (WIP/COMP) secara rekursif jika moving_avg_cost masih 0.
func (s *RecipeService) resolveSubRecipeCost(ctx context.Context, itemID int64, visited map[int64]bool, depth int) (float64, error) {
	if visited[itemID] || depth > maxRecipeBOMDepth {
		return 0, nil
	}

	subRec, err := s.repo.GetLatestRecipeByOutputItem(ctx, itemID)
	if errors.Is(err, repository.ErrRecipeNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if subRec.BatchYield <= 0 {
		return 0, nil
	}

	visited[itemID] = true
	defer delete(visited, itemID)

	bomItems, err := s.repo.ListBOMItems(ctx, subRec.ID)
	if err != nil {
		return 0, err
	}
	if len(bomItems) == 0 {
		return subRec.StdHPP, nil
	}

	total := 0.0
	for _, c := range bomItems {
		unitCost := c.MovingAvgCost
		if unitCost <= 0 {
			unitCost, err = s.resolveSubRecipeCost(ctx, c.ComponentItemID, visited, depth+1)
			if err != nil {
				return 0, err
			}
		}
		total += c.QtyReq * unitCost
	}
	return total / subRec.BatchYield, nil
}

func roundCurrency(v float64) float64 {
	return math.Round(v*100) / 100
}
