package models

import (
	"time"

	"github.com/google/uuid"
)

type TransactionStatus string
type EntryType string

const (
	StatusPending   TransactionStatus = "PENDING"
	StatusSucceeded TransactionStatus = "SUCCEEDED"
	StatusFailed    TransactionStatus = "FAILED"
	StatusRefunded  TransactionStatus = "REFUNDED"

	EntryCredit EntryType = "CREDIT"
	EntryDebit  EntryType = "DEBIT"
)

// Account represents a user's wallet or a merchant's account
type Account struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Balance   int64     `json:"balance"`  // Store as cents
	Currency  string    `json:"currency"` // e.g. "USD"
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Transaction represents an atomic money movement operation
type Transaction struct {
	ID             uuid.UUID         `json:"id"`
	OrderID        uuid.UUID         `json:"order_id"`
	Amount         int64             `json:"amount"`
	Currency       string            `json:"currency"`
	Status         TransactionStatus `json:"status"`
	IdempotencyKey string            `json:"idempotency_key"`
	ErrorMessage   string            `json:"error_message,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// LedgerEntry is the double-entry bookkeeping log
type LedgerEntry struct {
	ID            uuid.UUID `json:"id"`
	TransactionID uuid.UUID `json:"transaction_id"`
	AccountID     uuid.UUID `json:"account_id"`
	Amount        int64     `json:"amount"`
	Type          EntryType `json:"type"` // CREDIT or DEBIT
	CreatedAt     time.Time `json:"created_at"`
}

type ChargeRequest struct {
	OrderID uuid.UUID `json:"order_id"`
	Amount  int64     `json:"amount"`
}

type ChargeResponse struct {
	TransactionID uuid.UUID         `json:"transaction_id"`
	Status        TransactionStatus `json:"status"`
}
