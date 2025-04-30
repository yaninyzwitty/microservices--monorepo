package events

import (
	"fmt"
	"log/slog"

	"github.com/yaninyzwitty/eccomerce-microservices-backend/pb"
	"google.golang.org/protobuf/encoding/protojson"
)

func HandleCategoryCreated(payload string) ([]byte, error) {
	category := &pb.Category{} // ✅ properly allocated
	if err := protojson.Unmarshal([]byte(payload), category); err != nil {
		return nil, fmt.Errorf("error unmarshalling category: %w", err)
	}

	slog.Info("name", "val", category.Name)
	slog.Info("id", "val", category.Id)
	slog.Info("description", "val", category.Description)
	slog.Info("createdAt", "val", category.CreatedAt)

	return protojson.Marshal(category)
}

func HandleProductCreated(payload string) ([]byte, error) {
	product := &pb.Product{} // ✅ properly allocated

	if err := protojson.Unmarshal([]byte(payload), product); err != nil {
		return nil, fmt.Errorf("error unmarshalling product: %w", err)
	}

	return protojson.Marshal(product)
}
func HandleStockCreated(payload string) ([]byte, error) {
	stockLevel := &pb.StockLevel{} // ✅ properly allocated

	if err := protojson.Unmarshal([]byte(payload), stockLevel); err != nil {
		return nil, fmt.Errorf("error unmarshalling product: %w", err)
	}

	return protojson.Marshal(stockLevel)
}
