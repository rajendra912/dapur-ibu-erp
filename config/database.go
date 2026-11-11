package config

import (
	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql" // Driver MySQL
)

// DB adalah variabel global untuk diakses dari file/modul lain
var DB *sql.DB

func ConnectDB() {
	// DSN (Data Source Name) untuk Laragon: root tanpa password.
	// loc=Local wajib agar DATETIME (cutoff_date) dibaca/ditulis memakai zona
	// waktu laptop, sehingga perbandingan dengan NOW() MySQL tetap konsisten.
	dsn := "root:@tcp(127.0.0.1:3306)/dapur_ibu_db?parseTime=true&loc=Local"

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Gagal membuka database: %v", err)
	}

	// Tes ping untuk memastikan MySQL merespons
	if err = db.Ping(); err != nil {
		log.Fatalf("Database tidak merespons. Pastikan MySQL Laragon menyala! Error: %v", err)
	}

	DB = db
	log.Println("Koneksi ke database dapur_ibu_db berhasil!")
}