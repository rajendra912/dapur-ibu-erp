package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/rajendra912/dapur-ibu-erp/internal/models"
	"github.com/rajendra912/dapur-ibu-erp/internal/repository"
	"github.com/rajendra912/dapur-ibu-erp/internal/service"
)

// Guard adalah pembungkus RBAC milik Rajen. Bila nil, route dibuka tanpa pengecekan role (mode dev).
type Guard func(roles ...string) func(http.Handler) http.Handler

type MRPHandler struct {
	mrp       *service.MRPService
	purchases *service.PurchaseService
}

func NewMRPHandler(mrp *service.MRPService, purchases *service.PurchaseService) *MRPHandler {
	return &MRPHandler{mrp: mrp, purchases: purchases}
}

// RegisterRoutes mendaftarkan route modul MRP & Pembelian ke router gorilla/mux proyek.
func (h *MRPHandler) RegisterRoutes(r *mux.Router, guard Guard) {
	wrap := func(f http.HandlerFunc, roles ...string) http.Handler {
		if guard == nil {
			return f
		}
		return guard(roles...)(f)
	}
	r.Handle("/api/mrp/explode", wrap(h.Explode, "OWNER", "WAREHOUSE")).Methods("POST")
	r.Handle("/api/mrp/production-runs", wrap(h.ListRuns, "OWNER", "WAREHOUSE", "KITCHEN")).Methods("GET")
	r.Handle("/api/mrp/shopping-lists", wrap(h.ShoppingList, "OWNER", "WAREHOUSE")).Methods("GET")
	r.Handle("/api/purchases", wrap(h.CreatePurchase, "OWNER", "WAREHOUSE")).Methods("POST")
	r.Handle("/api/purchases", wrap(h.ListPurchases, "OWNER", "WAREHOUSE")).Methods("GET")
}

func (h *MRPHandler) Explode(w http.ResponseWriter, r *http.Request) {
	var req service.ExplodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "body JSON tidak valid")
		return
	}
	res, err := h.mrp.Explode(r.Context(), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *MRPHandler) ListRuns(w http.ResponseWriter, r *http.Request) {
	runs, err := h.mrp.ListRuns(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, runs)
}

// GET /api/mrp/shopping-lists?production_run_id=1
func (h *MRPHandler) ShoppingList(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.URL.Query().Get("production_run_id"))
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "query production_run_id wajib berupa angka")
		return
	}
	list, err := h.mrp.ShoppingList(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *MRPHandler) CreatePurchase(w http.ResponseWriter, r *http.Request) {
	var req models.PurchaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "body JSON tidak valid")
		return
	}
	p, err := h.purchases.Create(r.Context(), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (h *MRPHandler) ListPurchases(w http.ResponseWriter, r *http.Request) {
	list, err := h.purchases.List(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// ---------- helper ----------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrNoLockedOrders):
		writeError(w, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, repository.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, repository.ErrRunClosed):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrBOMCycle):
		writeError(w, http.StatusUnprocessableEntity, err.Error())
	default:
		// error resep/item (mis. "belum memiliki resep") juga sampai di sini; pesan tetap dikirim agar mudah debug saat dev
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}
