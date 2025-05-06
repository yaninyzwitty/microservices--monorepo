package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/yaninyzwitty/eccomerce-microservices-backend/pb"
	pkg "github.com/yaninyzwitty/eccomerce-microservices-backend/pkg/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

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

	commandAddress := fmt.Sprintf(":%d", cfg.OrderServer.Port)
	commandConn, err := grpc.NewClient(commandAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("failed to create grpc client", "error", err)
		os.Exit(1)
	}
	defer commandConn.Close()

	commandClient := pb.NewOrderServiceClient(commandConn)

	// Create product request with proper data
	req := &pb.GetOrderRequest{
		OrderId: int64(135469340458799105),
	}

	resp, err := commandClient.GetOrder(ctx, req)
	if err != nil {
		slog.Error("Failed to create user", "error", err)
		os.Exit(1)
	}

	logger.Info("listing orders", slog.Any("order", resp.Order))

}
