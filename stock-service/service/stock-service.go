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
	warehouseIds := []int64{134496091390406657, 134496170075549697, 134496187540631553, 134496202942115841, 134496215558582273}
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
