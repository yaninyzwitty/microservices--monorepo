package notification

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/resend/resend-go/v2"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/pb"
	"google.golang.org/protobuf/encoding/protojson"
)

type NotificationService struct {
	resendClient *resend.Client
}

func NewService(resend *resend.Client) *NotificationService {
	return &NotificationService{resendClient: resend}
}

func (s *NotificationService) HandleProductCreated(ctx context.Context, msg pulsar.Message) (string, error) {
	var product pb.Product
	if err := protojson.Unmarshal(msg.Payload(), &product); err != nil {
		slog.Error("invalid product.created payload", "err", err)
		return "", fmt.Errorf("failed to umarshal payload to proto: %v", err)
	}

	emailId, err := s.SendProductCreatedEmail("Acme <onboarding@resend.dev>", []string{"ianmwa143@gmail.com"})
	if err != nil {
		slog.Error("failed to send product created email", "err", err)
		return "", fmt.Errorf("failed to send product created email: %v", err)
	}

	return emailId, nil

}

func (s *NotificationService) SendProductCreatedEmail(From string, TO []string) (string, error) {
	params := &resend.SendEmailRequest{
		From:    From,
		To:      TO,
		Html:    "<strong>hello world</strong>",
		Subject: "Hello from Golang",
		Cc:      []string{"cc@example.com"},
		Bcc:     []string{"bcc@example.com"},
		ReplyTo: "replyto@example.com",
	}

	sent, err := s.resendClient.Emails.Send(params)
	if err != nil {
		return "", fmt.Errorf("failed to send email: %v", err)
	}

	return sent.Id, nil

}
