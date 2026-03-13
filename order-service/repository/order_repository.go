package repository

import (
	"context"
	"errors"
	"order-service/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderRepository interface {
	CreateOrder(ctx context.Context, order *models.Order) error
	GetOrderByID(ctx context.Context, id uuid.UUID) (*models.Order, error)
	GetOrdersByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.Order, error)
	UpdateOrderStatus(ctx context.Context, id uuid.UUID, status models.OrderStatus) error
}

type orderRepository struct {
	db *pgxpool.Pool
}

func NewOrderRepository(db *pgxpool.Pool) OrderRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) CreateOrder(ctx context.Context, order *models.Order) error {
	query := `
		INSERT INTO orders (id, user_id, amount, currency, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.Exec(ctx, query, order.ID, order.UserID, order.Amount, order.Currency, order.Status, order.CreatedAt, order.UpdatedAt)
	return err
}

func (r *orderRepository) GetOrderByID(ctx context.Context, id uuid.UUID) (*models.Order, error) {
	query := `
		SELECT id, user_id, amount, currency, status, created_at, updated_at
		FROM orders
		WHERE id = $1
	`
	row := r.db.QueryRow(ctx, query, id)

	var order models.Order
	err := row.Scan(&order.ID, &order.UserID, &order.Amount, &order.Currency, &order.Status, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Not found
		}
		return nil, err
	}
	return &order, nil
}

func (r *orderRepository) GetOrdersByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.Order, error) {
	query := `
		SELECT id, user_id, amount, currency, status, created_at, updated_at
		FROM orders
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		err := rows.Scan(&order.ID, &order.UserID, &order.Amount, &order.Currency, &order.Status, &order.CreatedAt, &order.UpdatedAt)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

func (r *orderRepository) UpdateOrderStatus(ctx context.Context, id uuid.UUID, status models.OrderStatus) error {
	query := `
		UPDATE orders
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`
	cmdTag, err := r.db.Exec(ctx, query, status, id)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return errors.New("order not found")
	}
	return nil
}
