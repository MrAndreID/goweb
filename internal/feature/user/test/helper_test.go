package user_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MrAndreID/gomiddleware/v2"
	"github.com/MrAndreID/goweb/internal/application/view"
	"github.com/MrAndreID/goweb/internal/entity"
	"github.com/MrAndreID/goweb/internal/feature/user"

	"github.com/labstack/echo/v5"
)

// mockRepository adalah implementasi InterfaceRepository yang dapat diprogram
// per test untuk menguji service dan handler tanpa menyentuh jaringan.
type mockRepository struct {
	createFunc func(ctx context.Context, data user.CreateData) (*user.User, error)
	listFunc   func(ctx context.Context, params user.ListParams) (*user.ListResult, error)
	updateFunc func(ctx context.Context, id string, data user.UpdateData) error
	deleteFunc func(ctx context.Context, id string) error

	// Menyimpan argumen terakhir untuk assertion.
	lastCreate    user.CreateData
	lastUpdate    user.UpdateData
	lastListID    string
	lastDelete    string
	lastRequestID string
}

func (m *mockRepository) Create(ctx context.Context, data user.CreateData) (*user.User, error) {
	m.lastCreate = data

	if m.createFunc != nil {
		return m.createFunc(ctx, data)
	}

	return &user.User{ID: "generated-id", Name: data.Name}, nil
}

func (m *mockRepository) List(ctx context.Context, params user.ListParams) (*user.ListResult, error) {
	m.lastListID = params.ID
	m.lastRequestID = entity.RequestIDFromContext(ctx)

	if m.listFunc != nil {
		return m.listFunc(ctx, params)
	}

	return &user.ListResult{}, nil
}

func (m *mockRepository) Update(ctx context.Context, id string, data user.UpdateData) error {
	m.lastUpdate = data

	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, data)
	}

	return nil
}

func (m *mockRepository) Delete(ctx context.Context, id string) error {
	m.lastDelete = id

	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}

	return nil
}

// newServiceWithMock membangun Service asli di atas mockRepository.
func newServiceWithMock(repo user.InterfaceRepository) *user.Service {
	return user.NewService(repo)
}

// newTestServer membangun Echo dengan renderer global (layout + content feature)
// dan handler user terdaftar, siap dipakai httptest.
func newTestServer(t *testing.T, service user.InterfaceService) *echo.Echo {
	t.Helper()

	renderer, err := view.NewRenderer(user.Views)

	if err != nil {
		t.Fatalf("failed to build renderer: %v", err)
	}

	e := echo.New()
	e.Renderer = renderer

	user.NewHandler(e.Group(""), service)

	return e
}

// newTestServerWithRequestID sama seperti newTestServer namun memasang
// middleware EchoSetRequestID agar request ID tersedia di context, untuk
// menguji propagasi korelasi log dari handler ke service.
func newTestServerWithRequestID(t *testing.T, service user.InterfaceService) *echo.Echo {
	t.Helper()

	renderer, err := view.NewRenderer(user.Views)

	if err != nil {
		t.Fatalf("failed to build renderer: %v", err)
	}

	e := echo.New()
	e.Renderer = renderer
	e.Pre(gomiddleware.EchoSetRequestID)

	user.NewHandler(e.Group(""), service)

	return e
}

// doRequest menjalankan sebuah request terhadap Echo dan mengembalikan recorder.
func doRequest(e *echo.Echo, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	return rec
}
