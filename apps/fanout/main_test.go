package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/caarlos0/env/v11"
	"github.com/redis/go-redis/v9"
)

func TestConfigDefaultsMatchViteProxy(t *testing.T) {
	var cfg Config
	if err := env.ParseWithOptions(&cfg, env.Options{Environment: map[string]string{}}); err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 4200 {
		t.Fatalf("port %d, want 4200 so the Vite proxy reaches /events", cfg.Port)
	}
}

func TestHealthz(t *testing.T) {
	mux := newMux(redis.NewClient(&redis.Options{}))

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("got %d, want %d", rec.Code, http.StatusNoContent)
	}
}
