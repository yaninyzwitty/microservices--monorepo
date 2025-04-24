package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/helpers"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/pb"
	pkg "github.com/yaninyzwitty/eccomerce-microservices-backend/pkg/config"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/pkg/database"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/pkg/queue"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/pkg/snowflake"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/stock-service/controller"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/stock-service/repository"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/stock-service/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
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

	slog.Info("Connected to CockroachDB successfully")
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

	pulsarConsumer, err := pulsarService.CreateConsumer(ctx, pulsarClient, cfg.NotificationServer.ConsumerTopic, "stock-service-1")
	if err != nil {
		slog.Error("failed to create pulsar consumer", "error", err)
		os.Exit(1)
	}
	defer pulsarConsumer.Close()

	grpcAddress := fmt.Sprintf(":%d", cfg.StockServer.Port)
	lis, err := net.Listen("tcp", grpcAddress)
	if err != nil {
		slog.Error("failed to listen", "error", err)
		os.Exit(1)
	}
	stockRepo := repository.NewStockRepository(pool)

	stockController := controller.NewStockController(stockRepo)

	server := grpc.NewServer()

	pb.RegisterStockServiceServer(server, stockController)
	reflection.Register(server)

	// Signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		slog.Info("Shutdown signal received", "signal", sig)

		gracefulCtx, gracefulCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer gracefulCancel()

		done := make(chan struct{})
		go func() {
			server.GracefulStop()
			close(done)
		}()

		select {
		case <-done:
			slog.Info("gRPC server shut down gracefully")
		case <-gracefulCtx.Done():
			slog.Warn("Graceful shutdown timeout reached, forcing gRPC server to stop")
			server.Stop()
		}

		cancel() // Cancel other goroutines if needed
	}()

	go func() {
		for {
			msg, err := pulsarConsumer.Receive(ctx)
			if err != nil {
				slog.Error("failed to receive pulsar message", "error", err)
				pulsarConsumer.Nack(msg)
				continue
			} else {

				type EventHandler func([]byte) error

				eventHandler := map[string]EventHandler{
					"product.created": func(payload []byte) error {
						var product pb.Product
						if err := protojson.Unmarshal(payload, &product); err != nil {
							return err
						}
						stockService := service.NewStockService(stockController)
						return stockService.CreateStockProduct(ctx, &product)
					},
				}

				eventType := strings.Split(msg.Key(), ":")[0]

				if handler, ok := eventHandler[eventType]; ok {
					if err := handler(msg.Payload()); err != nil {
						slog.Error("handler error", "error", err)
						pulsarConsumer.Nack(msg)
					} else {
						pulsarConsumer.Ack(msg)
					}
				} else {
					pulsarConsumer.Nack(msg)
				}

			}

		}

	}()
	slog.Info("Starting gRPC server", "port", cfg.StockServer.Port)
	if err := server.Serve(lis); err != nil {
		slog.Error("gRPC server encountered an error while serving", "error", err)
		os.Exit(1)
	}

}
