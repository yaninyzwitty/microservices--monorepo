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
	"github.com/resend/resend-go/v2"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/helpers"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/notification-service/internal/controller"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/notification-service/internal/repository"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/notification-service/service"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/pb"
	pkg "github.com/yaninyzwitty/eccomerce-microservices-backend/pkg/config"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/pkg/queue"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/pkg/snowflake"
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

	resendClient := resend.NewClient(helpers.GetEnvOrDefault("RESEND_API_KEY", ""))
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
	grpcAddress := fmt.Sprintf(":%d", cfg.NotificationServer.Port)
	lis, err := net.Listen("tcp", grpcAddress)
	if err != nil {
		slog.Error("failed to listen", "error", err)
		os.Exit(1)
	}

	notificationRepo := repository.NewNotificationRepository(resendClient)
	notificationController := controller.NewNotificationController(notificationRepo)
	server := grpc.NewServer()
	pb.RegisterNotificationServiceServer(server, notificationController)
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
				if ctx.Err() != nil {
					slog.Info("Receive interrupted due to shutdown, exiting message loop...")
					return
				}

				slog.Error("failed to receive pulsar message", "error", err)

				// Only call Nack if msg is not nil
				if msg != nil {
					pulsarConsumer.Nack(msg)
				}
				continue
			}

			type EventHandler func([]byte) error
			eventHandler := map[string]EventHandler{
				"product.created": func(payload []byte) error {
					var product pb.Product
					if err := protojson.Unmarshal(payload, &product); err != nil {
						return err
					}
					notificationService := service.NewNotificationService(notificationController)
					return notificationService.SendProductNotification(ctx, &product)
				},
				"stock.created": func(payload []byte) error {
					var stockLevel pb.StockLevel
					if err := protojson.Unmarshal(payload, &stockLevel); err != nil {
						return err
					}
					notificationService := service.NewNotificationService(notificationController)
					return notificationService.SendStockNotification(ctx, &stockLevel)
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
				slog.Error("unknown event type", "event_type", eventType)
				pulsarConsumer.Nack(msg)
			}
		}
	}()

	slog.Info("Starting gRPC server", "port", cfg.StockServer.Port)
	if err := server.Serve(lis); err != nil {
		slog.Error("gRPC server encountered an error while serving", "error", err)
		os.Exit(1)
	}

}
