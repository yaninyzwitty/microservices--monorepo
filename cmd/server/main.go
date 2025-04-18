package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/database"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/helpers"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/pkg"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/queue"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var cfg pkg.Config
	file, err := os.Open("config.yaml")
	if err != nil {
		slog.Error("failed to open config.yaml", "error", err)
		os.Exit(1)
	}
	defer file.Close()

	if err := cfg.LoadConfig(file); err != nil {
		slog.Error("failed to load config: ", "error", err)
		os.Exit(1)
	}

	if err := godotenv.Load(); err != nil {
		slog.Error("failed to load .env file", "error", err)
		os.Exit(1)
	}

	password := helpers.GetEnvOrDefault("COCROACH_PASSWORD", "")
	if password == "" {
		slog.Warn("COCROACH_PASSWORD is empty - make sure this is intentional")
		os.Exit(1)
	}

	roachConfig := &database.DBConfig{
		Host:     cfg.Roach.Host,
		Port:     cfg.Roach.Port,
		User:     cfg.Roach.Username,
		Database: cfg.Roach.DbName,
		SSLMode:  cfg.Roach.SSLMode,
		Password: password,
	}

	db, err := database.NewDB(cfg.Roach.MaxRetries, 1*time.Second, roachConfig)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	slog.Info("Connected to CockroachDB successfully")

	// pool := db.Pool()

	pulsarToken := helpers.GetEnvOrDefault("PULSAR_TOKEN", "")
	if password == "" {
		slog.Warn("PULSAR_TOKEN is empty - make sure this is intentional")
		os.Exit(1)
	}

	pulsarCfg := &queue.Config{
		URI:       cfg.Queue.Uri,
		TopicName: cfg.Queue.Topic,
		Token:     pulsarToken,
	}

	// initialize pulsar service
	pulsarService := queue.NewService(pulsarCfg)

	// create pulsar client

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

}
