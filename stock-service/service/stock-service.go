package service

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"

	"github.com/yaninyzwitty/eccomerce-microservices-backend/pb"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/stock-service/controller"
)

type StockService struct {
	stockController *controller.StockController
}

func NewStockService(stockController *controller.StockController) *StockService {
	return &StockService{
		stockController: stockController,
	}
}

func (s *StockService) CreateStockProduct(ctx context.Context, product *pb.Product) error {
	warehouseIds := []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	randIdx := rand.Intn(len(warehouseIds))

	stockResponse, err := s.stockController.CreateStock(ctx, &pb.CreateStockRequest{
		ProductId:   product.Id,
		WarehouseId: warehouseIds[randIdx],
		Quantity:    product.Stock,
	})

	if err != nil {
		return fmt.Errorf("failed to create stock: %v", err)
	}

	slog.Info("Quantity", "QTY", stockResponse.Stocklevel.Quantity)

	return nil
}
