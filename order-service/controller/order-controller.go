package controller

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/pb"
)

type OrderController struct {
	pb.UnimplementedOrderServiceServer
	pool *pgxpool.Pool
}

func NewOrderController(pool *pgxpool.Pool) *OrderController {
	return &OrderController{pool: pool}
}

func (c *OrderController) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
	return nil, nil
}
func (c *OrderController) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.GetOrderResponse, error) {
	return nil, nil
}
func (c *OrderController) ListOrders(ctx context.Context, req *pb.ListOrdersRequest) (*pb.ListOrdersResponse, error) {
	return nil, nil
}
