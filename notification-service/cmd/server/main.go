package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/helpers"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/pb"
	pkg "github.com/yaninyzwitty/eccomerce-microservices-backend/pkg/config"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/pkg/queue"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/pkg/snowflake"
	"google.golang.org/protobuf/encoding/protojson"
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

	// resendClient := resend.NewClient(helpers.GetEnvOrDefault("RESEND_API_KEY", ""))
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

	pulsarConsumer, err := pulsarService.CreateConsumer(ctx, pulsarClient, cfg.NotificationServer.ConsumerTopic, "notification-service-1")
	if err != nil {
		slog.Error("failed to create pulsar consumer", "error", err)
		os.Exit(1)
	}
	defer pulsarConsumer.Close()

	for {
		msg, err := pulsarConsumer.Receive(ctx)
		if err != nil {
			log.Fatal(err)
		} else {
			log.Printf("Received message : %s", string(msg.Payload()))

			var product pb.Product
			if err := protojson.Unmarshal(msg.Payload(), &product); err != nil {
				log.Fatal(err)
			}

			slog.Info("productId", "val", product.Id)
			slog.Info("name", "val", product.Name)
			slog.Info("description", "val", product.Description)
			slog.Info("price", "val", product.Price)
		}

		pulsarConsumer.Ack(msg)
	}

}
