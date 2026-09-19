package user

import "time"

type Email struct {
	ID        string     `json:"id"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt"`
	UserID    string     `json:"userId"`
	Email     string     `json:"email"`
}

type User struct {
	ID        string     `json:"id"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt"`
	Name      string     `json:"name"`
	Emails    []Email    `json:"emails"`
}

type CreateData struct {
	Name   string   `json:"name"`
	Emails []string `json:"emails"`
}

type UpdateData struct {
	Name   *string  `json:"name,omitempty"`
	Emails []string `json:"emails"`
}

type ListParams struct {
	Page                  string
	Limit                 string
	OrderBy               string
	SortBy                string
	Search                string
	DisableCalculateTotal string
	ID                    string
}

type Paginator struct {
	Total *int64
	Page  int
	Limit int
}

type ListResult struct {
	Users     []User
	Paginator Paginator
}
