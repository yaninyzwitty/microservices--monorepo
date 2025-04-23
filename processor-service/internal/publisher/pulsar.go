package publisher

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/apache/pulsar-client-go/pulsar"
)

// PulsarPublisher defines an interface for publishing messages to Pulsar.
type PulsarPublisher interface {
	Publish(ctx context.Context, eventType string, key string, payload []byte) error
}

// pulsarProducer implements the PulsarPublisher interface.
type pulsarProducer struct {
	producer pulsar.Producer
}

// NewPulsarProducer creates a new instance of pulsarProducer.
func NewPulsarProducer(producer pulsar.Producer) PulsarPublisher {
	return &pulsarProducer{
		producer: producer,
	}
}

// Publish sends a message to the Pulsar topic using SendAsync.
func (p *pulsarProducer) Publish(ctx context.Context, eventType string, key string, payload []byte) error {
	messageChan := make(chan error, 1)

	p.producer.SendAsync(ctx, &pulsar.ProducerMessage{
		Key:     key,
		Payload: payload,
		Properties: map[string]string{
			"event_type": eventType,
		},
	}, func(mi pulsar.MessageID, pm *pulsar.ProducerMessage, err error) {
		messageChan <- err
	})

	select {
	case err := <-messageChan:
		if err != nil {
			return fmt.Errorf("failed to publish message: %w", err)
		}
	case <-ctx.Done():
		return fmt.Errorf("context canceled while publishing message")
	}

	slog.Info("Message sent to Pulsar", "key", key, "eventType", eventType)
	return nil
}
