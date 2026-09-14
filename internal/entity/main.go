package entity

import (
	"context"
	"errors"
	"regexp"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// RequestIDHeader adalah nama header korelasi yang dipakai lintas layanan
// (frontend -> backend), selaras dengan gomiddleware.EchoSetRequestID.
const RequestIDHeader = "X-Request-ID"

// requestIDKey adalah key privat untuk menyimpan request ID di context.
// Tipe khusus mencegah tabrakan dengan key context milik paket lain.
type requestIDKey struct{}

// ContextWithRequestID menautkan request ID ke context agar bisa diteruskan
// dari handler ke lapisan repository tanpa membocorkan detail framework.
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

// RequestIDFromContext mengambil request ID dari context; string kosong bila
// tidak ada.
func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey{}).(string); ok {
		return id
	}

	return ""
}

type MainResponse[T any] struct {
	Status  bool           `json:"status"`
	Message string         `json:"message"`
	Data    T              `json:"data"`
	Meta    *PaginatorMeta `json:"meta"`
	Error   []any          `json:"error"`
}

type PaginatorMeta struct {
	Total *int64 `json:"total"`
	Page  int    `json:"page"`
	Limit int    `json:"limit"`
}

func BlacklistValidation(field string) validation.RuleFunc {
	return func(value interface{}) error {
		val, ok := value.(string)

		if !ok {
			return errors.New("must be a valid string")
		}

		if val == "" {
			return nil
		}

		match, _ := regexp.MatchString(`^[^'"\[\]<>\{\}]+$`, val)

		if !match {
			return errors.New("must contains safe characters")
		}

		return nil
	}
}
