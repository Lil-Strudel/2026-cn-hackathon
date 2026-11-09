package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/caarlos0/env/v11"
	"github.com/elastic/go-elasticsearch/v9"
	"github.com/redis/go-redis/v9"
)

func TestConfigDefaultsMatchViteProxy(t *testing.T) {
	var cfg Config
	if err := env.ParseWithOptions(&cfg, env.Options{Environment: map[string]string{}}); err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 4100 {
		t.Fatalf("port %d, want 4100 so the Vite proxy reaches /api", cfg.Port)
	}
}

func TestNewRedisRejectsBadURL(t *testing.T) {
	if _, err := newRedis("not a url"); err == nil {
		t.Fatal("want an error for a malformed redis url")
	}
}

func TestHealthz(t *testing.T) {
	es, err := elasticsearch.NewTyped()
	if err != nil {
		t.Fatal(err)
	}
	mux := newMux(redis.NewClient(&redis.Options{}), es)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("got %d, want %d", rec.Code, http.StatusNoContent)
	}
}
