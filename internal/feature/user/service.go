package user

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrNameRequired  = errors.New("name is required")
	ErrEmailRequired = errors.New("at least one email is required")
)

type Service struct {
	repository InterfaceRepository
}

func NewService(repository InterfaceRepository) *Service {
	return &Service{repository: repository}
}

type InterfaceService interface {
	Create(ctx context.Context, data CreateData) (*User, error)
	List(ctx context.Context, params ListParams) (*ListResult, error)
	Update(ctx context.Context, id string, data UpdateData) error
	Delete(ctx context.Context, id string) error
}

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

func (s *Service) List(ctx context.Context, params ListParams) (*ListResult, error) {
	if params.ID != "" && !isValidID(params.ID) {
		return nil, ErrNotFound
	}

	return s.repository.List(ctx, params)
}

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

func (s *Service) Delete(ctx context.Context, id string) error {
	if !isValidID(id) {
		return ErrNotFound
	}

	return s.repository.Delete(ctx, id)
}

func isValidID(id string) bool {
	id = strings.TrimSpace(id)

	if id == "" {
		return false
	}

	_, err := uuid.Parse(id)

	return err == nil
}

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
