package service

import (
	"errors"
	"testing"
	"time"

	"dapur-ibu-erp/internal/models"
)

func orderWithStatus(status string, cutoff time.Time) models.Order {
	return models.Order{Status: status, CutoffDate: cutoff}
}

func TestComputeCutoffDate(t *testing.T) {
	cases := []struct {
		event string
		want  string
	}{
		// Contoh dari jobdesc: acara 20 Nov 2026 -> cutoff 19 Nov 2026 00:00.
		{"2026-11-20", "2026-11-19 00:00:00"},
		// Pergantian bulan dan tahun harus benar.
		{"2026-12-01", "2026-11-30 00:00:00"},
		{"2027-01-01", "2026-12-31 00:00:00"},
	}
	for _, c := range cases {
		event, err := ParseDate(c.event)
		if err != nil {
			t.Fatalf("ParseDate(%q) error: %v", c.event, err)
		}
		got := ComputeCutoffDate(event).Format("2006-01-02 15:04:05")
		if got != c.want {
			t.Errorf("ComputeCutoffDate(%s) = %s, want %s", c.event, got, c.want)
		}
	}
}

func TestIsEditable(t *testing.T) {
	cutoff := time.Date(2026, 11, 19, 0, 0, 0, 0, time.Local)

	cases := []struct {
		name   string
		status string
		now    time.Time
		want   bool
	}{
		{"DRAFT sebelum cutoff", "DRAFT", cutoff.Add(-time.Hour), true},
		{"CONFIRMED sebelum cutoff", "CONFIRMED", cutoff.Add(-time.Minute), true},
		{"DRAFT tepat saat cutoff", "DRAFT", cutoff, false},
		{"CONFIRMED setelah cutoff", "CONFIRMED", cutoff.Add(time.Hour), false},
		{"LOCKED sebelum cutoff", "LOCKED", cutoff.Add(-time.Hour), false},
		{"DONE sebelum cutoff", "DONE", cutoff.Add(-time.Hour), false},
	}
	for _, c := range cases {
		o := orderWithStatus(c.status, cutoff)
		if got := IsEditable(o, c.now); got != c.want {
			t.Errorf("%s: IsEditable = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestParseDate(t *testing.T) {
	valid := []string{"2026-11-20", "2026-11-20 13:45:00", "2026-11-20T13:45:00+07:00"}
	for _, s := range valid {
		if _, err := ParseDate(s); err != nil {
			t.Errorf("ParseDate(%q) seharusnya valid, error: %v", s, err)
		}
	}

	invalidDates := []string{"", "20-11-2026", "besok", "2026-13-40"}
	for _, s := range invalidDates {
		if _, err := ParseDate(s); err == nil {
			t.Errorf("ParseDate(%q) seharusnya tidak valid", s)
		} else {
			var ve ValidationError
			if !errors.As(err, &ve) {
				t.Errorf("ParseDate(%q) harus mengembalikan ValidationError, dapat %T", s, err)
			}
		}
	}
}

func TestNormalizeStatus(t *testing.T) {
	if got, err := normalizeStatus(" draft ", "DRAFT", "CONFIRMED"); err != nil || got != "DRAFT" {
		t.Errorf("normalizeStatus lowercase = %q, %v", got, err)
	}
	if _, err := normalizeStatus("LOCKED", "DRAFT", "CONFIRMED"); err == nil {
		t.Error("normalizeStatus(LOCKED) untuk create seharusnya error")
	}
}
