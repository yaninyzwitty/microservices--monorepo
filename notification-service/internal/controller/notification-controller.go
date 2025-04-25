package controller

import (
	"context"

	"github.com/yaninyzwitty/eccomerce-microservices-backend/notification-service/internal/repository"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type NotificationController struct {
	pb.UnimplementedNotificationServiceServer
	repo *repository.NotificationRepository
}

func NewNotificationController(repo *repository.NotificationRepository) *NotificationController {
	return &NotificationController{
		repo: repo,
	}
}

func (c *NotificationController) SendProductEmail(ctx context.Context, req *pb.SendProductEmailRequest) (*pb.SendProductEmailResponse, error) {

	sent, res, err := c.repo.SendProductNotification(ctx, &pb.Product{
		Id:          req.Id,
		CategoryId:  req.CategoryId,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
		CreatedAt:   req.CreatedAt,
		UpdatedAt:   req.UpdatedAt,
	}, "Acme <onboarding@resend.dev>", "ianmwa143@gmail.com")

	if err != nil || !sent {
		return nil, status.Errorf(codes.Internal, "failed to send product email: %v", err)
	}

	return &pb.SendProductEmailResponse{
		Success: sent,
		EmailId: res,
	}, nil
}

func (c *NotificationController) SendStockEmail(ctx context.Context, req *pb.SendStockEmailRequest) (*pb.SendStockEmailResponse, error) {
	sent, res, err := c.repo.SendStockNotification(ctx, &pb.StockLevel{
		ProductId:   req.ProductId,
		WarehouseId: req.WarehouseId,
		Quantity:    req.Quantity,
		CreatedAt:   req.CreatedAt,
	}, "Acme <onboarding@resend.dev>", "ianmwa143@gmail.com")

	if err != nil || !sent {
		return nil, status.Errorf(codes.Internal, "failed to send product email: %v", err)
	}

	return &pb.SendStockEmailResponse{
		Success: sent,
		EmailId: res,
	}, nil

}
