package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"

	"github.com/Lil-Strudel/2026-cn-hackathon/internal/config"
	"github.com/Lil-Strudel/2026-cn-hackathon/internal/httpserver"
	"github.com/Lil-Strudel/2026-cn-hackathon/internal/telemetry"
)

type Config struct {
	Port                  int    `env:"PORT" envDefault:"4100"`
	RedisURL              string `env:"REDIS_URL" envDefault:"redis://localhost:6379"`
	ElasticsearchURL      string `env:"ELASTICSEARCH_URL" envDefault:"http://localhost:9200"`
	ElasticsearchUsername string `env:"ELASTICSEARCH_USERNAME"`
	ElasticsearchPassword string `env:"ELASTICSEARCH_PASSWORD"`
	OpenAIAPIKey          string `env:"OPENAI_API_KEY"`
}

func main() {
	if err := run(); err != nil {
		slog.Error("api exited", "err", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load[Config]()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	shutdown, err := telemetry.Setup(ctx, "api")
	if err != nil {
		return fmt.Errorf("telemetry: %w", err)
	}
	defer func() { _ = shutdown(context.Background()) }()

	rdb, err := newRedis(cfg.RedisURL)
	if err != nil {
		return err
	}
	defer func() { _ = rdb.Close() }()

	es, err := elasticsearch.NewTyped(
		elasticsearch.WithAddresses(cfg.ElasticsearchURL),
		elasticsearch.WithBasicAuth(cfg.ElasticsearchUsername, cfg.ElasticsearchPassword),
		elasticsearch.WithInstrumentation(elasticsearch.NewOpenTelemetryInstrumentation(nil, false)),
	)
	if err != nil {
		return fmt.Errorf("elasticsearch: %w", err)
	}

	return httpserver.Run(ctx, fmt.Sprintf(":%d", cfg.Port), newMux(rdb, es))
}

func newMux(rdb *redis.Client, es *elasticsearch.TypedClient) *http.ServeMux {
	mux := httpserver.New(map[string]httpserver.ReadinessCheck{
		"redis": func(ctx context.Context) error { return rdb.Ping(ctx).Err() },
		"elasticsearch": func(ctx context.Context) error {
			ok, err := es.Ping().Do(ctx)
			if err == nil && !ok {
				err = fmt.Errorf("ping failed")
			}
			return err
		},
	})
	mux.HandleFunc("GET /api/hello", func(w http.ResponseWriter, r *http.Request) {
		info, err := es.Info().Do(r.Context())
		if err != nil {
			slog.ErrorContext(r.Context(), "elasticsearch info", "err", err)
			http.Error(w, "elasticsearch unavailable", http.StatusBadGateway)
			return
		}
		pong, err := rdb.Ping(r.Context()).Result()
		if err != nil {
			slog.ErrorContext(r.Context(), "redis ping", "err", err)
			http.Error(w, "redis unavailable", http.StatusBadGateway)
			return
		}
		httpserver.WriteJSON(w, http.StatusOK, map[string]string{
			"message":       "hello from api",
			"elasticsearch": info.Version.Int,
			"redis":         pong,
		})
	})

	return mux
}

func newRedis(url string) (*redis.Client, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("redis url: %w", err)
	}
	rdb := redis.NewClient(opts)
	if err := redisotel.InstrumentTracing(rdb); err != nil {
		return nil, err
	}
	if err := redisotel.InstrumentMetrics(rdb); err != nil {
		return nil, err
	}
	return rdb, nil
}
