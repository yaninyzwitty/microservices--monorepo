package controller

import (
	"context"

	"github.com/yaninyzwitty/eccomerce-microservices-backend/pb"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/pkg/snowflake"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/stock-service/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type StockController struct {
	pb.UnimplementedStockServiceServer
	stockRepo *repository.StockRepository
}

func NewStockController(repo *repository.StockRepository) *StockController {
	return &StockController{
		stockRepo: repo,
	}
}

func (c *StockController) CreateStock(ctx context.Context, req *pb.CreateStockRequest) (*pb.CreateStockResponse, error) {
	if req.ProductId <= 0 || req.WarehouseId <= 0 || req.Quantity <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid productId, warehouseId or quantity id")
	}

	stockLevelResponse, err := c.stockRepo.AddStockProduct(ctx, &pb.StockLevel{
		ProductId:   req.ProductId,
		WarehouseId: req.WarehouseId,
		Quantity:    req.Quantity,
		CreatedAt:   timestamppb.Now(),
	})

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create stock: %v", err)
	}

	return &pb.CreateStockResponse{
		Stocklevel: stockLevelResponse,
	}, nil
}

func (c *StockController) UpdateStock(ctx context.Context, req *pb.UpdateStockRequest) (*pb.UpdateStockResponse, error) {
	return nil, nil
}

func (c *StockController) ReserveStock(ctx context.Context, req *pb.ReserveStockRequest) (*pb.ReserveStockResponse, error) {
	if req.Order == nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid order")
	}

	reserveOrderRes, err := c.stockRepo.ReserveStockItem(ctx, req.Order)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to reserve stock: %v", err)
	}

	return &pb.ReserveStockResponse{
		Order: reserveOrderRes,
	}, nil
}
func (c *StockController) RemoveAllFromStock(ctx context.Context, req *pb.RemoveAllFromStockRequest) (*pb.RemoveAllFromStockResponse, error) {
	return nil, nil
}
func (c *StockController) AddWarehouse(ctx context.Context, req *pb.AddWarehouseRequest) (*pb.AddWarehouseResponse, error) {
	if req.Name == "" || req.Location == "" {
		return nil, status.Errorf(codes.InvalidArgument, "invalid name or location")
	}

	warehouseId, err := snowflake.GenerateID()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate warehouse id: %v", err)
	}

	AddWarehouseResponse, err := c.stockRepo.AddWarehouse(ctx, &pb.Warehouse{
		Id:        int64(warehouseId),
		Name:      req.Name,
		Location:  req.Location,
		CreatedAt: timestamppb.Now(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create warehouse: %v", err)

	}

	return &pb.AddWarehouseResponse{
		Warehouse: AddWarehouseResponse,
	}, nil
}
func (c *StockController) DeleteWarehouse(ctx context.Context, req *pb.DeleteWarehouseRequest) (*pb.DeleteWarehouseResponse, error) {
	return nil, nil
}
