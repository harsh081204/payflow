package models

import (
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	StatusCreated        OrderStatus = "CREATED"
	StatusPaymentPending OrderStatus = "PAYMENT_PENDING"
	StatusPaid           OrderStatus = "PAID"
	StatusFailed         OrderStatus = "FAILED"
	StatusShipped        OrderStatus = "SHIPPED"
)

type Order struct {
	ID        uuid.UUID   `json:"id"`
	UserID    uuid.UUID   `json:"user_id"`
	Amount    int64       `json:"amount"`   // Store as cents
	Currency  string      `json:"currency"` // e.g. "USD"
	Status    OrderStatus `json:"status"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

type CreateOrderRequest struct {
	UserID   uuid.UUID `json:"user_id"` // could also be extracted from JWT
	Amount   int64     `json:"amount"`
	Currency string    `json:"currency"`
}

type CreateOrderResponse struct {
	Order Order `json:"order"`
}
