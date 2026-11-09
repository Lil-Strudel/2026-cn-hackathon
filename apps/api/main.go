package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"

	"github.com/Lil-Strudel/2026-cn-hackathon/internal/config"
	"github.com/Lil-Strudel/2026-cn-hackathon/internal/httpserver"
	"github.com/Lil-Strudel/2026-cn-hackathon/internal/search"
	"github.com/Lil-Strudel/2026-cn-hackathon/internal/telemetry"
	"github.com/Lil-Strudel/2026-cn-hackathon/internal/transcribe"
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
	if err := search.EnsureIndices(ctx, es); err != nil {
		return err
	}
	vocab, err := search.NewVocabulary(es)
	if err != nil {
		return err
	}
	// Seeding only an empty vocabulary keeps a removed default from coming back
	// on the next start, and retries a seed that failed on the first start.
	empty, err := vocab.Empty(ctx)
	if err != nil {
		return err
	}
	if empty {
		if err := vocab.Seed(ctx, search.DefaultTerms); err != nil {
			return err
		}
	}
	history := search.NewCaptions(es)

	var transcriber transcribe.Transcriber = transcribe.NewOpenAI(cfg.OpenAIAPIKey)
	if cfg.OpenAIAPIKey == "" {
		slog.Warn("OPENAI_API_KEY is not set, captions come from the fake transcriber")
		transcriber = transcribe.Fake{}
	}
	audio, err := newAudioHandler(ctx, transcriber, vocab, redisPublisher(rdb), history)
	if err != nil {
		return fmt.Errorf("audio handler: %w", err)
	}
	mux := newMux(rdb, es, vocab, history)
	mux.Handle("GET /api/talks/{talkID}/audio", audio)

	err = httpserver.Run(ctx, fmt.Sprintf(":%d", cfg.Port), mux)
	waitCtx, cancel := context.WithTimeout(context.Background(), drainTimeout+time.Second)
	defer cancel()
	audio.Wait(waitCtx)
	return err
}

func newMux(rdb *redis.Client, es *elasticsearch.TypedClient, vocab *search.Vocabulary, history *search.Captions) *http.ServeMux {
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
	mux.HandleFunc("POST /api/corrections", handleCorrect(vocab))
	mux.HandleFunc("GET /api/vocabulary", handleListTerms(vocab))
	mux.HandleFunc("POST /api/vocabulary", handleAddTerms(vocab))
	mux.HandleFunc("DELETE /api/vocabulary/{term}", handleRemoveTerm(vocab))
	mux.HandleFunc("GET /api/talks/{talkID}/captions", handleCaptionHistory(history))
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
