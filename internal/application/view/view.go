package view

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/sirupsen/logrus"
)

//go:embed layouts/*.html
var layoutFS embed.FS

//go:embed pages/*.html
var pageFS embed.FS

//go:embed static/*
var staticFS embed.FS

func StaticFS() fs.FS {
	return echo.MustSubFS(staticFS, "static")
}

const layoutName = "base"

const ErrorPage = "error"

const contentGlob = "templates/*.html"

const globalPageGlob = "pages/*.html"

type Renderer struct {
	pages map[string]*template.Template
}

func NewRenderer(featureViews ...embed.FS) (*Renderer, error) {
	base, err := template.New(layoutName).ParseFS(layoutFS, "layouts/*.html")

	if err != nil {
		return nil, fmt.Errorf("view: failed to parse global layout: %w", err)
	}

	renderer := &Renderer{pages: make(map[string]*template.Template)}

	if err := renderer.register(base, pageFS, globalPageGlob); err != nil {
		return nil, err
	}

	for _, fsys := range featureViews {
		if err := renderer.register(base, fsys, contentGlob); err != nil {
			return nil, err
		}
	}

	return renderer, nil
}

func (r *Renderer) register(base *template.Template, fsys fs.FS, glob string) error {
	entries, err := fs.Glob(fsys, glob)

	if err != nil {
		return fmt.Errorf("view: failed to scan content templates: %w", err)
	}

	for _, entry := range entries {
		name := strings.TrimSuffix(filepath.Base(entry), filepath.Ext(entry))

		if _, exists := r.pages[name]; exists {
			return fmt.Errorf("view: duplicate page name %q (template %q clashes with an already registered page)", name, entry)
		}

		raw, err := fs.ReadFile(fsys, entry)

		if err != nil {
			return fmt.Errorf("view: failed to read content template %q: %w", entry, err)
		}

		cloned, err := base.Clone()

		if err != nil {
			return fmt.Errorf("view: failed to clone layout for %q: %w", name, err)
		}

		if _, err := cloned.Parse(string(raw)); err != nil {
			return fmt.Errorf("view: failed to parse content template %q: %w", name, err)
		}

		r.pages[name] = cloned
	}

	return nil
}

func (r *Renderer) Render(c *echo.Context, w io.Writer, name string, data any) error {
	page, ok := r.pages[name]

	if !ok {
		return fmt.Errorf("view: template %q is not registered", name)
	}

	return page.ExecuteTemplate(w, layoutName, data)
}

type errorPageData struct {
	Title   string
	Code    int
	Message string
}

func (r *Renderer) HTTPErrorHandler(c *echo.Context, err error) {
	var tag string = "internal.application.view.HTTPErrorHandler."

	if resp, uErr := echo.UnwrapResponse(c.Response()); uErr == nil && resp.Committed {
		return
	}

	code := http.StatusInternalServerError

	if status := echo.StatusCode(err); status != 0 {
		code = status
	}

	message := http.StatusText(code)

	logrus.WithFields(logrus.Fields{
		"tag":        tag + "01",
		"statusCode": code,
		"error":      err.Error(),
	}).Error("request failed")

	if c.Request().Method == http.MethodHead {
		if nErr := c.NoContent(code); nErr != nil {
			logrus.WithFields(logrus.Fields{
				"tag":   tag + "02",
				"error": nErr.Error(),
			}).Error("failed to write no-content error response")
		}

		return
	}

	renderErr := c.Render(code, ErrorPage, errorPageData{
		Title:   message,
		Code:    code,
		Message: message,
	})

	if renderErr == nil {
		return
	}

	logrus.WithFields(logrus.Fields{
		"tag":   tag + "03",
		"error": renderErr.Error(),
	}).Error("failed to render error page, falling back to plain text")

	if sErr := c.String(code, message); sErr != nil {
		logrus.WithFields(logrus.Fields{
			"tag":   tag + "04",
			"error": sErr.Error(),
		}).Error("failed to write fallback error response")
	}
}
