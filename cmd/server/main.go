package main

import (
	"fmt"
	"log"
	"net/http"

	"dapur-ibu-erp/config" // Mengambil fungsi database dari folder config
	"dapur-ibu-erp/internal/handler"
	"dapur-ibu-erp/internal/worker"

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

	// 4. [FIKO] Modul Order Management & Automasi Cutoff H-2
	//    Semua rute pesanan didaftarkan lewat satu fungsi agar mudah di-merge.
	handler.RegisterOrderRoutes(r)

	// 5. [FIKO] Jalankan worker penguncian pesanan otomatis di latar belakang
	go worker.StartCutoffWorker(config.DB)

	// 6. Nyalakan Server
	port := ":8080"
	log.Printf("Server siap menerima pesanan di http://localhost%s", port)
	log.Fatal(http.ListenAndServe(port, r))
}
