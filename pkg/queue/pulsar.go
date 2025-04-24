package queue

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/apache/pulsar-client-go/pulsar"
)

// Config holds the configuration for Pulsar connection.
type Config struct {
	URI       string
	Token     string
	TopicName string
}

// Service provides methods to interact with Pulsar.
type Service struct {
	cfg *Config
}

var (
	subscriptionName = "my-subscription"
)

// NewService creates a new Pulsar service.
func NewService(cfg *Config) *Service {
	return &Service{cfg: cfg}
}

// CreateConnection establishes a connection to Pulsar.
func (s *Service) CreateConnection(ctx context.Context) (pulsar.Client, error) {
	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL:            s.cfg.URI,
		Authentication: pulsar.NewAuthenticationToken(s.cfg.Token),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Pulsar client: %w", err)
	}
	slog.Info("Pulsar connection established")
	return client, nil
}

// CreateProducer creates a producer for the configured topic.
func (s *Service) CreateProducer(ctx context.Context, client pulsar.Client) (pulsar.Producer, error) {
	producer, err := client.CreateProducer(pulsar.ProducerOptions{
		Topic: s.cfg.TopicName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Pulsar producer: %w", err)
	}
	slog.Info("Pulsar producer created", "topic", s.cfg.TopicName)
	return producer, nil
}

// CreateConsumer creates a consumer for a specific topic.
func (s *Service) CreateConsumer(ctx context.Context, client pulsar.Client, consumerTopic, subscriptionName string) (pulsar.Consumer, error) {

	consumer, err := client.Subscribe(pulsar.ConsumerOptions{
		Topic:                       consumerTopic,
		SubscriptionName:            subscriptionName,
		SubscriptionInitialPosition: pulsar.SubscriptionPositionEarliest,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Pulsar consumer: %w", err)
	}
	slog.Info("Pulsar consumer created", "topic", consumerTopic)
	return consumer, nil
}
