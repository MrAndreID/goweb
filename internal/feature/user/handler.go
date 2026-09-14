package user

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/MrAndreID/goweb/internal/entity"
	"github.com/labstack/echo/v5"
	"github.com/sirupsen/logrus"
)

// handler adalah adapter masuk berbasis Echo v5. Hanya lapisan ini yang
// mengetahui detail framework (routing, form parsing, rendering HTML).
type handler struct {
	service InterfaceService
}

// NewHandler mendaftarkan seluruh route SSR feature user.
//
// Form HTML hanya mendukung GET dan POST, sehingga update dan delete
// diekspos sebagai POST ke sub-path (bukan PATCH/DELETE).
func NewHandler(g *echo.Group, service InterfaceService) *handler {
	h := &handler{service: service}

	g.GET("/users", h.Index)
	g.GET("/users/create", h.CreateForm)
	g.POST("/users", h.Create)
	g.GET("/users/:id/edit", h.EditForm)
	g.POST("/users/:id", h.Update)
	g.POST("/users/:id/delete", h.Delete)

	return h
}

// pageData adalah bentuk data seragam yang dikirim ke template.
type pageData struct {
	Title     string
	Users     []User
	User      *User
	Paginator Paginator
	Form      formValues
	Error     string
	Success   string
}

type formValues struct {
	ID     string
	Name   string
	Emails string
}

// requestContext mengambil request ID yang dipasang middleware Echo lalu
// menautkannya ke context request, sehingga lapisan repository bisa
// meneruskannya ke backend untuk korelasi log tanpa mengenal Echo.
func requestContext(c *echo.Context) context.Context {
	ctx := c.Request().Context()

	if requestID, ok := c.Get("RequestID").(*string); ok && requestID != nil {
		ctx = entity.ContextWithRequestID(ctx, *requestID)
	}

	return ctx
}

func (h *handler) Index(c *echo.Context) error {
	var tag string = "internal.feature.user.handler.Index."

	params := ListParams{
		Page:                  c.QueryParam("page"),
		Limit:                 c.QueryParam("limit"),
		OrderBy:               c.QueryParam("orderBy"),
		SortBy:                c.QueryParam("sortBy"),
		Search:                c.QueryParam("search"),
		DisableCalculateTotal: c.QueryParam("disableCalculateTotal"),
		ID:                    c.QueryParam("id"),
	}

	result, err := h.service.List(requestContext(c), params)

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"tag":   tag + "01",
			"error": err.Error(),
		}).Error("failed to list users")

		return c.Render(http.StatusOK, "users_index", pageData{
			Title: "Users",
			Error: humanizeError(err),
		})
	}

	return c.Render(http.StatusOK, "users_index", pageData{
		Title:     "Users",
		Users:     result.Users,
		Paginator: result.Paginator,
		Success:   c.QueryParam("success"),
	})
}

func (h *handler) CreateForm(c *echo.Context) error {
	return c.Render(http.StatusOK, "users_form", pageData{
		Title: "Create User",
	})
}

func (h *handler) Create(c *echo.Context) error {
	var tag string = "internal.feature.user.handler.Create."

	form := formValues{
		Name:   c.FormValue("name"),
		Emails: c.FormValue("emails"),
	}

	data := CreateData{
		Name:   form.Name,
		Emails: parseEmails(form.Emails),
	}

	if _, err := h.service.Create(requestContext(c), data); err != nil {
		logrus.WithFields(logrus.Fields{
			"tag":   tag + "01",
			"error": err.Error(),
		}).Error("failed to create user")

		return c.Render(http.StatusOK, "users_form", pageData{
			Title: "Create User",
			Form:  form,
			Error: humanizeError(err),
		})
	}

	return c.Redirect(http.StatusSeeOther, "/users?success=user+created")
}

func (h *handler) EditForm(c *echo.Context) error {
	var tag string = "internal.feature.user.handler.EditForm."

	id := c.Param("id")

	result, err := h.service.List(requestContext(c), ListParams{ID: id})

	if err != nil || len(result.Users) == 0 {
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"tag":   tag + "01",
				"error": err.Error(),
			}).Error("failed to load user for edit")
		}

		return c.Render(http.StatusOK, "users_form", pageData{
			Title: "Edit User",
			Error: "user not found",
			Form:  formValues{ID: id},
		})
	}

	current := result.Users[0]

	return c.Render(http.StatusOK, "users_form", pageData{
		Title: "Edit User",
		User:  &current,
		Form: formValues{
			ID:     current.ID,
			Name:   current.Name,
			Emails: joinEmails(current.Emails),
		},
	})
}

func (h *handler) Update(c *echo.Context) error {
	var tag string = "internal.feature.user.handler.Update."

	id := c.Param("id")
	name := c.FormValue("name")
	emails := parseEmails(c.FormValue("emails"))

	data := UpdateData{Name: &name, Emails: emails}

	if err := h.service.Update(requestContext(c), id, data); err != nil {
		logrus.WithFields(logrus.Fields{
			"tag":   tag + "01",
			"error": err.Error(),
		}).Error("failed to update user")

		return c.Render(http.StatusOK, "users_form", pageData{
			Title: "Edit User",
			Form: formValues{
				ID:     id,
				Name:   name,
				Emails: c.FormValue("emails"),
			},
			Error: humanizeError(err),
		})
	}

	return c.Redirect(http.StatusSeeOther, "/users?success=user+updated")
}

func (h *handler) Delete(c *echo.Context) error {
	var tag string = "internal.feature.user.handler.Delete."

	id := c.Param("id")

	if err := h.service.Delete(requestContext(c), id); err != nil {
		logrus.WithFields(logrus.Fields{
			"tag":   tag + "01",
			"error": err.Error(),
		}).Error("failed to delete user")

		return c.Redirect(http.StatusSeeOther, "/users?success=failed+to+delete+user")
	}

	return c.Redirect(http.StatusSeeOther, "/users?success=user+deleted")
}

// parseEmails memecah input textarea/field (dipisah baris baru atau koma)
// menjadi daftar email.
func parseEmails(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ',' || r == ';'
	})

	emails := make([]string, 0, len(fields))

	for _, field := range fields {
		field = strings.TrimSpace(field)

		if field != "" {
			emails = append(emails, field)
		}
	}

	return emails
}

// joinEmails mengubah daftar Email menjadi teks satu email per baris untuk form.
func joinEmails(emails []Email) string {
	values := make([]string, 0, len(emails))

	for _, email := range emails {
		values = append(values, email.Email)
	}

	return strings.Join(values, "\n")
}

// humanizeError memetakan error domain/proses ke pesan yang ramah pengguna.
func humanizeError(err error) string {
	switch {
	case errors.Is(err, ErrNameRequired):
		return "name is required"
	case errors.Is(err, ErrEmailRequired):
		return "at least one valid email is required"
	case errors.Is(err, ErrValidation):
		return "the data sent is invalid"
	case errors.Is(err, ErrDuplicateEmail):
		return "one of the emails is already used"
	case errors.Is(err, ErrNotFound):
		return "user not found"
	case errors.Is(err, ErrUnauthorized):
		return "the application key was rejected by the backend"
	default:
		return "failed to reach the backend, please try again"
	}
}
