package handler

import (
	"encoding/json"
	"net/http"

	"dapur-ibu-erp/internal/models"
	"dapur-ibu-erp/internal/repository"
)

// UserHandler adalah struct yang menampung akses ke Repository
type UserHandler struct {
	Repo *repository.UserRepository
}

// RegisterUser: Endpoint untuk mendaftarkan user baru
func (h *UserHandler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	// 1. Tangkap data JSON dari request body
	var creds models.UserCredentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, `{"error": "Format JSON tidak valid"}`, http.StatusBadRequest)
		return
	}

	// 2. Validasi sederhana
	if creds.Username == "" || creds.Password == "" || creds.Role == "" {
		http.Error(w, `{"error": "Username, password, dan role tidak boleh kosong"}`, http.StatusBadRequest)
		return
	}

	// 3. Panggil fungsi simpan ke database
	err := h.Repo.CreateUser(&creds)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// 4. Kirim respons sukses
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "User " + creds.Username + " berhasil didaftarkan!",
	})
}