package service

import (
	"context"
	"errors"
	"order-service/models"
	"order-service/repository"
	"time"

	"github.com/google/uuid"
)

var (
	ErrOrderNotFound    = errors.New("order not found")
	ErrInvalidOrderData = errors.New("invalid order data")
)

type OrderService interface {
	CreateOrder(ctx context.Context, req *models.CreateOrderRequest) (*models.Order, error)
	GetOrder(ctx context.Context, id uuid.UUID) (*models.Order, error)
	GetOrdersByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.Order, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status models.OrderStatus) error
}

type orderService struct {
	repo repository.OrderRepository
}

func NewOrderService(repo repository.OrderRepository) OrderService {
	return &orderService{repo: repo}
}

func (s *orderService) CreateOrder(ctx context.Context, req *models.CreateOrderRequest) (*models.Order, error) {
	if req.Amount <= 0 || req.Currency == "" {
		return nil, ErrInvalidOrderData
	}

	order := &models.Order{
		ID:        uuid.New(),
		UserID:    req.UserID,
		Amount:    req.Amount,
		Currency:  req.Currency,
		Status:    models.StatusCreated,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.CreateOrder(ctx, order); err != nil {
		return nil, err
	}

	// Here a message should be published to Kafka/RabbitMQ: 'order.created'
	// for the Payment Service and Notification Service to pick up.

	return order, nil
}

func (s *orderService) GetOrder(ctx context.Context, id uuid.UUID) (*models.Order, error) {
	order, err := s.repo.GetOrderByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, ErrOrderNotFound
	}
	return order, nil
}

func (s *orderService) GetOrdersByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.Order, error) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.GetOrdersByUserID(ctx, userID, limit, offset)
}

func (s *orderService) UpdateStatus(ctx context.Context, id uuid.UUID, status models.OrderStatus) error {
	err := s.repo.UpdateOrderStatus(ctx, id, status)
	if err != nil && err.Error() == "order not found" {
		return ErrOrderNotFound
	}
	return err
}
