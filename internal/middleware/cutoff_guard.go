package middleware

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"[github.com/rajendra912/dapur-ibu-erp/](https://github.com/rajendra912/dapur-ibu-erp/)internal/models"
	"[github.com/rajendra912/dapur-ibu-erp/](https://github.com/rajendra912/dapur-ibu-erp/)internal/repository"
	"[github.com/rajendra912/dapur-ibu-erp/](https://github.com/rajendra912/dapur-ibu-erp/)internal/service"

	"github.com/gorilla/mux"
)

// cutoffErr mengirim respons error JSON. Prefix "cutoff" dipakai agar tidak
// bentrok dengan helper milik anggota tim lain di package middleware.
func cutoffErr(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// GuardOrderEditable adalah middleware pelindung pesanan (Cutoff Guard).
// Dipasang pada rute PUT/DELETE /api/orders/{id}:
//   - 404 jika pesanan tidak ada,
//   - 400 jika status sudah LOCKED atau DONE,
//   - 400 jika waktu sekarang sudah melewati cutoff_date (H-2 23:59:59).
//
// Service tetap melakukan pengecekan ulang di dalam transaksi (double guard).
func GuardOrderEditable(db *sql.DB) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
			if err != nil {
				cutoffErr(w, http.StatusBadRequest, "id pesanan tidak valid")
				return
			}

			order, err := repository.GetOrderByID(db, id)
			if errors.Is(err, repository.ErrOrderNotFound) {
				cutoffErr(w, http.StatusNotFound, "pesanan tidak ditemukan")
				return
			}
			if err != nil {
				cutoffErr(w, http.StatusInternalServerError, "gagal memeriksa status pesanan")
				return
			}

			if order.Status == models.OrderStatusLocked {
				cutoffErr(w, http.StatusBadRequest, service.ErrOrderLocked.Error())
				return
			}
			if order.Status == models.OrderStatusDone {
				cutoffErr(w, http.StatusBadRequest, service.ErrOrderDone.Error())
				return
			}
			if !time.Now().Before(order.CutoffDate) {
				cutoffErr(w, http.StatusBadRequest, service.ErrOrderLocked.Error())
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
