package outbox

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type OutboxRepository struct {
	pool *pgxpool.Pool
}

func NewOutboxRepository(pool *pgxpool.Pool) *OutboxRepository {
	return &OutboxRepository{pool: pool}
}

// FetchMessages retrieves up to 5 unprocessed outbox messages.
func (r *OutboxRepository) FetchMessages(ctx context.Context) ([]*pb.OutboxEvent, error) {
	query := `
		SELECT id, event_type, payload, created_at, processed_at
		FROM outbox
		WHERE processed_at IS NULL
		ORDER BY created_at ASC
		LIMIT 5
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}
	defer rows.Close()

	var messages []*pb.OutboxEvent

	for rows.Next() {
		var (
			id          int64
			eventType   string
			payload     []byte
			createdAt   time.Time
			processedAt *time.Time // nullable
		)

		if err := rows.Scan(&id, &eventType, &payload, &createdAt, &processedAt); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		msg := &pb.OutboxEvent{
			Id:        id,
			EventType: eventType,
			Payload:   payload,
			CreatedAt: timestamppb.New(createdAt),
		}

		// if processedAt != nil {
		// 	msg.ProcessedAt = timestamppb.New(*processedAt)
		// }

		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error while iterating rows: %w", err)
	}

	return messages, nil
}

// MarkMessageProcessed updates the processed_at column to now for a given message ID.
func (r *OutboxRepository) MarkMessageProcessed(ctx context.Context, messageID int64) error {
	query := `
		UPDATE outbox
		SET processed_at = now()
		WHERE id = $1
	`

	cmdTag, err := r.pool.Exec(ctx, query, messageID)
	if err != nil {
		return fmt.Errorf("failed to mark message %d as processed: %w", messageID, err)
	}

	if cmdTag.RowsAffected() != 1 {
		return fmt.Errorf("no rows updated for message ID %d", messageID)
	}

	return nil
}
