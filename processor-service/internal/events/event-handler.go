package events

import (
	"fmt"

	"github.com/yaninyzwitty/eccomerce-microservices-backend/pb"
	"google.golang.org/protobuf/encoding/protojson"
)

func HandleCategoryCreated(payload string) ([]byte, error) {
	category := &pb.Category{} // ✅ properly allocated
	if err := protojson.Unmarshal([]byte(payload), category); err != nil {
		return nil, fmt.Errorf("error unmarshalling category: %w", err)
	}

	return protojson.Marshal(category)
}

func HandleOrderCreated(payload string) ([]byte, error) {
	order := &pb.Order{} // ✅ properly allocated
	if err := protojson.Unmarshal([]byte(payload), order); err != nil {
		return nil, fmt.Errorf("error unmarshalling order: %w", err)
	}

	return protojson.Marshal(order)

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
