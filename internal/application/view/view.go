// Package view menyediakan layout global dan renderer SSR untuk seluruh
// feature. Layout hidup di lapisan application (global), sedangkan setiap
// feature hanya menyumbang blok "content" (dan opsional "title").
//
// Renderer meng-compose layout global + content tiap halaman menjadi satu
// template tree per halaman pada saat startup, sehingga tidak ada parsing
// saat request dan template yang rusak langsung ketahuan ketika boot.
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

// StaticFS mengembalikan filesystem berisi aset statis (CSS, dll.) yang
// di-root pada folder "static", siap dilayani lewat echo.StaticFS.
func StaticFS() fs.FS {
	return echo.MustSubFS(staticFS, "static")
}

// layoutName adalah nama template terluar yang dieksekusi untuk setiap halaman.
const layoutName = "base"

// ErrorPage adalah nama halaman generik lintas-feature untuk menampilkan error
// (404, 405, 500, dll.) sebagai HTML ber-layout.
const ErrorPage = "error"

// contentGlob adalah pola file content di dalam FS milik feature.
const contentGlob = "templates/*.html"

// globalPageGlob adalah pola halaman generik milik lapisan application (view).
const globalPageGlob = "pages/*.html"

// Renderer mengimplementasikan echo.Renderer. Ia memetakan nama halaman
// (mis. "users_index") ke tree yang sudah tergabung dengan layout global.
type Renderer struct {
	pages map[string]*template.Template
}

// NewRenderer membangun renderer dari kumpulan FS content milik feature.
//
// Setiap file content mendefinisikan halaman lewat:
//
//	{{ define "content" }} ... {{ end }}
//	{{ define "title" }} ... {{ end }}   (opsional)
//
// dan didaftarkan dengan nama = base name file tanpa ekstensi
// (mis. templates/users_index.html -> "users_index").
func NewRenderer(featureViews ...embed.FS) (*Renderer, error) {
	base, err := template.New(layoutName).ParseFS(layoutFS, "layouts/*.html")

	if err != nil {
		return nil, fmt.Errorf("view: failed to parse global layout: %w", err)
	}

	renderer := &Renderer{pages: make(map[string]*template.Template)}

	// Halaman generik milik lapisan application (mis. "error").
	if err := renderer.register(base, pageFS, globalPageGlob); err != nil {
		return nil, err
	}

	// Content milik tiap feature.
	for _, fsys := range featureViews {
		if err := renderer.register(base, fsys, contentGlob); err != nil {
			return nil, err
		}
	}

	return renderer, nil
}

// register memindai file content pada fsys sesuai glob, meng-clone layout
// global untuk tiap halaman, lalu memetakannya dengan nama = base name file.
func (r *Renderer) register(base *template.Template, fsys fs.FS, glob string) error {
	entries, err := fs.Glob(fsys, glob)

	if err != nil {
		return fmt.Errorf("view: failed to scan content templates: %w", err)
	}

	for _, entry := range entries {
		name := strings.TrimSuffix(filepath.Base(entry), filepath.Ext(entry))

		// Nama halaman harus unik lintas feature dan halaman global. Tabrakan
		// dibuat gagal saat startup, bukan menimpa diam-diam.
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

// Render mengeksekusi layout global untuk halaman bernama name.
func (r *Renderer) Render(c *echo.Context, w io.Writer, name string, data any) error {
	page, ok := r.pages[name]

	if !ok {
		return fmt.Errorf("view: template %q is not registered", name)
	}

	return page.ExecuteTemplate(w, layoutName, data)
}

// errorPageData adalah data yang dikirim ke halaman error.
type errorPageData struct {
	Title   string
	Code    int
	Message string
}

// HTTPErrorHandler adalah echo.HTTPErrorHandler yang merender error sebagai
// HALAMAN HTML ber-layout (bukan JSON), sesuai sifat aplikasi SSR.
//
// Alur: abaikan bila response sudah ter-commit; tentukan status via
// echo.StatusCode (fallback 500); render halaman "error"; bila render gagal
// (mis. HEAD request atau template bermasalah), sediakan fallback teks agar
// klien tetap menerima respons dengan status yang benar.
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

	// HEAD tidak boleh punya body.
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

	// Fallback: bila render halaman error gagal, tetap kirim respons teks.
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
