package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"

	"github.com/Lil-Strudel/2026-cn-hackathon/internal/config"
	"github.com/Lil-Strudel/2026-cn-hackathon/internal/httpserver"
	"github.com/Lil-Strudel/2026-cn-hackathon/internal/telemetry"
)

type Config struct {
	Port     int    `env:"PORT" envDefault:"4200"`
	RedisURL string `env:"REDIS_URL" envDefault:"redis://localhost:6379"`
}

func main() {
	if err := run(); err != nil {
		slog.Error("fanout exited", "err", err)
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

	shutdown, err := telemetry.Setup(ctx, "fanout")
	if err != nil {
		return fmt.Errorf("telemetry: %w", err)
	}
	defer func() { _ = shutdown(context.Background()) }()

	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return fmt.Errorf("redis url: %w", err)
	}
	rdb := redis.NewClient(opts)
	defer func() { _ = rdb.Close() }()
	if err := redisotel.InstrumentTracing(rdb); err != nil {
		return err
	}

	mux := httpserver.New(map[string]httpserver.ReadinessCheck{
		"redis": func(ctx context.Context) error { return rdb.Ping(ctx).Err() },
	})

	return httpserver.Run(ctx, fmt.Sprintf(":%d", cfg.Port), mux)
}
