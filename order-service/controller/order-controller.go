package controller

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/pb"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/pkg/snowflake"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type OrderController struct {
	pb.UnimplementedOrderServiceServer
	pool *pgxpool.Pool
}

func NewOrderController(pool *pgxpool.Pool) *OrderController {
	return &OrderController{pool: pool}
}

func (c *OrderController) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
	if len(req.Items) == 0 {
		return nil, status.Error(codes.InvalidArgument, "items cannot be empty")
	}
	if req.UserId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id must be a positive integer")
	}

	tx, err := c.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to begin transaction")
	}
	slog.Info("Transaction started")

	var rollbackErr error
	defer func() {
		if err != nil || rollbackErr != nil {
			slog.Info("Rolling back transaction due to error")
			if rb := tx.Rollback(ctx); rb != nil {
				slog.Error("Failed to rollback transaction", "rollbackError", rb)
			}
		}
	}()

	// Calculate total price
	var totalPrice float64
	for _, item := range req.Items {
		var price float64
		err = tx.QueryRow(ctx, `SELECT price FROM products WHERE id = $1`, item.ProductId).Scan(&price)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, status.Errorf(codes.NotFound, "product with id %d not found", item.ProductId)
			}
			return nil, status.Errorf(codes.Internal, "failed to query product price: %v", err)
		}
		totalPrice += price * float64(item.Quantity)
	}

	// Generate order ID
	orderId, err := snowflake.GenerateID()
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate order ID")
	}

	// Insert order
	_, err = tx.Exec(ctx, `
		INSERT INTO orders (id, user_id, total_price, status)
		VALUES ($1, $2, $3, $4)`,
		orderId, req.UserId, totalPrice, "PENDING",
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to insert order: %v", err)
	}

	// Insert order items
	orderItemStmt := `
		INSERT INTO order_items (order_id, product_id, quantity, unit_price)
		VALUES ($1, $2, $3, $4)`

	for _, item := range req.Items {
		_, err = tx.Exec(ctx, orderItemStmt, orderId, item.ProductId, item.Quantity, item.UnitPrice)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to insert order item: %v", err)
		}
	}

	// Prepare Order protobuf
	order := pb.Order{
		Id:         int64(orderId),
		UserId:     req.UserId,
		TotalPrice: float32(totalPrice),
		Status:     "PENDING",
		CreatedAt:  timestamppb.Now(),
		UpdatedAt:  timestamppb.Now(),
		Items:      req.Items,
	}

	// Insert into outbox
	outboxID, err := snowflake.GenerateID()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate outbox ID: %v", err)
	}

	payload, err := protojson.Marshal(&order)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal order payload: %v", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO outbox (id, event_type, payload) VALUES ($1, $2, $3)`,
		outboxID, "order.created", payload,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to insert outbox event: %v", err)
	}

	// Commit transaction
	err = tx.Commit(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit transaction: %v", err)
	}

	slog.Info("Order created and committed", "orderID", orderId)

	return &pb.CreateOrderResponse{Order: &order}, nil
}

func (c *OrderController) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.GetOrderResponse, error) {
	if req.OrderId <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "order id must be positive")
	}

	rows, err := c.pool.Query(ctx, `
		SELECT 
			o.id, o.user_id, o.status, o.total_price, o.created_at, o.updated_at,
			oi.product_id, oi.quantity, oi.unit_price
		FROM orders o
		LEFT JOIN order_items oi ON o.id = oi.order_id
		WHERE o.id = $1
	`, req.OrderId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query failed: %v", err)
	}
	defer rows.Close()

	var (
		order      *pb.Order
		orderItems []*pb.OrderItem
		createdAt  time.Time
		updatedAt  time.Time
	)

	var returnedOrder pb.Order
	var returnedOrderItem pb.OrderItem

	found := false
	for rows.Next() {
		if err := rows.Scan(&returnedOrder.Id, &returnedOrder.UserId, &returnedOrder.Status, &returnedOrder.TotalPrice, &createdAt, &updatedAt, &returnedOrderItem.ProductId, &returnedOrderItem.Quantity, &returnedOrderItem.UnitPrice); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan row: %v", err)

		}

		// initialize order one
		if order == nil {
			order = &pb.Order{
				Id:         returnedOrder.Id,
				UserId:     returnedOrder.UserId,
				Status:     returnedOrder.Status,
				TotalPrice: returnedOrder.TotalPrice,
				CreatedAt:  timestamppb.New(createdAt),
				UpdatedAt:  timestamppb.New(updatedAt),
				Items:      []*pb.OrderItem{},
			}
		}

		// Append order items only if product_id is present (LEFT JOIN may yield null)
		if returnedOrderItem.ProductId != 0 {
			order.Items = append(order.Items, &pb.OrderItem{
				ProductId: returnedOrderItem.ProductId,
				Quantity:  returnedOrderItem.Quantity,
				UnitPrice: returnedOrderItem.UnitPrice,
			})

		}

		found = true

	}
	if err := rows.Err(); err != nil {
		return nil, status.Errorf(codes.Internal, "row iteration error: %v", err)
	}

	if !found {
		return nil, status.Errorf(codes.NotFound, "order not found")
	}

	order.Items = orderItems

	return &pb.GetOrderResponse{Order: order}, nil
}

func (c *OrderController) ListOrders(ctx context.Context, req *pb.ListOrdersRequest) (*pb.ListOrdersResponse, error) {
	if req.UserId <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "user_id must be positive")
	}
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	rows, err := c.pool.Query(ctx, `
		SELECT 
			o.id, o.user_id, o.status, o.total_price, o.created_at, o.updated_at,
			oi.product_id, oi.quantity, oi.unit_price
		FROM orders o
		LEFT JOIN order_items oi ON o.id = oi.order_id
		WHERE o.user_id = $1
		ORDER BY o.created_at DESC
		OFFSET $2 LIMIT $3
	`, req.UserId, req.Offset, req.Limit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query failed: %v", err)
	}
	defer rows.Close()

	return &pb.ListOrdersResponse{}, nil

}
