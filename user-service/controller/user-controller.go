package controller

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/pb"
	"github.com/yaninyzwitty/eccomerce-microservices-backend/pkg/snowflake"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type UserController struct {
	pb.UnimplementedUserServiceServer
	pool *pgxpool.Pool
}

func NewUserController(pool *pgxpool.Pool) *UserController {
	return &UserController{
		pool: pool,
	}
}

func (c *UserController) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	if req.Email == "" || req.FirstName == "" || req.LastName == "" {
		return nil, status.Errorf(codes.InvalidArgument, "email, firstname and lastname are invalid")
	}

	userId, err := snowflake.GenerateID()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate user id")
	}

	query := `
		INSERT INTO users (id, first_name, last_name, email)
		VALUES ($1, $2, $3, $4)
		RETURNING id, first_name, last_name, email, created_at, updated_at
	`
	var user pb.User
	var createdAt, updatedAt time.Time
	if err := c.pool.QueryRow(ctx, query, userId, req.FirstName, req.LastName, req.Email).Scan(&user.Id, &user.FirstName, &user.LastName, &user.Email, &createdAt, &updatedAt); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create user")
	}

	user.CreatedAt = timestamppb.New(createdAt)
	user.UpdatedAt = timestamppb.New(updatedAt)

	return &pb.CreateUserResponse{
		User: &user,
	}, nil
}
func (c *UserController) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	if req.GetId() <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "user ID is required")
	}

	query := `DELETE FROM users WHERE id = $1`
	tag, err := c.pool.Exec(ctx, query, req.GetId())
	if err != nil {
		return &pb.DeleteUserResponse{
			Deleted: false,
		}, status.Errorf(codes.Internal, "failed to delete user: %v", err)
	}

	if tag.RowsAffected() == 0 {
		return &pb.DeleteUserResponse{
			Deleted: false,
		}, status.Errorf(codes.NotFound, "user not found")
	}

	return &pb.DeleteUserResponse{
		Deleted: true,
	}, nil

}
func (c *UserController) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	if req.Email == "" || req.FirstName == "" || req.LastName == "" {
		return nil, status.Errorf(codes.InvalidArgument, "email, firstname and lastname are invalid")
	}
	if req.GetId() <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "user ID is required")
	}

	query := `
		UPDATE users
		SET first_name = $1,
		    last_name = $2,
		    email = $3,
		    updated_at = NOW()
		WHERE id = $4
		RETURNING id, first_name, last_name, email, created_at, updated_at
	`

	var user pb.User
	var createdAt, updatedAt time.Time

	err := c.pool.QueryRow(ctx, query, req.FirstName, req.LastName, req.Email, req.GetId()).
		Scan(&user.Id, &user.FirstName, &user.LastName, &user.Email, &createdAt, &updatedAt)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			return nil, status.Errorf(codes.AlreadyExists, "email already exists")
		}
		return nil, status.Errorf(codes.Internal, "failed to update user: %v", err)
	}

	user.CreatedAt = timestamppb.New(createdAt)
	user.UpdatedAt = timestamppb.New(updatedAt)

	return &pb.UpdateUserResponse{
		User: &user,
	}, nil

}
func (c *UserController) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	if req.GetId() <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "user ID is required")
	}

	query := `
		SELECT id, first_name, last_name, email, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user pb.User
	var createdAt, updatedAt time.Time

	if err := c.pool.QueryRow(ctx, query, req.GetId()).Scan(&user.Id, &user.FirstName, &user.LastName, &user.Email, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to fetch user: %v", err)

	}

	user.CreatedAt = timestamppb.New(createdAt)
	user.UpdatedAt = timestamppb.New(updatedAt)

	return &pb.GetUserResponse{
		User: &user,
	}, nil

}
func (c *UserController) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	limit := req.GetLimit()
	if limit <= 0 || limit > 100 {
		limit = 20 // sensible default
	}

	offset := max(req.GetOffset(), 0)

	query := `
		SELECT id, first_name, last_name, email, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := c.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list users: %v", err)
	}
	defer rows.Close()

	var users []*pb.User

	for rows.Next() {
		var user pb.User
		var createdAt, updatedAt time.Time

		if err := rows.Scan(&user.Id, &user.FirstName, &user.LastName, &user.Email, &createdAt, &updatedAt); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan user: %v", err)
		}

		user.CreatedAt = timestamppb.New(createdAt)
		user.UpdatedAt = timestamppb.New(updatedAt)

		users = append(users, &user)

	}
	if err := rows.Err(); err != nil {
		return nil, status.Errorf(codes.Internal, "row iteration error: %v", err)
	}
	return &pb.ListUsersResponse{
		Users: users,
	}, nil
}
