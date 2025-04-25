package repository

import (
	"context"
	"fmt"

	"github.com/resend/resend-go/v2"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/pb"
)

type NotificationRepository struct {
	resendClient *resend.Client
}

func NewNotificationRepository(resendClient *resend.Client) *NotificationRepository {
	return &NotificationRepository{
		resendClient: resendClient,
	}
}

func (r *NotificationRepository) SendProductNotification(ctx context.Context, product *pb.Product, from, to string) (bool, string, error) {

	html := fmt.Sprintf(`
	<div style="font-family: Arial, sans-serif; padding: 20px; background-color: #f9f9f9;">
	  <h1 style="color: #4CAF50;">🎉 New Product Alert!</h1>
	  <p>We are excited to introduce a new product to our collection!</p>
	  <div style="padding: 15px; background-color: #ffffff; border: 1px solid #ddd; border-radius: 8px;">
		<h2 style="color: #333;">%s</h2>
		<p><strong>Description:</strong> %s</p>
		<p><strong>Price:</strong> $%.2f</p>
		<p><strong>Stock Available:</strong> %d units</p>
		<p><strong>Category ID:</strong> %d</p>
		<p style="font-size: 12px; color: #888;">Created at: %s</p>
	  </div>
	  <p style="margin-top: 20px;">Visit our store to explore more amazing products!</p>
	</div>`,
		product.Name,
		product.Description,
		product.Price,
		product.Stock,
		product.CategoryId,
		product.CreatedAt.AsTime().Format("Jan 2, 2006 at 3:04pm"),
	)

	params := &resend.SendEmailRequest{
		From:    from,
		To:      []string{to},
		Html:    html,
		Subject: fmt.Sprintf("New Product: %s", product.Name),
	}

	email, err := r.resendClient.Emails.Send(params)
	if err != nil {
		return false, "", err
	}

	return true, email.Id, nil
}
func (r *NotificationRepository) SendStockNotification(ctx context.Context, stock *pb.StockLevel, from, to string) (bool, string, error) {
	html := fmt.Sprintf(`
	<div style="font-family: Arial, sans-serif; padding: 20px; background-color: #f0f8ff;">
	  <h1 style="color: #2196F3;">📦 Stock Level Update</h1>
	  <p>We wanted to keep you in the loop with the latest inventory status.</p>
	  <div style="padding: 15px; background-color: #ffffff; border: 1px solid #ccc; border-radius: 8px;">
		<p><strong>Product ID:</strong> %d</p>
		<p><strong>Warehouse ID:</strong> %d</p>
		<p><strong>Quantity in Stock:</strong> %d units</p>
		<p style="font-size: 12px; color: #888;">Updated on: %s</p>
	  </div>
	  <p style="margin-top: 20px;">Keep an eye on your stock to ensure timely replenishment!</p>
	</div>`,
		stock.ProductId,
		stock.WarehouseId,
		stock.Quantity,
		stock.CreatedAt.AsTime().Format("Jan 2, 2006 at 3:04pm"),
	)

	params := &resend.SendEmailRequest{
		From:    from,
		To:      []string{to},
		Html:    html,
		Subject: fmt.Sprintf("Stock Update for Product ID: %d", stock.ProductId),
	}

	email, err := r.resendClient.Emails.Send(params)
	if err != nil {
		return false, "", err
	}

	return true, email.Id, nil
}
