package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/rajendra912/dapur-ibu-erp/internal/models"
)

// RecipeRepo menangani akses database untuk tabel `recipes` dan `recipe_bom_items`.
type RecipeRepo struct {
	db *sql.DB
}

// NewRecipeRepo membuat instance RecipeRepo baru.
func NewRecipeRepo(db *sql.DB) *RecipeRepo {
	return &RecipeRepo{db: db}
}

// IsActiveItem memvalidasi apakah item ada di tabel `items` dan berstatus aktif.
func (r *RecipeRepo) IsActiveItem(ctx context.Context, itemID int64) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM items WHERE id = ? AND is_active = TRUE`,
		itemID,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CreateRecipe menyimpan header resep baru ke tabel `recipes`.
func (r *RecipeRepo) CreateRecipe(ctx context.Context, req models.CreateRecipeRequest) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO recipes (output_item_id, recipe_name, batch_yield, std_hpp)
		 VALUES (?, ?, ?, 0.00)`,
		req.OutputItemID, req.RecipeName, req.BatchYield,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetRecipeByID mengambil satu header resep berdasarkan ID.
func (r *RecipeRepo) GetRecipeByID(ctx context.Context, id int64) (*models.RecipeHeader, error) {
	var rec models.RecipeHeader
	err := r.db.QueryRowContext(ctx,
		`SELECT r.id, r.output_item_id, COALESCE(i.name, ''), r.recipe_name, r.batch_yield, COALESCE(r.std_hpp, 0)
		 FROM recipes r
		 LEFT JOIN items i ON i.id = r.output_item_id
		 WHERE r.id = ?`,
		id,
	).Scan(&rec.ID, &rec.OutputItemID, &rec.OutputItemName, &rec.RecipeName, &rec.BatchYield, &rec.StdHPP)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRecipeNotFound
	}
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

// GetLatestRecipeByOutputItem mengambil resep terbaru milik sebuah output item (untuk BOM bertingkat).
func (r *RecipeRepo) GetLatestRecipeByOutputItem(ctx context.Context, outputItemID int64) (*models.RecipeHeader, error) {
	var rec models.RecipeHeader
	err := r.db.QueryRowContext(ctx,
		`SELECT id, output_item_id, recipe_name, batch_yield, COALESCE(std_hpp, 0)
		 FROM recipes
		 WHERE output_item_id = ?
		 ORDER BY id DESC
		 LIMIT 1`,
		outputItemID,
	).Scan(&rec.ID, &rec.OutputItemID, &rec.RecipeName, &rec.BatchYield, &rec.StdHPP)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRecipeNotFound
	}
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

// ListRecipes mengambil seluruh daftar resep beserta nama item outputnya.
func (r *RecipeRepo) ListRecipes(ctx context.Context) ([]models.RecipeHeader, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT r.id, r.output_item_id, COALESCE(i.name, ''), r.recipe_name, r.batch_yield, COALESCE(r.std_hpp, 0)
		 FROM recipes r
		 LEFT JOIN items i ON i.id = r.output_item_id
		 ORDER BY r.id DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []models.RecipeHeader{}
	for rows.Next() {
		var rec models.RecipeHeader
		if err := rows.Scan(&rec.ID, &rec.OutputItemID, &rec.OutputItemName, &rec.RecipeName, &rec.BatchYield, &rec.StdHPP); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

// ListBOMItems mengambil seluruh komponen penyusun satu resep beserta info item dan harga rata-ratanya.
func (r *RecipeRepo) ListBOMItems(ctx context.Context, recipeID int64) ([]models.RecipeBOMItem, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT b.id, b.recipe_id, b.component_item_id,
		        COALESCE(i.name, ''), COALESCE(i.item_type, ''), COALESCE(u.code, ''),
		        b.qty_req, COALESCE(i.moving_avg_cost, 0)
		 FROM recipe_bom_items b
		 LEFT JOIN items i ON i.id = b.component_item_id
		 LEFT JOIN units u ON u.id = i.usage_unit_id
		 WHERE b.recipe_id = ?
		 ORDER BY b.id ASC`,
		recipeID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []models.RecipeBOMItem{}
	for rows.Next() {
		var item models.RecipeBOMItem
		if err := rows.Scan(
			&item.ID, &item.RecipeID, &item.ComponentItemID,
			&item.ComponentName, &item.ComponentType, &item.UsageUnit,
			&item.QtyReq, &item.MovingAvgCost,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// AddBOMItems menambahkan satu atau lebih komponen ke tabel `recipe_bom_items` dalam satu transaksi.
func (r *RecipeRepo) AddBOMItems(ctx context.Context, recipeID int64, items []models.RecipeComponentInput) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO recipe_bom_items (recipe_id, component_item_id, qty_req) VALUES (?, ?, ?)`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, it := range items {
		if _, err := stmt.ExecContext(ctx, recipeID, it.ComponentItemID, it.QtyReq); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// UpdateRecipeStdHPP memperbarui nilai `std_hpp` pada tabel `recipes`.
func (r *RecipeRepo) UpdateRecipeStdHPP(ctx context.Context, recipeID int64, stdHPP float64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE recipes SET std_hpp = ? WHERE id = ?`,
		stdHPP, recipeID,
	)
	return err
}
