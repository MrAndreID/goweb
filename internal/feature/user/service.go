package user

import (
	"context"
	"errors"
	"strings"
	"uuid"
)

// Kesalahan validasi tingkat proses bisnis (sebelum menyentuh backend).
var (
	ErrNameRequired  = errors.New("name is required")
	ErrEmailRequired = errors.New("at least one email is required")
)

// Service memuat proses bisnis feature user. Lapisan ini tidak mengetahui
// detail HTTP maupun framework; ia hanya berbicara ke port InterfaceRepository.
type Service struct {
	repository InterfaceRepository
}

// NewService membangun service dengan dependensi repository.
func NewService(repository InterfaceRepository) *Service {
	return &Service{repository: repository}
}

// InterfaceService adalah port masuk (inbound) yang dipakai handler.
type InterfaceService interface {
	Create(ctx context.Context, data CreateData) (*User, error)
	List(ctx context.Context, params ListParams) (*ListResult, error)
	Update(ctx context.Context, id string, data UpdateData) error
	Delete(ctx context.Context, id string) error
}

// Create memvalidasi input lalu meminta repository membuat user di backend.
func (s *Service) Create(ctx context.Context, data CreateData) (*User, error) {
	data.Name = strings.TrimSpace(data.Name)
	data.Emails = sanitizeEmails(data.Emails)

	if data.Name == "" {
		return nil, ErrNameRequired
	}

	if len(data.Emails) == 0 {
		return nil, ErrEmailRequired
	}

	return s.repository.Create(ctx, data)
}

// List meneruskan filter/pagination ke backend. Bila filter ID diisi, ID wajib
// berupa UUID yang valid agar tidak melakukan round-trip untuk id yang jelas
// salah.
func (s *Service) List(ctx context.Context, params ListParams) (*ListResult, error) {
	if params.ID != "" && !isValidID(params.ID) {
		return nil, ErrNotFound
	}

	return s.repository.List(ctx, params)
}

// Update memvalidasi input parsial lalu meminta repository memperbaruinya.
func (s *Service) Update(ctx context.Context, id string, data UpdateData) error {
	if !isValidID(id) {
		return ErrNotFound
	}

	if data.Name != nil {
		name := strings.TrimSpace(*data.Name)
		data.Name = &name
	}

	if data.Emails != nil {
		data.Emails = sanitizeEmails(data.Emails)
	}

	return s.repository.Update(ctx, id, data)
}

// Delete meminta repository menghapus user (soft delete di backend).
func (s *Service) Delete(ctx context.Context, id string) error {
	if !isValidID(id) {
		return ErrNotFound
	}

	return s.repository.Delete(ctx, id)
}

// isValidID memastikan id tidak kosong dan berformat UUID sesuai kontrak backend.
func isValidID(id string) bool {
	id = strings.TrimSpace(id)

	if id == "" {
		return false
	}

	_, err := uuid.Parse(id)

	return err == nil
}

// sanitizeEmails membuang spasi dan elemen kosong dari daftar email.
func sanitizeEmails(emails []string) []string {
	cleaned := make([]string, 0, len(emails))

	for _, email := range emails {
		email = strings.TrimSpace(email)

		if email != "" {
			cleaned = append(cleaned, email)
		}
	}

	return cleaned
}
