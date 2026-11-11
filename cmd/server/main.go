package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/rajendra912/dapur-ibu-erp/config" // Mengambil fungsi database dari folder config
	"github.com/rajendra912/dapur-ibu-erp/internal/handler"
	"github.com/rajendra912/dapur-ibu-erp/internal/repository"
	"github.com/rajendra912/dapur-ibu-erp/internal/service"

	"github.com/gorilla/mux"
)

func main() {
	// 1. Inisialisasi koneksi Database
	config.ConnectDB()

	// 2. Inisialisasi Router
	r := mux.NewRouter()

	// 3. Buat rute uji coba (Health Check)
	r.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "🚀 Server ERP Katering Dapur Ibu API berjalan lancar!")
	}).Methods("GET")

	// 3b. Modul MRP & Pembelian (Leo)
	// PENTING: sesuaikan "config.DB" dengan variabel koneksi di config/database.go
	mrpSvc := service.NewMRPService(repository.NewMRPRepo(config.DB))
	purSvc := service.NewPurchaseService(repository.NewPurchaseRepo(config.DB))
	handler.NewMRPHandler(mrpSvc, purSvc).RegisterRoutes(r, nil) // nil = tanpa RBAC dulu

	// 4. Nyalakan Server
	port := ":8080"
	log.Printf("Server siap menerima pesanan di http://localhost%s", port)
	log.Fatal(http.ListenAndServe(port, r))
}
