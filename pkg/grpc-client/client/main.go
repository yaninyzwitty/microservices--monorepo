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

	commandAddress := fmt.Sprintf(":%d", cfg.GrpcServer.Port)
	commandConn, err := grpc.Dial(commandAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("failed to create grpc client", "error", err)
		os.Exit(1)
	}
	defer commandConn.Close()

	commandClient := pb.NewProductServiceClient(commandConn)

	// Create product request with proper data
	req := &pb.CreateProductRequest{
		Name:        "kali",
		Description: "jenna",
		Price:       32.44,
		Stock:       100,
		CategoryId:  int64(133592043875221505),
	}

	resp, err := commandClient.CreateProduct(ctx, req)
	if err != nil {
		slog.Error("Failed to create product", "error", err)
		os.Exit(1)
	}

	slog.Info("val", "name", resp.Product.Name)
	slog.Info("val", "id", resp.Product.Id)
	slog.Info("val", "description", resp.Product.Description)

}
