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

var (
	ErrNotFound       = errors.New("user not found")
	ErrDuplicateEmail = errors.New("duplicate email")
	ErrValidation     = errors.New("validation failed")
	ErrUnauthorized   = errors.New("backend rejected the application key")
	ErrBackend        = errors.New("backend request failed")
)

type InterfaceRepository interface {
	Create(ctx context.Context, data CreateData) (*User, error)
	List(ctx context.Context, params ListParams) (*ListResult, error)
	Update(ctx context.Context, id string, data UpdateData) error
	Delete(ctx context.Context, id string) error
}

type RepositoryConfig struct {
	BaseURL          string
	AppKey           string
	Timeout          int
	MaxResponseBytes int
	MaxIdleConns     int
	MaxConnsPerHost  int
	RetryCount       int
}

type Repository struct {
	client *resty.Client
}

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

	if cfg.AppKey != "" {
		client.SetHeader("X-App-Key", cfg.AppKey)
	}

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
