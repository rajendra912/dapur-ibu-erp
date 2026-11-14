package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/rajendra912/dapur-ibu-erp/config"
	"github.com/rajendra912/dapur-ibu-erp/internal/handler"
	"github.com/rajendra912/dapur-ibu-erp/internal/repository"
	"github.com/rajendra912/dapur-ibu-erp/internal/service"
	"github.com/rajendra912/dapur-ibu-erp/internal/worker"

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

	// 4. Modul MRP & Pembelian (Leo)
	mrpSvc := service.NewMRPService(repository.NewMRPRepo(config.DB))
	purSvc := service.NewPurchaseService(repository.NewPurchaseRepo(config.DB))
	handler.NewMRPHandler(mrpSvc, purSvc).RegisterRoutes(r, nil) // nil = tanpa RBAC dulu

	// 5. Modul Order Management & Automasi Cutoff H-2 (Fiko)
	handler.RegisterOrderRoutes(r)
	go worker.StartCutoffWorker(config.DB)

	// 6. Modul Recipe & Kitchen Engine (Syarif)
	recipeSvc := service.NewRecipeService(repository.NewRecipeRepo(config.DB))
	handler.NewRecipeHandler(recipeSvc).RegisterRoutes(r, nil) // nil = tanpa RBAC dulu

	// 7. Nyalakan Server
	port := ":8080"
	log.Printf("Server siap menerima pesanan di http://localhost%s", port)
	log.Fatal(http.ListenAndServe(port, r))
}