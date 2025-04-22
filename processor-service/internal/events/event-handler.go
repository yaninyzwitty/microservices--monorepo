package events

import (
	"fmt"

	"github.com/yaninyzwitty/eccomerce-microservices-backend/pb"
	"google.golang.org/protobuf/encoding/protojson"
)

func HandleCategoryCreated(payload string) ([]byte, error) {
	var category pb.OutboxEvent
	if err := protojson.Unmarshal([]byte(payload), &category); err != nil {
		return nil, fmt.Errorf("error unmarshalling category: %w", err)
	}

	return protojson.Marshal(&category)
}

func HandleProductCreated(payload string) ([]byte, error) {
	var product pb.Product
	if err := protojson.Unmarshal([]byte(payload), &product); err != nil {
		return nil, fmt.Errorf("error unmarshalling product: %w", err)
	}

	return protojson.Marshal(&product)
}
