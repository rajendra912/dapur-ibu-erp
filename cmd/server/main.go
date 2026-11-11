package main

import (
	"fmt"
	"log"
	"net/http"

	"dapur-ibu-erp/config"
	"dapur-ibu-erp/internal/handler"
	"dapur-ibu-erp/internal/repository"

	"github.com/gorilla/mux"
)

func main() {
	// 1. Inisialisasi koneksi Database
	config.ConnectDB()

	// 2. Inisialisasi Router & Komponen
	r := mux.NewRouter()
	userRepo := &repository.UserRepository{DB: config.DB}
	userHandler := &handler.UserHandler{Repo: userRepo}

	// 3. Daftarkan Rute (Endpoints)
	// Rute Health Check
	r.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "🚀 Server ERP Katering Dapur Ibu API berjalan lancar!")
	}).Methods("GET")

	// Rute Registrasi User Baru (POST)
	r.HandleFunc("/api/register", userHandler.RegisterUser).Methods("POST")

	// 4. Nyalakan Server
	port := ":8080"
	log.Printf("Server siap menerima pesanan di http://localhost%s", port)
	log.Fatal(http.ListenAndServe(port, r))
}