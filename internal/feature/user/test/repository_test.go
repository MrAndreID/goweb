package user_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MrAndreID/goweb/internal/entity"
	"github.com/MrAndreID/goweb/internal/feature/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newBackend membangun httptest server yang menjalankan handler yang diberikan,
// lalu membangun repository asli yang menunjuk ke server tersebut.
func newBackend(t *testing.T, handler http.HandlerFunc) (*user.Repository, *httptest.Server) {
	t.Helper()

	server := httptest.NewServer(handler)

	t.Cleanup(server.Close)

	repo := user.NewRepository(user.RepositoryConfig{
		BaseURL:          server.URL,
		AppKey:           "test-key",
		Timeout:          5,
		MaxResponseBytes: 1 << 20,
		MaxIdleConns:     10,
		MaxConnsPerHost:  10,
		RetryCount:       0,
	})

	return repo, server
}

// unreachableRepo membangun repository yang menunjuk ke alamat yang tidak bisa
// dihubungi, untuk memicu cabang error transport (bukan error status).
func unreachableRepo() *user.Repository {
	return user.NewRepository(user.RepositoryConfig{
		BaseURL:          "http://127.0.0.1:0",
		AppKey:           "test-key",
		Timeout:          1,
		MaxResponseBytes: 1 << 20,
		MaxIdleConns:     10,
		MaxConnsPerHost:  10,
		RetryCount:       0,
	})
}

func TestRepositoryCreate(t *testing.T) {
	t.Run("success returns user and forwards app key", func(t *testing.T) {
		repo, _ := newBackend(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "test-key", r.Header.Get("X-App-Key"))
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/api/v1/user", r.URL.Path)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"status":true,"message":"CREATED","data":{"id":"id-1","name":"Andre","emails":[]},"meta":null,"error":null}`))
		})

		result, err := repo.Create(context.Background(), user.CreateData{Name: "Andre", Emails: []string{"a@x.com"}})

		require.NoError(t, err)
		assert.Equal(t, "id-1", result.ID)
		assert.Equal(t, "Andre", result.Name)
	})

	t.Run("maps 409 to ErrDuplicateEmail", func(t *testing.T) {
		repo, _ := newBackend(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusConflict)
		})

		_, err := repo.Create(context.Background(), user.CreateData{Name: "Andre", Emails: []string{"a@x.com"}})

		assert.ErrorIs(t, err, user.ErrDuplicateEmail)
	})

	t.Run("forwards X-Request-ID from context for log correlation", func(t *testing.T) {
		repo, _ := newBackend(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "req-123", r.Header.Get("X-Request-ID"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"status":true,"message":"CREATED","data":{"id":"id-1","name":"Andre","emails":[]},"meta":null,"error":null}`))
		})

		ctx := entity.ContextWithRequestID(context.Background(), "req-123")

		_, err := repo.Create(ctx, user.CreateData{Name: "Andre", Emails: []string{"a@x.com"}})

		require.NoError(t, err)
	})

	t.Run("omits X-App-Key header when app key is empty", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Empty(t, r.Header.Get("X-App-Key"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"status":true,"message":"CREATED","data":{"id":"id-1","name":"Andre","emails":[]},"meta":null,"error":null}`))
		}))
		t.Cleanup(server.Close)

		repo := user.NewRepository(user.RepositoryConfig{
			BaseURL:          server.URL,
			AppKey:           "",
			Timeout:          5,
			MaxResponseBytes: 1 << 20,
			MaxIdleConns:     10,
			MaxConnsPerHost:  10,
			RetryCount:       0,
		})

		_, err := repo.Create(context.Background(), user.CreateData{Name: "Andre", Emails: []string{"a@x.com"}})

		require.NoError(t, err)
	})

	t.Run("transport error returns ErrBackend", func(t *testing.T) {
		// Base URL yang tidak bisa dihubungi memaksa cabang error transport.
		repo := unreachableRepo()

		_, err := repo.Create(context.Background(), user.CreateData{Name: "Andre", Emails: []string{"a@x.com"}})

		assert.ErrorIs(t, err, user.ErrBackend)
	})
}

func TestRepositoryList(t *testing.T) {
	t.Run("success with meta populates paginator and forwards filled query only", func(t *testing.T) {
		repo, _ := newBackend(t, func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()

			// Hanya param terisi yang dikirim (buildListQuery menyaring kosong).
			assert.Equal(t, "2", query.Get("page"))
			assert.Equal(t, "5", query.Get("limit"))
			assert.Equal(t, "andre", query.Get("search"))
			assert.False(t, query.Has("orderBy"))
			assert.False(t, query.Has("sortBy"))
			assert.False(t, query.Has("disableCalculateTotal"))
			assert.False(t, query.Has("id"))

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":true,"message":"OK","data":[{"id":"id-1","name":"Andre","emails":[]}],"meta":{"total":1,"page":2,"limit":5},"error":null}`))
		})

		result, err := repo.List(context.Background(), user.ListParams{Page: "2", Limit: "5", Search: "andre"})

		require.NoError(t, err)
		require.Len(t, result.Users, 1)
		require.NotNil(t, result.Paginator.Total)
		assert.Equal(t, int64(1), *result.Paginator.Total)
		assert.Equal(t, 2, result.Paginator.Page)
		assert.Equal(t, 5, result.Paginator.Limit)
	})

	t.Run("success with nil meta leaves paginator zero", func(t *testing.T) {
		repo, _ := newBackend(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":true,"message":"OK","data":[],"meta":null,"error":null}`))
		})

		result, err := repo.List(context.Background(), user.ListParams{})

		require.NoError(t, err)
		assert.Empty(t, result.Users)
		assert.Nil(t, result.Paginator.Total)
		assert.Equal(t, 0, result.Paginator.Page)
	})

	t.Run("maps 400 to ErrValidation", func(t *testing.T) {
		repo, _ := newBackend(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		})

		_, err := repo.List(context.Background(), user.ListParams{})

		assert.ErrorIs(t, err, user.ErrValidation)
	})

	t.Run("transport error returns ErrBackend", func(t *testing.T) {
		repo := unreachableRepo()

		_, err := repo.List(context.Background(), user.ListParams{})

		assert.ErrorIs(t, err, user.ErrBackend)
	})
}

func TestRepositoryUpdate(t *testing.T) {
	t.Run("success sends id in path", func(t *testing.T) {
		repo, _ := newBackend(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPatch, r.Method)
			assert.Equal(t, "/api/v1/user/id-1", r.URL.Path)

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":true,"message":"OK","data":null,"meta":null,"error":null}`))
		})

		name := "Updated"
		err := repo.Update(context.Background(), "id-1", user.UpdateData{Name: &name})

		require.NoError(t, err)
	})

	t.Run("maps 401 to ErrUnauthorized", func(t *testing.T) {
		repo, _ := newBackend(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		})

		err := repo.Update(context.Background(), "id-1", user.UpdateData{})

		assert.ErrorIs(t, err, user.ErrUnauthorized)
	})

	t.Run("maps 403 to ErrUnauthorized", func(t *testing.T) {
		repo, _ := newBackend(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		})

		err := repo.Update(context.Background(), "id-1", user.UpdateData{})

		assert.ErrorIs(t, err, user.ErrUnauthorized)
	})

	t.Run("maps 404 to ErrNotFound", func(t *testing.T) {
		repo, _ := newBackend(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})

		err := repo.Update(context.Background(), "id-1", user.UpdateData{})

		assert.ErrorIs(t, err, user.ErrNotFound)
	})

	t.Run("transport error returns ErrBackend", func(t *testing.T) {
		repo := unreachableRepo()

		err := repo.Update(context.Background(), "id-1", user.UpdateData{})

		assert.ErrorIs(t, err, user.ErrBackend)
	})
}

func TestRepositoryDelete(t *testing.T) {
	t.Run("success sends delete", func(t *testing.T) {
		repo, _ := newBackend(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodDelete, r.Method)
			assert.Equal(t, "/api/v1/user/id-1", r.URL.Path)

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":true,"message":"OK","data":null,"meta":null,"error":null}`))
		})

		err := repo.Delete(context.Background(), "id-1")

		require.NoError(t, err)
	})

	t.Run("maps 500 to ErrBackend", func(t *testing.T) {
		repo, _ := newBackend(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		err := repo.Delete(context.Background(), "id-1")

		assert.ErrorIs(t, err, user.ErrBackend)
	})

	t.Run("transport error returns ErrBackend", func(t *testing.T) {
		repo := unreachableRepo()

		err := repo.Delete(context.Background(), "id-1")

		assert.ErrorIs(t, err, user.ErrBackend)
	})
}
