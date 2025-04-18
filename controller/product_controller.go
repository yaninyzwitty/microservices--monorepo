package controller

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/pb"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/snowflake"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	if req.Name == "" || req.Description == "" {
		slog.Warn("Invalid product name or description", "name", req.Name, "description", req.Description)
		return nil, status.Errorf(codes.InvalidArgument, "invalid product name and description: %s, %s", req.Name, req.Description)
	}
	if req.Price <= 0 {
		slog.Warn("Invalid product price", "price", req.Price)
		return nil, status.Errorf(codes.InvalidArgument, "invalid product price: %f", req.Price)
	}
	if req.CategoryId <= 0 {
		slog.Warn("Invalid category ID", "categoryId", req.CategoryId)
		return nil, status.Errorf(codes.InvalidArgument, "invalid category ID: %d", req.CategoryId)
	}

	slog.Info("categoryid", "id", req.CategoryId)

	// Generate product ID
	productId, err := snowflake.GenerateID()
	if err != nil {
		slog.Error("Failed to generate product ID", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to generate product id: %v", err)
	}
	slog.Info("Generated product ID", "productId", productId)

	tx, err := c.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		slog.Error("Failed to begin transaction", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to begin transaction: %v", err)
	}
	slog.Info("Transaction started")

	var txError error
	defer func() {
		if txError != nil || err != nil {
			slog.Info("Rolling back transaction due to error")
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				slog.Error("Failed to rollback transaction", "rollbackError", rollbackErr)
			}
		}
	}()

	// ✅ Check if the category exists
	var categoryExists bool
	checkCategoryQuery := `SELECT EXISTS(SELECT 1 FROM categories WHERE id = $1)`
	err = tx.QueryRow(ctx, checkCategoryQuery, req.CategoryId).Scan(&categoryExists)
	if err != nil {
		slog.Error("Failed to check if category exists", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to validate category: %v", err)
	}
	if !categoryExists {
		slog.Warn("Category does not exist", "categoryId", req.CategoryId)
		return nil, status.Errorf(codes.InvalidArgument, "category with id %d does not exist", req.CategoryId)
	}

	// Insert product
	insertProductQuery := `INSERT INTO products (id, category_id, name, description, price) 
						   VALUES($1, $2, $3, $4, $5)`
	_, txError = tx.Exec(ctx, insertProductQuery, productId, req.CategoryId, req.Name, req.Description, req.Price)
	if txError != nil {
		slog.Error("Failed to insert product", "error", txError)
		return nil, status.Errorf(codes.Internal, "failed to insert product: %v", txError)
	}

	// Prepare product data
	product := &pb.Product{
		Id:          int64(productId),
		CategoryId:  req.CategoryId,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
		CreatedAt:   timestamppb.Now(),
		UpdatedAt:   timestamppb.Now(),
	}

	// Create outbox event
	outboxId, txError := snowflake.GenerateID()
	if txError != nil {
		slog.Error("Failed to generate outbox ID", "error", txError)
		return nil, status.Errorf(codes.Internal, "failed to generate outbox id: %v", txError)
	}
	slog.Info("Generated outbox ID", "outboxId", outboxId)

	payload, txError := protojson.Marshal(product)
	if txError != nil {
		slog.Error("Failed to marshal product for outbox", "error", txError)
		return nil, status.Errorf(codes.Internal, "failed to marshal product: %v", txError)
	}

	insertOutboxQuery := `INSERT INTO outbox (id, event_type, payload) VALUES($1, $2, $3)`
	_, txError = tx.Exec(ctx, insertOutboxQuery, outboxId, "product.created", payload)
	if txError != nil {
		slog.Error("Failed to insert outbox event", "error", txError)
		return nil, status.Errorf(codes.Internal, "failed to insert outbox event: %v", txError)
	}

	txError = tx.Commit(ctx)
	if txError != nil {
		slog.Error("Failed to commit transaction", "error", txError)
		return nil, status.Errorf(codes.Internal, "failed to commit transaction: %v", txError)
	}
	slog.Info("Transaction committed successfully", "productId", productId)

	return &pb.CreateProductResponse{
		Product: product,
	}, nil
}

func (c *ProductController) CreateCategory(ctx context.Context, req *pb.CreateCategoryRequest) (*pb.CreateCategoryResponse, error) {
	// Validate the category name
	if req.Name == "" {
		slog.Error("Invalid category name", "name", req.Name)
		return nil, status.Errorf(codes.InvalidArgument, "invalid category name: %s", req.Name)
	}

	// Log category name validation
	slog.Info("Received category name", "val", req.Name)

	// Generate category ID using Snowflake
	categoryId, err := snowflake.GenerateID()
	if err != nil {
		slog.Error("Failed to generate category ID", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to generate category id: %v", err)
	}
	slog.Info("Generated category ID", "categoryId", categoryId)

	// Insert category query
	insertCategoryQuery := `INSERT INTO categories (id, name, description) VALUES($1, $2, $3)`

	// Start a new transaction
	tx, err := c.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		slog.Error("Failed to begin transaction", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to begin transaction: %v", err)
	}
	slog.Info("Transaction started")

	// Ensure rollback in case of an error during transaction
	defer func() {
		if err != nil {
			slog.Info("Rolling back transaction due to error", "error", err)
			tx.Rollback(ctx)
		}
	}()

	// Insert category into the database
	_, err = tx.Exec(ctx, insertCategoryQuery, int64(categoryId), req.Name, req.Description)
	if err != nil {
		slog.Error("Failed to insert category into database", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to insert category: %v", err)
	}
	slog.Info("Category successfully inserted into the database", "categoryId", categoryId)

	// Prepare category data for the response
	category := &pb.Category{
		Id:          int64(categoryId),
		Name:        req.Name,
		Description: req.Description,
		CreatedAt:   timestamppb.Now(),
	}

	// Generate outbox ID for event propagation
	outboxId, err := snowflake.GenerateID()
	if err != nil {
		slog.Error("Failed to generate outbox ID", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to generate outbox id: %v", err)
	}
	slog.Info("Generated outbox ID", "outboxId", outboxId)

	// Serialize the category data for the outbox event
	payload, err := protojson.Marshal(category)
	if err != nil {
		slog.Error("Failed to marshal category for outbox event", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to marshal category: %v", err)
	}
	slog.Info("Category data marshaled for outbox event")

	// Insert outbox event into the database for messaging systems
	insertOutboxQuery := `INSERT INTO outbox (id, event_type, payload) VALUES($1, $2, $3)`
	_, err = tx.Exec(ctx, insertOutboxQuery, outboxId, "category.created", payload)
	if err != nil {
		slog.Error("Failed to insert outbox event", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to insert outbox event: %v", err)
	}
	slog.Info("Outbox event successfully inserted", "outboxId", outboxId)

	// Commit the transaction if everything is successful
	err = tx.Commit(ctx)
	if err != nil {
		slog.Error("Failed to commit transaction", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to commit transaction: %v", err)
	}
	slog.Info("Transaction successfully committed")

	// Return the successfully created category as a response
	slog.Info("Returning successfully created category", "categoryId", category.Id, "name", category.Name)
	return &pb.CreateCategoryResponse{
		Id:          category.Id,
		Name:        req.Name,
		Description: req.Description,
		CreatedAt:   category.CreatedAt,
	}, nil
}

func (c *ProductController) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.GetProductResponse, error) {
	// Validate the product ID
	if req.ProductId <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid product id: %d", req.ProductId)
	}

	// Query to fetch the product from the database
	selectProductQuery := `SELECT id, category_id, name, description, price, created_at, updated_at 
	                         FROM products WHERE id = $1`

	// Fetch the product from the database
	var product pb.Product
	err := c.pool.QueryRow(ctx, selectProductQuery, req.ProductId).Scan(
		&product.Id,
		&product.CategoryId,
		&product.Name,
		&product.Description,
		&product.Price,
		&product.CreatedAt,
		&product.UpdatedAt,
	)

	// Handle case where product is not found
	if err == pgx.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "product not found with id: %d", req.ProductId)
	}

	// Handle other errors
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch product: %v", err)
	}

	// Return the product details in the response
	return &pb.GetProductResponse{
		Product: &product,
	}, nil
}

func (c *ProductController) GetCategory(ctx context.Context, req *pb.GetCategoryRequest) (*pb.GetCategoryResponse, error) {
	// Validate the category ID
	if req.Id <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid category id: %d", req.Id)
	}

	slog.Info("value", "id", req.Id)
	// Query to fetch the category from the database
	selectCategoryQuery := `select id, name, description, created_at from categories where id = 133592043875221505`

	// Fetch the category from the database
	var category pb.Category
	var createdAt time.Time
	err := c.pool.QueryRow(ctx, selectCategoryQuery).Scan(
		&category.Id,
		&category.Name,
		&category.Description,
		&createdAt,
	)

	// Handle case where category is not found
	if err == pgx.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "category not found with id: %d", req.Id)
	}

	// Handle other errors
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch category: %v", err)
	}

	// Return the category details in the response
	return &pb.GetCategoryResponse{
		Id:          category.Id,
		Name:        category.Name,
		Description: category.Description,
		CreatedAt:   timestamppb.New(createdAt),
	}, nil
}

func (c *ProductController) ListProducts(ctx context.Context, req *pb.ListProductsRequest) (*pb.ListProductsResponse, error) {

	return &pb.ListProductsResponse{}, nil
}
