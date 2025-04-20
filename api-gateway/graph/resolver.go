package graph

import "github.com/yaninyzwitty/eccomerce-microservices-backend/pb"

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	ProductClient pb.ProductServiceClient
}
