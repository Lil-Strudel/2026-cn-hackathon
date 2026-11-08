package httpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReadyz(t *testing.T) {
	tests := []struct {
		name   string
		checks map[string]ReadinessCheck
		want   int
	}{
		{"no checks", nil, http.StatusOK},
		{"all pass", map[string]ReadinessCheck{"redis": func(context.Context) error { return nil }}, http.StatusOK},
		{"one fails", map[string]ReadinessCheck{
			"redis":         func(context.Context) error { return nil },
			"elasticsearch": func(context.Context) error { return errors.New("down") },
		}, http.StatusServiceUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			New(tt.checks).ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/readyz", nil))
			if rec.Code != tt.want {
				t.Fatalf("got %d, want %d: %s", rec.Code, tt.want, rec.Body)
			}
		})
	}
}
