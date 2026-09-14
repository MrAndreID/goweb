package user

import "time"

// Email merepresentasikan satu email milik user (mengikuti kontrak backend).
type Email struct {
	ID        string     `json:"id"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt"`
	UserID    string     `json:"userId"`
	Email     string     `json:"email"`
}

// User merepresentasikan satu user beserta daftar email-nya.
type User struct {
	ID        string     `json:"id"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt"`
	Name      string     `json:"name"`
	Emails    []Email    `json:"emails"`
}

// CreateData adalah payload untuk membuat user (POST /api/v1/user).
// Emails wajib berisi minimal satu email valid.
type CreateData struct {
	Name   string   `json:"name"`
	Emails []string `json:"emails"`
}

// UpdateData adalah payload untuk memperbarui user (PATCH /api/v1/user/{id}).
// Kedua field opsional; bila Emails diisi, seluruh email lama diganti.
type UpdateData struct {
	Name   *string  `json:"name,omitempty"`
	Emails []string `json:"emails,omitempty"`
}

// ListParams memuat filter/pagination untuk daftar user (GET /api/v1/user).
// Seluruh nilai berupa string mengikuti kontrak query backend.
type ListParams struct {
	Page                  string
	Limit                 string
	OrderBy               string
	SortBy                string
	Search                string
	DisableCalculateTotal string
	ID                    string
}

// Paginator adalah metadata pagination yang dikembalikan bersama daftar user.
type Paginator struct {
	Total *int64
	Page  int
	Limit int
}

// ListResult membungkus hasil daftar user beserta metadata pagination.
type ListResult struct {
	Users     []User
	Paginator Paginator
}
