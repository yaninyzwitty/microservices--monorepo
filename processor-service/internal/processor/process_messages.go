package processor

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/yaninyzwitty/eccomerce-microservices-backend/pb"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/processor-service/internal/events"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/processor-service/internal/publisher"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/processor-service/outbox"
)

type ProcessMessage struct {
	repo               outbox.OutboxRepository
	pulsarProducerRepo publisher.PulsarPublisher
}

func NewProcessMessage(repo outbox.OutboxRepository, producerRepo publisher.PulsarPublisher) *ProcessMessage {
	return &ProcessMessage{repo: repo, pulsarProducerRepo: producerRepo}
}

func (pm *ProcessMessage) ProcessMessages(ctx context.Context) error {
	messages, err := pm.repo.FetchMessages(ctx)
	if err != nil {
		return fmt.Errorf("error fetching messages: %w", err)
	}

	for _, message := range messages {
		slog.Info("started processing messages...")
		payload, err := pm.handleEvent(message)

		if err != nil {
			slog.Error("Failed to process event", "error", err, "eventType", message.EventType)
			continue
		}

		key := fmt.Sprintf("%s:%v", message.EventType, message.Id)

		if err := pm.pulsarProducerRepo.Publish(ctx, message.EventType, key, payload); err != nil {
			slog.Error("Failed to send message to Pulsar", "error", err, "messageID", message.Id)
			continue
		}

		if err := pm.repo.MarkMessageProcessed(ctx, message.Id); err != nil {
			slog.Error("Failed to mark message as processed", "error", err, "messageID", message.Id)
		}

	}

	return nil
}

func (pm *ProcessMessage) handleEvent(message *pb.OutboxEvent) ([]byte, error) {

	switch message.EventType {
	case "category.created":
		return events.HandleCategoryCreated(string(message.Payload))
	case "product.created":

		return events.HandleProductCreated(string(message.Payload))
	default:
		slog.Warn("Unknown event type, skipping", "eventType", message.EventType, "messageID", message.Id)
		return nil, nil
	}
}
