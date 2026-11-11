package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"dapur-ibu-erp/config"
	"dapur-ibu-erp/internal/models"
	"dapur-ibu-erp/internal/repository"
	"dapur-ibu-erp/internal/service"

	"github.com/gorilla/mux"
)

// Prefix "wh" supaya tidak bentrok dengan helper milik teman di package handler.
func whJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func whErr(w http.ResponseWriter, status int, msg string) {
	whJSON(w, status, map[string]string{"error": msg})
}

func RecordMovement(w http.ResponseWriter, r *http.Request) {
	var m models.Movement
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		whErr(w, http.StatusBadRequest, "format JSON tidak valid")
		return
	}

	tx, err := config.DB.Begin()
	if err != nil {
		whErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback() // aman dipanggil setelah Commit

	balance, err := service.RecordMovement(tx, m)
	if err != nil {
		switch err {
		case service.ErrStokKurang, service.ErrQtyInvalid, service.ErrJenisMutasi, service.ErrItemNotFound:
			whErr(w, http.StatusBadRequest, err.Error())
		default:
			whErr(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	if err := tx.Commit(); err != nil {
		whErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	whJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "Mutasi stok tercatat", "item_id": m.ItemID, "balance_after": balance,
	})
}

func validateItem(it models.Item) string {
	switch {
	case it.Name == "":
		return "nama bahan wajib diisi"
	case it.ItemType != "RAW" && it.ItemType != "WIP" && it.ItemType != "COMP" && it.ItemType != "MENU":
		return "item_type harus RAW, WIP, COMP, atau MENU"
	case it.ItemType == "RAW" && (it.UsageUnitID == 0 || it.PurchaseUnitID == 0):
		return "bahan RAW wajib punya satuan pakai dan satuan beli"
	case it.ItemType == "RAW" && !it.ConversionRatio.IsPositive():
		return "conversion_ratio harus lebih dari 0"
	}
	return ""
}

func GetUnits(w http.ResponseWriter, r *http.Request) {
	list, err := repository.ListUnits(config.DB)
	if err != nil {
		whErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	whJSON(w, http.StatusOK, list)
}

func CreateUnit(w http.ResponseWriter, r *http.Request) {
	var u models.Unit
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil || u.Code == "" || u.Name == "" {
		whErr(w, http.StatusBadRequest, "code dan name wajib diisi")
		return
	}
	id, err := repository.CreateUnit(config.DB, u)
	if err != nil {
		whErr(w, http.StatusBadRequest, "gagal menyimpan satuan (kode mungkin sudah ada)")
		return
	}
	u.ID = id
	whJSON(w, http.StatusCreated, u)
}

func GetItems(w http.ResponseWriter, r *http.Request) {
	list, err := repository.ListItems(config.DB)
	if err != nil {
		whErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	whJSON(w, http.StatusOK, list)
}

func CreateItem(w http.ResponseWriter, r *http.Request) {
	var it models.Item
	if err := json.NewDecoder(r.Body).Decode(&it); err != nil {
		whErr(w, http.StatusBadRequest, "format JSON tidak valid")
		return
	}
	if msg := validateItem(it); msg != "" {
		whErr(w, http.StatusBadRequest, msg)
		return
	}
	id, err := repository.CreateItem(config.DB, it)
	if err != nil {
		whErr(w, http.StatusBadRequest, "gagal menyimpan bahan (SKU mungkin sudah dipakai atau satuan tidak ada)")
		return
	}
	it.ID = id
	whJSON(w, http.StatusCreated, it)
}

func UpdateItem(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		whErr(w, http.StatusBadRequest, "id tidak valid")
		return
	}
	var it models.Item
	if err := json.NewDecoder(r.Body).Decode(&it); err != nil {
		whErr(w, http.StatusBadRequest, "format JSON tidak valid")
		return
	}
	if msg := validateItem(it); msg != "" {
		whErr(w, http.StatusBadRequest, msg)
		return
	}
	if ok, err := repository.ItemExists(config.DB, id); err != nil || !ok {
		whErr(w, http.StatusNotFound, "bahan tidak ditemukan")
		return
	}
	if err := repository.UpdateItem(config.DB, id, it); err != nil {
		whErr(w, http.StatusBadRequest, "gagal memperbarui bahan")
		return
	}
	it.ID = id
	whJSON(w, http.StatusOK, it)
}

func DeleteItem(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		whErr(w, http.StatusBadRequest, "id tidak valid")
		return
	}
	n, err := repository.DeactivateItem(config.DB, id)
	if err != nil {
		whErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if n == 0 {
		whErr(w, http.StatusNotFound, "bahan tidak ditemukan atau sudah nonaktif")
		return
	}
	whJSON(w, http.StatusOK, map[string]string{"message": "bahan dinonaktifkan"})
}
