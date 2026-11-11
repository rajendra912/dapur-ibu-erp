package models

// User mencerminkan tabel 'users' di database
type User struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"` // Tanda "-" artinya password tidak akan pernah dimunculkan saat data diubah ke JSON (demi keamanan)
	Role         string `json:"role"`
}

// Struct untuk menerima data saat registrasi atau login
type UserCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"` // Opsional saat registrasi
}