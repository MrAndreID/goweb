package user

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/MrAndreID/goweb/internal/entity"

	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"
)

// Error domain yang bisa dikenali lapisan atas tanpa perlu tahu detail HTTP.
var (
	ErrNotFound       = errors.New("user not found")
	ErrDuplicateEmail = errors.New("duplicate email")
	ErrValidation     = errors.New("validation failed")
	ErrUnauthorized   = errors.New("backend rejected the application key")
	ErrBackend        = errors.New("backend request failed")
)

// InterfaceRepository adalah port keluar (outbound) menuju backend.
// Hanya implementasi di file ini yang boleh mengetahui detail HTTP,
// base URL, dan header X-App-Key.
type InterfaceRepository interface {
	Create(ctx context.Context, data CreateData) (*User, error)
	List(ctx context.Context, params ListParams) (*ListResult, error)
	Update(ctx context.Context, id string, data UpdateData) error
	Delete(ctx context.Context, id string) error
}

// RepositoryConfig memuat seluruh opsi koneksi keluar menuju backend.
// Dikumpulkan dalam satu struct agar penambahan opsi tidak mengubah tanda
// tangan konstruktor dan pemanggilan tetap mudah dibaca.
type RepositoryConfig struct {
	// BaseURL adalah alamat dasar backend, mis. http://127.0.0.1:10001.
	BaseURL string
	// AppKey dikirim sebagai header X-App-Key. Bila kosong, header TIDAK
	// dikirim sama sekali (agar salah konfigurasi mudah terdeteksi di backend).
	AppKey string
	// Timeout adalah batas total per request (detik). <= 0 berarti tanpa batas.
	Timeout int
	// MaxResponseBytes membatasi ukuran body response yang dibaca ke memori.
	// Ini proteksi utama terhadap out-of-memory bila backend mengirim body
	// sangat besar. <= 0 berarti tidak dibatasi (tidak disarankan).
	MaxResponseBytes int
	// MaxIdleConns membatasi jumlah koneksi idle yang di-pool lintas host.
	MaxIdleConns int
	// MaxConnsPerHost membatasi total koneksi (aktif + idle) per host untuk
	// mencegah ledakan goroutine/koneksi saat backend lambat.
	MaxConnsPerHost int
	// RetryCount adalah jumlah percobaan ulang untuk kegagalan transient.
	RetryCount int
}

// Repository adalah adapter keluar berbasis resty menuju backend GoAPI.
type Repository struct {
	client *resty.Client
}

// NewRepository membangun repository yang di-harden untuk pemakaian jangka
// panjang: timeout granular pada transport, batas ukuran body (anti-OOM),
// batas connection pool, dan retry transient. Seluruh koneksi keluar
// workspace terpusat di sini.
//
// Catatan resource: resty membaca lalu MENUTUP response body secara internal
// (defer close), sehingga pemanggil tidak perlu menutup body manual dan tidak
// ada koneksi yang menggantung selama transport dikonfigurasi dengan timeout.
func NewRepository(cfg RepositoryConfig) *Repository {
	timeout := time.Duration(cfg.Timeout) * time.Second

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          cfg.MaxIdleConns,
		MaxIdleConnsPerHost:   cfg.MaxIdleConns,
		MaxConnsPerHost:       cfg.MaxConnsPerHost,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: timeout,
		ExpectContinueTimeout: 1 * time.Second,
	}

	client := resty.New().
		SetBaseURL(cfg.BaseURL).
		SetTransport(transport).
		SetHeader("Accept", "application/json").
		SetHeader("Content-Type", "application/json").
		SetTimeout(timeout).
		SetResponseBodyLimit(cfg.MaxResponseBytes).
		SetRetryCount(cfg.RetryCount).
		SetRetryWaitTime(200 * time.Millisecond).
		SetRetryMaxWaitTime(2 * time.Second)

	// Header X-App-Key hanya dikirim bila terkonfigurasi. Mengirim header
	// kosong hanya membuat backend menolak dengan pesan yang membingungkan.
	if cfg.AppKey != "" {
		client.SetHeader("X-App-Key", cfg.AppKey)
	}

	// Teruskan request ID (dari context) ke backend untuk korelasi log
	// lintas layanan. Dipasang sekali di sini agar seluruh method tidak perlu
	// menyetel header secara manual.
	client.OnBeforeRequest(func(_ *resty.Client, req *resty.Request) error {
		if requestID := entity.RequestIDFromContext(req.Context()); requestID != "" {
			req.SetHeader(entity.RequestIDHeader, requestID)
		}

		return nil
	})

	return &Repository{client: client}
}

func (r *Repository) Create(ctx context.Context, data CreateData) (*User, error) {
	var (
		tag    string = "internal.feature.user.repository.Create."
		result entity.MainResponse[*User]
	)

	resp, err := r.client.R().
		SetContext(ctx).
		SetBody(data).
		SetResult(&result).
		Post("/api/v1/user")

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"tag":   tag + "01",
			"error": err.Error(),
		}).Error("failed to call create user endpoint")

		return nil, ErrBackend
	}

	if err := mapStatusError(resp.StatusCode()); err != nil {
		logrus.WithFields(logrus.Fields{
			"tag":        tag + "02",
			"statusCode": resp.StatusCode(),
		}).Error("backend returned an error on create user")

		return nil, err
	}

	return result.Data, nil
}

func (r *Repository) List(ctx context.Context, params ListParams) (*ListResult, error) {
	var (
		tag    string = "internal.feature.user.repository.List."
		result entity.MainResponse[[]User]
	)

	resp, err := r.client.R().
		SetContext(ctx).
		SetQueryParams(buildListQuery(params)).
		SetResult(&result).
		Get("/api/v1/user")

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"tag":   tag + "01",
			"error": err.Error(),
		}).Error("failed to call list user endpoint")

		return nil, ErrBackend
	}

	if err := mapStatusError(resp.StatusCode()); err != nil {
		logrus.WithFields(logrus.Fields{
			"tag":        tag + "02",
			"statusCode": resp.StatusCode(),
		}).Error("backend returned an error on list user")

		return nil, err
	}

	listResult := &ListResult{Users: result.Data}

	if result.Meta != nil {
		listResult.Paginator = Paginator{
			Total: result.Meta.Total,
			Page:  result.Meta.Page,
			Limit: result.Meta.Limit,
		}
	}

	return listResult, nil
}

func (r *Repository) Update(ctx context.Context, id string, data UpdateData) error {
	var tag string = "internal.feature.user.repository.Update."

	resp, err := r.client.R().
		SetContext(ctx).
		SetPathParam("id", id).
		SetBody(data).
		Patch("/api/v1/user/{id}")

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"tag":   tag + "01",
			"error": err.Error(),
		}).Error("failed to call update user endpoint")

		return ErrBackend
	}

	if err := mapStatusError(resp.StatusCode()); err != nil {
		logrus.WithFields(logrus.Fields{
			"tag":        tag + "02",
			"statusCode": resp.StatusCode(),
		}).Error("backend returned an error on update user")

		return err
	}

	return nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	var tag string = "internal.feature.user.repository.Delete."

	resp, err := r.client.R().
		SetContext(ctx).
		SetPathParam("id", id).
		Delete("/api/v1/user/{id}")

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"tag":   tag + "01",
			"error": err.Error(),
		}).Error("failed to call delete user endpoint")

		return ErrBackend
	}

	if err := mapStatusError(resp.StatusCode()); err != nil {
		logrus.WithFields(logrus.Fields{
			"tag":        tag + "02",
			"statusCode": resp.StatusCode(),
		}).Error("backend returned an error on delete user")

		return err
	}

	return nil
}

// buildListQuery hanya menyertakan query param yang terisi agar backend
// memakai nilai default efektifnya (page=1, limit=10).
func buildListQuery(params ListParams) map[string]string {
	query := make(map[string]string)

	for key, value := range map[string]string{
		"page":                  params.Page,
		"limit":                 params.Limit,
		"orderBy":               params.OrderBy,
		"sortBy":                params.SortBy,
		"search":                params.Search,
		"disableCalculateTotal": params.DisableCalculateTotal,
		"id":                    params.ID,
	} {
		if value != "" {
			query[key] = value
		}
	}

	return query
}

// mapStatusError menerjemahkan status HTTP backend menjadi error domain.
func mapStatusError(statusCode int) error {
	switch {
	case statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices:
		return nil
	case statusCode == http.StatusBadRequest:
		return ErrValidation
	case statusCode == http.StatusUnauthorized, statusCode == http.StatusForbidden:
		return ErrUnauthorized
	case statusCode == http.StatusConflict:
		return ErrDuplicateEmail
	case statusCode == http.StatusNotFound:
		return ErrNotFound
	default:
		return ErrBackend
	}
}
