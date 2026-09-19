package user_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/MrAndreID/goweb/internal/feature/user"

	"github.com/stretchr/testify/assert"
)

func postForm(path string, values url.Values) *http.Request {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return req
}

func TestHandlerIndex(t *testing.T) {
	t.Run("renders list on success", func(t *testing.T) {
		total := int64(1)
		repo := &mockRepository{
			listFunc: func(_ context.Context, _ user.ListParams) (*user.ListResult, error) {
				return &user.ListResult{
					Users:     []user.User{{ID: "id-1", Name: "Andre", Emails: []user.Email{{Email: "a@x.com"}}}},
					Paginator: user.Paginator{Total: &total, Page: 1, Limit: 10},
				}, nil
			},
		}

		e := newTestServer(t, newServiceWithMock(repo))
		rec := doRequest(e, httptest.NewRequest(http.MethodGet, "/users?success=done", nil))

		assert.Equal(t, http.StatusOK, rec.Code)
		body := rec.Body.String()
		assert.Contains(t, body, "Andre")
		assert.Contains(t, body, "a@x.com")
		assert.Contains(t, body, "done")
	})

	t.Run("propagates request id into service context", func(t *testing.T) {
		repo := &mockRepository{}
		e := newTestServerWithRequestID(t, newServiceWithMock(repo))

		req := httptest.NewRequest(http.MethodGet, "/users", nil)
		req.Header.Set("X-Request-ID", "req-abc")

		rec := doRequest(e, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "req-abc", repo.lastRequestID)
	})

	t.Run("renders error message when service fails", func(t *testing.T) {
		repo := &mockRepository{
			listFunc: func(_ context.Context, _ user.ListParams) (*user.ListResult, error) {
				return nil, user.ErrBackend
			},
		}

		e := newTestServer(t, newServiceWithMock(repo))
		rec := doRequest(e, httptest.NewRequest(http.MethodGet, "/users", nil))

		assert.Equal(t, http.StatusBadGateway, rec.Code)
		assert.Contains(t, rec.Body.String(), "failed to reach the backend")
	})
}

func TestHandlerCreateForm(t *testing.T) {
	e := newTestServer(t, newServiceWithMock(&mockRepository{}))
	rec := doRequest(e, httptest.NewRequest(http.MethodGet, "/users/create", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Create User")
}

func TestHandlerCreate(t *testing.T) {
	t.Run("redirects on success and parses emails", func(t *testing.T) {
		repo := &mockRepository{}
		e := newTestServer(t, newServiceWithMock(repo))

		form := url.Values{}
		form.Set("name", "Andre")
		form.Set("emails", "a@x.com\nb@x.com, c@x.com")

		rec := doRequest(e, postForm("/users", form))

		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/users?success=user+created", rec.Header().Get("Location"))
		assert.Equal(t, []string{"a@x.com", "b@x.com", "c@x.com"}, repo.lastCreate.Emails)
	})

	t.Run("re-renders form with error when service fails", func(t *testing.T) {
		repo := &mockRepository{
			createFunc: func(_ context.Context, _ user.CreateData) (*user.User, error) {
				return nil, user.ErrDuplicateEmail
			},
		}

		e := newTestServer(t, newServiceWithMock(repo))

		form := url.Values{}
		form.Set("name", "Andre")
		form.Set("emails", "a@x.com")

		rec := doRequest(e, postForm("/users", form))

		assert.Equal(t, http.StatusConflict, rec.Code)
		assert.Contains(t, rec.Body.String(), "already used")
	})
}

func TestHandlerCreateHumanizedErrors(t *testing.T) {
	cases := []struct {
		name       string
		formName   string
		formEmails string
		repoErr    error
		wantText   string
	}{
		{
			name:       "name required (service validation)",
			formName:   "   ",
			formEmails: "a@x.com",
			wantText:   "name is required",
		},
		{
			name:       "email required (service validation)",
			formName:   "Andre",
			formEmails: "   ",
			wantText:   "at least one valid email is required",
		},
		{
			name:       "invalid email format (service validation)",
			formName:   "Andre",
			formEmails: "not-an-email",
			wantText:   "one or more email addresses are invalid",
		},
		{
			name:       "validation error from backend",
			formName:   "Andre",
			formEmails: "a@x.com",
			repoErr:    user.ErrValidation,
			wantText:   "the data sent is invalid",
		},
		{
			name:       "not found from backend",
			formName:   "Andre",
			formEmails: "a@x.com",
			repoErr:    user.ErrNotFound,
			wantText:   "user not found",
		},
		{
			name:       "unauthorized from backend",
			formName:   "Andre",
			formEmails: "a@x.com",
			repoErr:    user.ErrUnauthorized,
			wantText:   "the application key was rejected by the backend",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockRepository{}

			if tc.repoErr != nil {
				repo.createFunc = func(_ context.Context, _ user.CreateData) (*user.User, error) {
					return nil, tc.repoErr
				}
			}

			e := newTestServer(t, newServiceWithMock(repo))

			form := url.Values{}
			form.Set("name", tc.formName)
			form.Set("emails", tc.formEmails)

			rec := doRequest(e, postForm("/users", form))

			if tc.repoErr == nil {
				assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
			} else {
				assert.Equal(t, expectedErrorStatus(tc.repoErr), rec.Code)
			}
			assert.Contains(t, rec.Body.String(), tc.wantText)
		})
	}
}

func TestHandlerEditForm(t *testing.T) {
	t.Run("renders form when user found", func(t *testing.T) {
		repo := &mockRepository{
			listFunc: func(_ context.Context, _ user.ListParams) (*user.ListResult, error) {
				return &user.ListResult{Users: []user.User{{
					ID:     "id-1",
					Name:   "Andre",
					Emails: []user.Email{{Email: "a@x.com"}, {Email: "b@x.com"}},
				}}}, nil
			},
		}

		e := newTestServer(t, newServiceWithMock(repo))
		rec := doRequest(e, httptest.NewRequest(http.MethodGet, "/users/3f2504e0-4f89-41d3-9a0c-0305e82c3301/edit", nil))

		assert.Equal(t, http.StatusOK, rec.Code)
		body := rec.Body.String()
		assert.Contains(t, body, "Edit User")
		assert.Contains(t, body, "Andre")
		// joinEmails menaruh satu email per baris.
		assert.Contains(t, body, "a@x.com\nb@x.com")
	})

	t.Run("renders not found when list empty", func(t *testing.T) {
		repo := &mockRepository{
			listFunc: func(_ context.Context, _ user.ListParams) (*user.ListResult, error) {
				return &user.ListResult{Users: []user.User{}}, nil
			},
		}

		e := newTestServer(t, newServiceWithMock(repo))
		rec := doRequest(e, httptest.NewRequest(http.MethodGet, "/users/3f2504e0-4f89-41d3-9a0c-0305e82c3301/edit", nil))

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "user not found")
	})

	t.Run("renders not found when service errors", func(t *testing.T) {
		repo := &mockRepository{
			listFunc: func(_ context.Context, _ user.ListParams) (*user.ListResult, error) {
				return &user.ListResult{}, user.ErrBackend
			},
		}

		e := newTestServer(t, newServiceWithMock(repo))
		rec := doRequest(e, httptest.NewRequest(http.MethodGet, "/users/3f2504e0-4f89-41d3-9a0c-0305e82c3301/edit", nil))

		assert.Equal(t, http.StatusBadGateway, rec.Code)
		assert.Contains(t, rec.Body.String(), "failed to reach the backend")
	})
}

func TestHandlerUpdate(t *testing.T) {
	t.Run("redirects on success", func(t *testing.T) {
		repo := &mockRepository{}
		e := newTestServer(t, newServiceWithMock(repo))

		form := url.Values{}
		form.Set("name", "New Name")
		form.Set("emails", "a@x.com")

		rec := doRequest(e, postForm("/users/3f2504e0-4f89-41d3-9a0c-0305e82c3301", form))

		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/users?success=user+updated", rec.Header().Get("Location"))
	})

	t.Run("re-renders form with error when service fails", func(t *testing.T) {
		repo := &mockRepository{
			updateFunc: func(_ context.Context, _ string, _ user.UpdateData) error {
				return user.ErrValidation
			},
		}

		e := newTestServer(t, newServiceWithMock(repo))

		form := url.Values{}
		form.Set("name", "New Name")
		form.Set("emails", "a@x.com")

		rec := doRequest(e, postForm("/users/3f2504e0-4f89-41d3-9a0c-0305e82c3301", form))

		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "the data sent is invalid")
	})

	t.Run("rejects empty name", func(t *testing.T) {
		repo := &mockRepository{}
		e := newTestServer(t, newServiceWithMock(repo))

		form := url.Values{}
		form.Set("name", "   ")
		form.Set("emails", "a@x.com")

		rec := doRequest(e, postForm("/users/3f2504e0-4f89-41d3-9a0c-0305e82c3301", form))

		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "name is required")
	})

	t.Run("rejects empty emails", func(t *testing.T) {
		repo := &mockRepository{}
		e := newTestServer(t, newServiceWithMock(repo))

		form := url.Values{}
		form.Set("name", "New Name")
		form.Set("emails", "   ")

		rec := doRequest(e, postForm("/users/3f2504e0-4f89-41d3-9a0c-0305e82c3301", form))

		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "at least one valid email is required")
	})
}

func TestHandlerDelete(t *testing.T) {
	t.Run("redirects with deleted message on success", func(t *testing.T) {
		repo := &mockRepository{}
		e := newTestServer(t, newServiceWithMock(repo))

		rec := doRequest(e, postForm("/users/3f2504e0-4f89-41d3-9a0c-0305e82c3301/delete", url.Values{}))

		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/users?success=user+deleted", rec.Header().Get("Location"))
		assert.Equal(t, "3f2504e0-4f89-41d3-9a0c-0305e82c3301", repo.lastDelete)
	})

	t.Run("redirects with failure message when service fails", func(t *testing.T) {
		repo := &mockRepository{
			deleteFunc: func(_ context.Context, _ string) error {
				return user.ErrBackend
			},
		}

		e := newTestServer(t, newServiceWithMock(repo))

		rec := doRequest(e, postForm("/users/3f2504e0-4f89-41d3-9a0c-0305e82c3301/delete", url.Values{}))

		assert.Equal(t, http.StatusBadGateway, rec.Code)
		assert.Contains(t, rec.Body.String(), "failed to reach the backend")
	})
}

func expectedErrorStatus(err error) int {
	switch err {
	case user.ErrValidation:
		return http.StatusUnprocessableEntity
	case user.ErrNotFound:
		return http.StatusNotFound
	case user.ErrUnauthorized:
		return http.StatusBadGateway
	default:
		return http.StatusBadGateway
	}
}
