package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/yaninyzwitty/eccomerce-microservices-backend/notification-service/internal/controller"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/pb"
)

type NotificationService struct {
	notificationController *controller.NotificationController
}

func NewNotificationService(notificationController *controller.NotificationController) *NotificationService {
	return &NotificationService{
		notificationController: notificationController,
	}
}

func (s *NotificationService) SendProductNotification(ctx context.Context, product *pb.Product) error {
	notificationRes, err := s.notificationController.SendProductEmail(ctx, &pb.SendProductEmailRequest{
		Id:          product.Id,
		CategoryId:  product.CategoryId,
		Name:        product.Name,
		Description: product.GetDescription(),
		Price:       product.Price,
		Stock:       product.Stock,
	})

	if err != nil || !notificationRes.Success {
		return fmt.Errorf("failed to send product notification: %v", err)
	}

	slog.Info("sent notication", "emailId", notificationRes.EmailId)
	return nil
}
func (s *NotificationService) SendStockNotification(ctx context.Context, stock *pb.StockLevel) error {
	stockNotificationRes, err := s.notificationController.SendStockEmail(ctx, &pb.SendStockEmailRequest{
		ProductId:   stock.ProductId,
		WarehouseId: stock.WarehouseId,
		Quantity:    stock.Quantity,
		CreatedAt:   stock.CreatedAt,
	})
	if err != nil || !stockNotificationRes.Success {
		return fmt.Errorf("failed to send product notification: %v", err)
	}

	slog.Info("sent notication", "emailId", stockNotificationRes.EmailId)
	return nil

}
