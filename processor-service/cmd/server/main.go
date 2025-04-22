package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/helpers"
	pkg "github.com/yaninyzwitty/eccomerce-microservices-backend/pkg/config"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/pkg/database"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/pkg/queue"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/pkg/snowflake"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/processor-service/internal/processor"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/processor-service/internal/publisher"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/processor-service/outbox"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var cfg pkg.Config
	file, err := os.Open("config.yaml")
	if err != nil {
		slog.Error("failed to open config.yaml", "error", err)
		os.Exit(1)
	}
	defer file.Close()

	if err := cfg.LoadConfig(file); err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	if err := godotenv.Load(); err != nil {
		slog.Error("failed to load .env file", "error", err)
		os.Exit(1)
	}

	if err := snowflake.InitSonyFlake(); err != nil {
		slog.Error("failed to initialize snowflake", "error", err)
		os.Exit(1)
	}

	password := helpers.GetEnvOrDefault("COCKROACH_PASSWORD", "")
	if password == "" {
		slog.Warn("COCKROACH_PASSWORD is empty - make sure this is intentional")
	}

	roachConfig := &database.DBConfig{
		Host:     cfg.Roach.Host,
		Port:     cfg.Roach.Port,
		User:     cfg.Roach.Username,
		Password: password,
		Database: cfg.Roach.DbName,
		SSLMode:  cfg.Roach.SSLMode,
	}

	db, err := database.NewDB(cfg.Roach.MaxRetries, 1*time.Second, roachConfig)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	slog.Info("Connected to CockroachDB", "service", "processor")
	pool := db.Pool()

	pulsarToken := helpers.GetEnvOrDefault("PULSAR_TOKEN", "")
	if pulsarToken == "" {
		slog.Warn("PULSAR_TOKEN is empty - make sure this is intentional")
	}

	pulsarCfg := &queue.Config{
		URI:       cfg.Queue.Uri,
		TopicName: cfg.Queue.Topic,
		Token:     pulsarToken,
	}
	pulsarService := queue.NewService(pulsarCfg)

	pulsarClient, err := pulsarService.CreateConnection(ctx)
	if err != nil {
		slog.Error("failed to create pulsar client", "error", err)
		os.Exit(1)
	}
	defer pulsarClient.Close()

	pulsarProducer, err := pulsarService.CreateProducer(ctx, pulsarClient)
	if err != nil {
		slog.Error("failed to create pulsar producer", "error", err)
		os.Exit(1)
	}
	defer pulsarProducer.Close()

	outboxRepo := outbox.NewOutboxRepository(pool)
	pulsarProducerService := publisher.NewPulsarProducer(pulsarProducer)
	pm := processor.NewProcessMessage(*outboxRepo, pulsarProducerService)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Processer.Port),
		Handler: mux,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup

	// Start HTTP server
	wg.Add(1)
	go func() {
		defer wg.Done()
		slog.Info("Starting HTTP server", "port", cfg.Processer.Port, "service", "processor")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server failed", "error", err)
		}
	}()

	// Start outbox processor
	wg.Add(1)
	go func() {
		defer wg.Done()

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := pm.ProcessMessages(ctx); err != nil {
					slog.Error("error processing messages", "error", err)
					// Optional: add retry/backoff or metrics
				}
			case <-ctx.Done():
				slog.Info("Stopping outbox processor")
				return
			}
		}
	}()

	// Wait for shutdown signal
	<-stop
	slog.Info("Shutdown signal received")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Graceful shutdown failed", "error", err)
	} else {
		slog.Info("Graceful shutdown completed")
	}

	wg.Wait()
	slog.Info("Server exited properly", "service", "processor")
}
