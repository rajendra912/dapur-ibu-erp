package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/rajendra912/dapur-ibu-erp/internal/models"
	"github.com/rajendra912/dapur-ibu-erp/internal/repository"
	"github.com/rajendra912/dapur-ibu-erp/internal/service"
)

// RecipeHandler menangani endpoint modul Recipe & Kitchen Engine (Syarif).
type RecipeHandler struct {
	svc *service.RecipeService
}

// NewRecipeHandler membuat instance RecipeHandler baru.
func NewRecipeHandler(svc *service.RecipeService) *RecipeHandler {
	return &RecipeHandler{svc: svc}
}

// RegisterRoutes mendaftarkan seluruh endpoint resep ke router gorilla/mux.
func (h *RecipeHandler) RegisterRoutes(r *mux.Router, guard Guard) {
	wrap := func(f http.HandlerFunc, roles ...string) http.Handler {
		if guard == nil {
			return f
		}
		return guard(roles...)(f)
	}

	r.Handle("/api/recipes", wrap(h.ListRecipes, "OWNER", "KITCHEN", "ADMIN")).Methods("GET")
	r.Handle("/api/recipes", wrap(h.CreateRecipe, "OWNER", "KITCHEN")).Methods("POST")
	r.Handle("/api/recipes/{id}", wrap(h.GetRecipe, "OWNER", "KITCHEN", "ADMIN")).Methods("GET")
	r.Handle("/api/recipes/{id}/components", wrap(h.AddComponents, "OWNER", "KITCHEN")).Methods("POST")
	r.Handle("/api/recipes/{id}/hpp", wrap(h.GetHPP, "OWNER", "KITCHEN", "ADMIN")).Methods("GET")
}

// Prefix "rcp" dipakai agar tidak bentrok dengan helper milik anggota tim lain di package handler.
func rcpJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func rcpErr(w http.ResponseWriter, status int, msg string) {
	rcpJSON(w, status, map[string]string{"error": msg})
}

func rcpServiceErr(w http.ResponseWriter, err error) {
	var ve service.ValidationError
	switch {
	case errors.As(err, &ve):
		rcpErr(w, http.StatusBadRequest, ve.Message)
	case errors.Is(err, repository.ErrRecipeNotFound):
		rcpErr(w, http.StatusNotFound, "resep tidak ditemukan")
	default:
		log.Printf("[recipe] error internal: %v", err)
		rcpErr(w, http.StatusInternalServerError, "terjadi kesalahan pada server")
	}
}

func rcpParseID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("id resep tidak valid")
	}
	return id, nil
}

// CreateRecipe menangani POST /api/recipes (Header Resep).
func (h *RecipeHandler) CreateRecipe(w http.ResponseWriter, r *http.Request) {
	var req models.CreateRecipeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rcpErr(w, http.StatusBadRequest, "format JSON tidak valid")
		return
	}

	rec, err := h.svc.CreateRecipe(r.Context(), req)
	if err != nil {
		rcpServiceErr(w, err)
		return
	}
	rcpJSON(w, http.StatusCreated, rec)
}

// ListRecipes menangani GET /api/recipes.
func (h *RecipeHandler) ListRecipes(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListRecipes(r.Context())
	if err != nil {
		rcpServiceErr(w, err)
		return
	}
	rcpJSON(w, http.StatusOK, list)
}

// GetRecipe menangani GET /api/recipes/{id}.
func (h *RecipeHandler) GetRecipe(w http.ResponseWriter, r *http.Request) {
	id, err := rcpParseID(r)
	if err != nil {
		rcpErr(w, http.StatusBadRequest, err.Error())
		return
	}

	rec, err := h.svc.GetRecipe(r.Context(), id)
	if err != nil {
		rcpServiceErr(w, err)
		return
	}
	rcpJSON(w, http.StatusOK, rec)
}

// AddComponents menangani POST /api/recipes/{id}/components (Detail Komponen BOM).
func (h *RecipeHandler) AddComponents(w http.ResponseWriter, r *http.Request) {
	id, err := rcpParseID(r)
	if err != nil {
		rcpErr(w, http.StatusBadRequest, err.Error())
		return
	}

	var req models.AddRecipeComponentsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rcpErr(w, http.StatusBadRequest, "format JSON tidak valid")
		return
	}

	components, err := h.svc.AddComponents(r.Context(), id, req)
	if err != nil {
		rcpServiceErr(w, err)
		return
	}
	rcpJSON(w, http.StatusCreated, map[string]interface{}{
		"message":    "komponen resep berhasil ditambahkan",
		"recipe_id":  id,
		"components": components,
	})
}

// GetHPP menangani GET /api/recipes/{id}/hpp (Kalkulasi HPP Standar per Porsi).
func (h *RecipeHandler) GetHPP(w http.ResponseWriter, r *http.Request) {
	id, err := rcpParseID(r)
	if err != nil {
		rcpErr(w, http.StatusBadRequest, err.Error())
		return
	}

	res, err := h.svc.CalculateHPP(r.Context(), id)
	if err != nil {
		rcpServiceErr(w, err)
		return
	}
	rcpJSON(w, http.StatusOK, res)
}
