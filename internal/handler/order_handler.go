package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rajendra912/dapur-ibu-erp/internal/middleware"
	"github.com/rajendra912/dapur-ibu-erp/internal/models"
	"github.com/rajendra912/dapur-ibu-erp/internal/repository"
	"github.com/rajendra912/dapur-ibu-erp/internal/service"

	"github.com/gorilla/mux"
	"github.com/rajendra912/dapur-ibu-erp/config"
)

// Prefix "ord" dipakai supaya helper JSON tidak bentrok dengan file handler
// milik anggota tim lain (mis. whJSON milik modul gudang).
func ordJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func ordErr(w http.ResponseWriter, status int, msg string) {
	ordJSON(w, status, map[string]string{"error": msg})
}

// ordServiceErr menerjemahkan error dari service menjadi status HTTP yang tepat.
func ordServiceErr(w http.ResponseWriter, err error) {
	var ve service.ValidationError
	switch {
	case errors.As(err, &ve):
		ordErr(w, http.StatusBadRequest, ve.Message)
	case errors.Is(err, repository.ErrOrderNotFound):
		ordErr(w, http.StatusNotFound, "pesanan tidak ditemukan")
	case errors.Is(err, service.ErrOrderLocked), errors.Is(err, service.ErrOrderDone):
		// Sesuai jobdesc: pesanan yang sudah terkunci = 400 Bad Request.
		ordErr(w, http.StatusBadRequest, err.Error())
	default:
		log.Printf("[order] error internal: %v", err)
		ordErr(w, http.StatusInternalServerError, "terjadi kesalahan pada server")
	}
}

func ordParseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
}

// CreateOrder menangani POST /api/orders.
func CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req models.OrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ordErr(w, http.StatusBadRequest, "format JSON tidak valid")
		return
	}

	order, err := service.CreateOrder(config.DB, req)
	if err != nil {
		ordServiceErr(w, err)
		return
	}
	ordJSON(w, http.StatusCreated, order)
}

// ListOrders menangani GET /api/orders?status=&event_date=&q=&limit=.
func ListOrders(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := models.OrderFilter{
		Status:    q.Get("status"),
		EventDate: q.Get("event_date"),
		Search:    strings.TrimSpace(q.Get("q")),
	}
	if filter.EventDate == "" {
		filter.EventDate = q.Get("date")
	}
	if l := q.Get("limit"); l != "" {
		n, err := strconv.Atoi(l)
		if err != nil || n <= 0 {
			ordErr(w, http.StatusBadRequest, "limit harus berupa angka positif")
			return
		}
		filter.Limit = n
	}

	list, err := service.ListOrders(config.DB, filter)
	if err != nil {
		ordServiceErr(w, err)
		return
	}
	ordJSON(w, http.StatusOK, map[string]interface{}{
		"total": len(list),
		"data":  list,
	})
}

// GetOrder menangani GET /api/orders/{id} (header + rincian menu).
func GetOrder(w http.ResponseWriter, r *http.Request) {
	id, err := ordParseID(r)
	if err != nil {
		ordErr(w, http.StatusBadRequest, "id pesanan tidak valid")
		return
	}
	order, err := service.GetOrderDetail(config.DB, id)
	if err != nil {
		ordServiceErr(w, err)
		return
	}
	ordJSON(w, http.StatusOK, order)
}

// UpdateOrder menangani PUT /api/orders/{id}.
// Middleware GuardOrderEditable sudah menolak lebih dulu jika pesanan terkunci.
func UpdateOrder(w http.ResponseWriter, r *http.Request) {
	id, err := ordParseID(r)
	if err != nil {
		ordErr(w, http.StatusBadRequest, "id pesanan tidak valid")
		return
	}

	var req models.OrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ordErr(w, http.StatusBadRequest, "format JSON tidak valid")
		return
	}

	order, err := service.UpdateOrder(config.DB, id, req)
	if err != nil {
		ordServiceErr(w, err)
		return
	}
	ordJSON(w, http.StatusOK, order)
}

// DeleteOrder menangani DELETE /api/orders/{id}.
func DeleteOrder(w http.ResponseWriter, r *http.Request) {
	id, err := ordParseID(r)
	if err != nil {
		ordErr(w, http.StatusBadRequest, "id pesanan tidak valid")
		return
	}
	if err := service.DeleteOrder(config.DB, id); err != nil {
		ordServiceErr(w, err)
		return
	}
	ordJSON(w, http.StatusOK, map[string]string{"message": "pesanan berhasil dihapus"})
}

// GetOrderSummary menangani GET /api/orders/summary?event_date=YYYY-MM-DD&status=
// Menampilkan rekap pesanan + total porsi per menu pada satu tanggal acara
// (dipakai dashboard dan menjadi bahan agregasi Level 0 modul MRP).
func GetOrderSummary(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	date := q.Get("event_date")
	if date == "" {
		date = q.Get("date")
	}
	if date == "" {
		ordErr(w, http.StatusBadRequest, "parameter event_date wajib diisi (YYYY-MM-DD)")
		return
	}

	summary, err := service.GetOrderSummary(config.DB, date, q.Get("status"))
	if err != nil {
		ordServiceErr(w, err)
		return
	}
	ordJSON(w, http.StatusOK, summary)
}

// RunCutoff menangani POST /api/cron/cutoff.
// Endpoint manual ini penting untuk demo ke dosen: penguncian bisa dipicu
// seketika tanpa menunggu worker jam 00:00.
func RunCutoff(w http.ResponseWriter, r *http.Request) {
	res, err := service.LockDueOrders(config.DB, time.Now())
	if err != nil {
		ordServiceErr(w, err)
		return
	}
	ordJSON(w, http.StatusOK, map[string]interface{}{
		"message": "cutoff selesai dijalankan",
		"result":  res,
	})
}

// RegisterOrderRoutes mendaftarkan seluruh endpoint modul pesanan.
// Rajen (integrator) cukup memanggil satu fungsi ini dari cmd/server/main.go.
func RegisterOrderRoutes(r *mux.Router) {
	guard := middleware.GuardOrderEditable(config.DB)

	r.HandleFunc("/api/orders", ListOrders).Methods("GET")
	r.HandleFunc("/api/orders", CreateOrder).Methods("POST")
	r.HandleFunc("/api/orders/summary", GetOrderSummary).Methods("GET")
	r.HandleFunc("/api/orders/{id}", GetOrder).Methods("GET")
	r.Handle("/api/orders/{id}", guard(http.HandlerFunc(UpdateOrder))).Methods("PUT", "PATCH")
	r.Handle("/api/orders/{id}", guard(http.HandlerFunc(DeleteOrder))).Methods("DELETE")

	// Worker cutoff (manual trigger).
	r.HandleFunc("/api/cron/cutoff", RunCutoff).Methods("POST")
}
