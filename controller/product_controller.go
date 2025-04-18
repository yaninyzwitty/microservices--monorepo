package controller

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/pb"
)

type ProductController struct {
	pb.UnimplementedProductServiceServer
	pool *pgxpool.Pool
}

func NewProductController(pool *pgxpool.Pool) *ProductController {
	return &ProductController{
		pool: pool,
	}
}

func (c *ProductController) CreateProduct(ctx context.Context, req *pb.CreateProductRequest) (*pb.CreateProductResponse, error) {
	return &pb.CreateProductResponse{}, nil
}

func (c *ProductController) CreateCategory(ctx context.Context, req *pb.CreateCategoryRequest) (*pb.CreateCategoryResponse, error) {
	return &pb.CreateCategoryResponse{}, nil
}

func (c *ProductController) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.GetProductResponse, error) {
	return &pb.GetProductResponse{}, nil
}

func (c *ProductController) GetCategory(ctx context.Context, req *pb.GetCategoryRequest) (*pb.GetCategoryResponse, error) {
	return &pb.GetCategoryResponse{}, nil
}

func (c *ProductController) ListProducts(ctx context.Context, req *pb.ListProductsRequest) (*pb.ListProductsResponse, error) {
	return &pb.ListProductsResponse{}, nil
}
