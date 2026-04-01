package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/sudores/twt-test-task/pkg/config"
	"github.com/sudores/twt-test-task/pkg/middleware"
	"github.com/sudores/twt-test-task/pkg/proxy"
)

func main() {
	cfgPath := getEnv("CONFIG_FILE", "config.yaml")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		l := zerolog.New(os.Stderr)
		l.Fatal().Err(err).Str("path", cfgPath).Msg("failed to load config")
	}

	logger := buildLogger(cfg.Log)

	p, err := proxy.New(proxy.Config{
		UpstreamURL: cfg.Proxy.UpstreamURL,
		Timeout:     cfg.Proxy.Timeout.Duration,
	}, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create proxy")
	}

	handler := p.Handler()
	if cfg.RateLimit.Enabled {
		rl := middleware.RateLimit(cfg.RateLimit.RequestsPerSecond, cfg.RateLimit.Burst, logger)
		handler = rl(handler)
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout.Duration,
		WriteTimeout: cfg.Server.WriteTimeout.Duration,
		IdleTimeout:  cfg.Server.IdleTimeout.Duration,
	}

	go func() {
		logger.Info().
			Int("port", cfg.Server.Port).
			Str("upstream", cfg.Proxy.UpstreamURL).
			Bool("rate_limit", cfg.RateLimit.Enabled).
			Msg("starting proxy")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal().Err(err).Msg("server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("forced shutdown")
	}
}

func buildLogger(cfg config.LogConfig) zerolog.Logger {
	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}
	if cfg.Pretty {
		return zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).
			With().Timestamp().Logger().Level(level)
	}
	return zerolog.New(os.Stdout).With().Timestamp().Logger().Level(level)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
