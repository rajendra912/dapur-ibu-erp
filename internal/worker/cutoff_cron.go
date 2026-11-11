package worker

import (
	"database/sql"
	"log"
	"os"
	"strings"
	"time"

	"[github.com/rajendra912/dapur-ibu-erp/](https://github.com/rajendra912/dapur-ibu-erp/)internal/service"
)

// StartCutoffWorker menjalankan penjadwal penguncian pesanan otomatis (Cutoff H-2).
// Worker berjalan sebagai goroutine di latar belakang sehingga tidak membebani
// interaksi pengguna di antarmuka utama (NFR-03).
//
// Jadwal:
//  1. Saat server dinyalakan -> langsung melakukan catch-up (mengunci pesanan
//     yang terlewat, misalnya karena server sempat mati).
//  2. Setiap pergantian hari pukul 00:00 waktu lokal -> mengunci semua pesanan
//     DRAFT/CONFIRMED yang cutoff_date-nya sudah terlewati.
//
// Set environment variable CUTOFF_WORKER=off untuk mematikan worker saat demo
// manual (penguncian tetap bisa dipicu via POST /api/cron/cutoff).
func StartCutoffWorker(db *sql.DB) {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("CUTOFF_WORKER")), "off") {
		log.Println("[cutoff] worker dimatikan lewat CUTOFF_WORKER=off")
		return
	}

	log.Println("[cutoff] worker aktif - penguncian otomatis pesanan H-2 berjalan")

	// Catch-up saat start, agar pesanan yang seharusnya terkunci tidak terlewat.
	runCutoff(db, "startup")

	for {
		next := NextMidnight(time.Now())
		log.Printf("[cutoff] pengecekan berikutnya pada %s", next.Format("2006-01-02 15:04:05"))
		time.Sleep(time.Until(next))
		runCutoff(db, "jadwal 00:00")
	}
}

// NextMidnight mengembalikan waktu 00:00:00 berikutnya pada zona waktu t.
func NextMidnight(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location()).AddDate(0, 0, 1)
}

func runCutoff(db *sql.DB, trigger string) {
	start := time.Now()
	res, err := service.LockDueOrders(db, start)
	if err != nil {
		log.Printf("[cutoff] %s: GAGAL mengunci pesanan: %v", trigger, err)
		return
	}
	log.Printf("[cutoff] %s: %d pesanan dikunci dalam %s", trigger, res.LockedCount, time.Since(start).Truncate(time.Millisecond))
	for _, o := range res.Orders {
		log.Printf("[cutoff]   - %s (id %d, acara %s, %d porsi, HPP/porsi %.2f)",
			o.OrderNum, o.OrderID, o.EventDate, o.TotalPortions, o.LockedHPPPerPortion)
	}
}
