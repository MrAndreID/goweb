package user_test

import (
	"context"
	"errors"
	"testing"

	"github.com/MrAndreID/goweb/internal/feature/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validUserID adalah UUID contoh (dari kontrak backend) yang lolos validasi.
const validUserID = "3f2504e0-4f89-41d3-9a0c-0305e82c3301"

func TestServiceCreate(t *testing.T) {
	t.Run("trims name and sanitizes emails then delegates", func(t *testing.T) {
		repo := &mockRepository{
			createFunc: func(_ context.Context, data user.CreateData) (*user.User, error) {
				return &user.User{ID: "id-1", Name: data.Name, Emails: []user.Email{{Email: data.Emails[0]}}}, nil
			},
		}

		service := newServiceWithMock(repo)

		result, err := service.Create(context.Background(), user.CreateData{
			Name:   "  Andre  ",
			Emails: []string{"  andre@gmail.com  ", "   ", ""},
		})

		require.NoError(t, err)
		assert.Equal(t, "id-1", result.ID)
		assert.Equal(t, "Andre", repo.lastCreate.Name)
		assert.Equal(t, []string{"andre@gmail.com"}, repo.lastCreate.Emails)
	})

	t.Run("returns ErrNameRequired when name empty after trim", func(t *testing.T) {
		repo := &mockRepository{}
		service := newServiceWithMock(repo)

		result, err := service.Create(context.Background(), user.CreateData{
			Name:   "   ",
			Emails: []string{"andre@gmail.com"},
		})

		assert.Nil(t, result)
		assert.ErrorIs(t, err, user.ErrNameRequired)
	})

	t.Run("returns ErrEmailRequired when no valid email", func(t *testing.T) {
		repo := &mockRepository{}
		service := newServiceWithMock(repo)

		result, err := service.Create(context.Background(), user.CreateData{
			Name:   "Andre",
			Emails: []string{"  ", ""},
		})

		assert.Nil(t, result)
		assert.ErrorIs(t, err, user.ErrEmailRequired)
	})

	t.Run("returns ErrEmailInvalid when an email has invalid format", func(t *testing.T) {
		repo := &mockRepository{}
		service := newServiceWithMock(repo)

		result, err := service.Create(context.Background(), user.CreateData{
			Name:   "Andre",
			Emails: []string{"not-an-email"},
		})

		assert.Nil(t, result)
		assert.ErrorIs(t, err, user.ErrEmailInvalid)
	})

	t.Run("propagates repository error", func(t *testing.T) {
		sentinel := errors.New("boom")
		repo := &mockRepository{
			createFunc: func(_ context.Context, _ user.CreateData) (*user.User, error) {
				return nil, sentinel
			},
		}

		service := newServiceWithMock(repo)

		_, err := service.Create(context.Background(), user.CreateData{
			Name:   "Andre",
			Emails: []string{"andre@gmail.com"},
		})

		assert.ErrorIs(t, err, sentinel)
	})
}

func TestServiceList(t *testing.T) {
	total := int64(5)

	repo := &mockRepository{
		listFunc: func(_ context.Context, params user.ListParams) (*user.ListResult, error) {
			return &user.ListResult{
				Users:     []user.User{{ID: "id-1"}},
				Paginator: user.Paginator{Total: &total, Page: 1, Limit: 10},
			}, nil
		},
	}

	service := newServiceWithMock(repo)

	result, err := service.List(context.Background(), user.ListParams{Page: "1"})

	require.NoError(t, err)
	require.Len(t, result.Users, 1)
	assert.Equal(t, int64(5), *result.Paginator.Total)
}

func TestServiceListWithIDFilter(t *testing.T) {
	t.Run("passes through when id filter is a valid uuid", func(t *testing.T) {
		repo := &mockRepository{}
		service := newServiceWithMock(repo)

		_, err := service.List(context.Background(), user.ListParams{ID: validUserID})

		require.NoError(t, err)
		assert.Equal(t, validUserID, repo.lastListID)
	})

	t.Run("returns ErrNotFound when id filter is not a valid uuid", func(t *testing.T) {
		repo := &mockRepository{}
		service := newServiceWithMock(repo)

		result, err := service.List(context.Background(), user.ListParams{ID: "bad-id"})

		assert.Nil(t, result)
		assert.ErrorIs(t, err, user.ErrNotFound)
	})
}

func TestServiceUpdate(t *testing.T) {
	t.Run("returns ErrNotFound when id blank", func(t *testing.T) {
		repo := &mockRepository{}
		service := newServiceWithMock(repo)

		err := service.Update(context.Background(), "   ", user.UpdateData{})

		assert.ErrorIs(t, err, user.ErrNotFound)
	})

	t.Run("returns ErrNotFound when id is not a valid uuid", func(t *testing.T) {
		repo := &mockRepository{}
		service := newServiceWithMock(repo)

		err := service.Update(context.Background(), "not-a-uuid", user.UpdateData{})

		assert.ErrorIs(t, err, user.ErrNotFound)
	})

	t.Run("trims name and sanitizes emails when provided", func(t *testing.T) {
		name := "  New Name  "
		repo := &mockRepository{}
		service := newServiceWithMock(repo)

		err := service.Update(context.Background(), validUserID, user.UpdateData{
			Name:   &name,
			Emails: []string{" a@x.com ", ""},
		})

		require.NoError(t, err)
		require.NotNil(t, repo.lastUpdate.Name)
		assert.Equal(t, "New Name", *repo.lastUpdate.Name)
		assert.Equal(t, []string{"a@x.com"}, repo.lastUpdate.Emails)
	})

	t.Run("keeps nil name and nil emails untouched", func(t *testing.T) {
		repo := &mockRepository{}
		service := newServiceWithMock(repo)

		err := service.Update(context.Background(), validUserID, user.UpdateData{})

		require.NoError(t, err)
		assert.Nil(t, repo.lastUpdate.Name)
		assert.Nil(t, repo.lastUpdate.Emails)
	})

	t.Run("returns ErrNameRequired when provided name is empty", func(t *testing.T) {
		name := "   "
		repo := &mockRepository{}
		service := newServiceWithMock(repo)

		err := service.Update(context.Background(), validUserID, user.UpdateData{Name: &name})

		assert.ErrorIs(t, err, user.ErrNameRequired)
	})

	t.Run("returns ErrEmailRequired when provided emails are empty", func(t *testing.T) {
		repo := &mockRepository{}
		service := newServiceWithMock(repo)

		err := service.Update(context.Background(), validUserID, user.UpdateData{Emails: []string{" ", ""}})

		assert.ErrorIs(t, err, user.ErrEmailRequired)
	})

	t.Run("returns ErrEmailInvalid when provided email has invalid format", func(t *testing.T) {
		repo := &mockRepository{}
		service := newServiceWithMock(repo)

		err := service.Update(context.Background(), validUserID, user.UpdateData{Emails: []string{"not-an-email"}})

		assert.ErrorIs(t, err, user.ErrEmailInvalid)
	})
}

func TestServiceDelete(t *testing.T) {
	t.Run("returns ErrNotFound when id blank", func(t *testing.T) {
		repo := &mockRepository{}
		service := newServiceWithMock(repo)

		err := service.Delete(context.Background(), "  ")

		assert.ErrorIs(t, err, user.ErrNotFound)
	})

	t.Run("returns ErrNotFound when id is not a valid uuid", func(t *testing.T) {
		repo := &mockRepository{}
		service := newServiceWithMock(repo)

		err := service.Delete(context.Background(), "12345")

		assert.ErrorIs(t, err, user.ErrNotFound)
	})

	t.Run("delegates to repository when id present", func(t *testing.T) {
		repo := &mockRepository{}
		service := newServiceWithMock(repo)

		err := service.Delete(context.Background(), validUserID)

		require.NoError(t, err)
		assert.Equal(t, validUserID, repo.lastDelete)
	})
}
