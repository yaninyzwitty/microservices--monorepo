package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/pb"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/pkg/snowflake"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type StockRepository struct {
	pool *pgxpool.Pool
}

const (
	stockCreatedEvent = "stock.created"
	orderCreatedEvent = "order.created"
)

func NewStockRepository(pool *pgxpool.Pool) *StockRepository {
	return &StockRepository{
		pool: pool,
	}
}

func (r *StockRepository) AddStockProduct(ctx context.Context, stockLevel *pb.StockLevel) (*pb.StockLevel, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		slog.Error("Failed to begin transaction", "error", err)
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	slog.Info("Transaction started")

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		} else if err != nil {
			slog.Info("Rolling back transaction due to error")
			_ = tx.Rollback(ctx)
		}
	}()

	var productExists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM products WHERE id = $1)`
	if err = tx.QueryRow(ctx, checkQuery, stockLevel.ProductId).Scan(&productExists); err != nil {
		slog.Error("Product existence check failed", "error", err)
		return nil, fmt.Errorf("check product: %w", err)
	}
	if !productExists {
		slog.Warn("Product does not exist", "productId", stockLevel.ProductId)
		return nil, fmt.Errorf("product with id %d does not exist", stockLevel.ProductId)
	}

	insertStockQuery := `INSERT INTO stock_levels (product_id, warehouse_id, quantity) VALUES ($1, $2, $3)`
	if _, err = tx.Exec(ctx, insertStockQuery, stockLevel.ProductId, stockLevel.WarehouseId, stockLevel.Quantity); err != nil {
		slog.Error("Insert stock failed", "error", err)
		return nil, fmt.Errorf("insert stock: %w", err)
	}

	stockLevel.CreatedAt = timestamppb.Now()

	outboxID, err := snowflake.GenerateID()
	if err != nil {
		slog.Error("Outbox ID generation failed", "error", err)
		return nil, fmt.Errorf("generate outbox id: %w", err)
	}

	payload, err := protojson.Marshal(stockLevel)
	if err != nil {
		slog.Error("Failed to marshal stock level", "error", err)
		return nil, fmt.Errorf("marshal stock level: %w", err)
	}

	insertOutboxQuery := `INSERT INTO outbox (id, event_type, payload) VALUES ($1, $2, $3)`
	if _, err = tx.Exec(ctx, insertOutboxQuery, outboxID, stockCreatedEvent, payload); err != nil {
		slog.Error("Insert outbox event failed", "error", err)
		return nil, fmt.Errorf("insert outbox: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		slog.Error("Transaction commit failed", "error", err)
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	slog.Info("Transaction committed", "productId", stockLevel.ProductId)
	return stockLevel, nil
}

func (r *StockRepository) AddWarehouse(ctx context.Context, warehouse *pb.Warehouse) (*pb.Warehouse, error) {
	insertWarehouseQuery := `
	INSERT INTO warehouses (id, name, location)
	VALUES ($1, $2, $3) returning id, name, location, created_at
`

	var warehouseRes pb.Warehouse
	var createdAt time.Time
	err := r.pool.QueryRow(ctx, insertWarehouseQuery,
		warehouse.Id, warehouse.Name, warehouse.Location).Scan(&warehouseRes.Id, &warehouse.Name, &warehouse.Location, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert warehouse: %w", err)
	}
	warehouseRes.CreatedAt = timestamppb.New(createdAt)
	return &warehouseRes, nil

}

func (r *StockRepository) ReserveStockItem(ctx context.Context, order *pb.Order) (*pb.Order, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	updateQuery := `
	UPDATE stock_levels
	SET quantity = quantity - $1
	WHERE product_id = $2 AND quantity >= $1
	RETURNING quantity`
	for _, item := range order.Items {
		var remaining int
		err := tx.QueryRow(ctx, updateQuery, item.Quantity, item.ProductId).Scan(&remaining)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("insufficient stock for product_id=%d ", item.ProductId)
			}
			return nil, fmt.Errorf("update failed for product_id=%d: %w", item.ProductId, err)

		}

	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return order, nil

}
