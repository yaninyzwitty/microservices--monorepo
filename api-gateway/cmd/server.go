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

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/graph"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/pb"
	pkg "github.com/yaninyzwitty/eccomerce-microservices-backend/pkg/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	var cfg pkg.Config

	file, err := os.Open("config.yaml")
	if err != nil {
		slog.Error("failed to open file, config.yaml", "error", err)
		os.Exit(1)
	}

	defer file.Close()

	if err := cfg.LoadConfig(file); err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	mux := chi.NewRouter()
	mux.Use(middleware.Logger)

	productGrpcClientAddress := fmt.Sprintf(":%d", cfg.GrpcServer.Port)
	orderGrpcClientAddress := fmt.Sprintf(":%d", cfg.OrderServer.Port)

	// Corrected gRPC Dial method
	productGrpcGrpcConn, err := grpc.NewClient(productGrpcClientAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("failed to create grpc client", "error", err)
		os.Exit(1)
	}
	defer productGrpcGrpcConn.Close()

	orderGrpcConn, err := grpc.NewClient(orderGrpcClientAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("failed to create grpc client", "error", err)
		os.Exit(1)
	}
	defer orderGrpcConn.Close()

	productClient := pb.NewProductServiceClient(productGrpcGrpcConn)
	orderClient := pb.NewOrderServiceClient(orderGrpcConn)
	srv := handler.New(graph.NewExecutableSchema(graph.Config{Resolvers: &graph.Resolver{
		ProductClient: productClient,
		OrderClient:   orderClient,
	}}))

	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})

	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))

	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	mux.Handle("/", playground.Handler("GraphQL playground", "/query"))
	mux.Handle("/query", srv)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.QqlgenServer.Port),
		Handler: mux,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		slog.Info("Starting GraphQL server", "port", cfg.QqlgenServer.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server stopped with error", "error", err)
		}
	}()

	<-stop
	slog.Info("Shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Graceful shutdown failed", "error", err)
	} else {
		slog.Info("Graceful shutdown completed")
	}

	wg.Wait()
	slog.Info("Server exited properly")

}
