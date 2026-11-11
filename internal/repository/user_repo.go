package repository

import (
	"database/sql"
	"errors"

	"dapur-ibu-erp/internal/models"

	"golang.org/x/crypto/bcrypt"
)

// UserRepository adalah struct untuk menampung koneksi DB
type UserRepository struct {
	DB *sql.DB
}

// HashPassword: Mengubah teks asli menjadi password acak
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

// CheckPasswordHash: Mencocokkan password yang diinput dengan yang ada di DB
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// CreateUser: Memasukkan user baru ke tabel MySQL
func (r *UserRepository) CreateUser(user *models.UserCredentials) error {
	// 1. Hash password-nya dulu sebelum masuk database
	hashedPassword, err := HashPassword(user.Password)
	if err != nil {
		return err
	}

	// 2. Eksekusi query INSERT
	query := `INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)`
	_, err = r.DB.Exec(query, user.Username, hashedPassword, user.Role)
	if err != nil {
		return errors.New("gagal menyimpan user, mungkin username sudah dipakai")
	}

	return nil
}

// GetUserByUsername: Mencari user berdasarkan username (dipakai saat Login)
func (r *UserRepository) GetUserByUsername(username string) (*models.User, error) {
	query := `SELECT id, username, password_hash, role FROM users WHERE username = ?`
	
	row := r.DB.QueryRow(query, username)
	
	var user models.User
	err := row.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("username tidak ditemukan")
		}
		return nil, err
	}
	
	return &user, nil
}