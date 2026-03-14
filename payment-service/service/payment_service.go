package service

import (
	"context"
	"errors"
	"log/slog"
	"payment-service/models"
	"payment-service/repository"
	"time"

	"github.com/google/uuid"
)

var (
	ErrDuplicateRequest = errors.New("duplicate request: idempotency key already used")
	ErrAccountNotFound  = errors.New("account not found")
	ErrInsufficientFund = errors.New("insufficient funds")
)

type PaymentService interface {
	Charge(ctx context.Context, userID uuid.UUID, req *models.ChargeRequest, idempotencyKey string) (*models.ChargeResponse, error)
}

type paymentService struct {
	repo repository.PaymentRepository
}

func NewPaymentService(repo repository.PaymentRepository) PaymentService {
	return &paymentService{repo: repo}
}

func (s *paymentService) Charge(ctx context.Context, userID uuid.UUID, req *models.ChargeRequest, idempotencyKey string) (*models.ChargeResponse, error) {
	// 1. Check Idempotency
	if idempotencyKey != "" {
		existingTx, err := s.repo.CheckIdempotency(ctx, idempotencyKey)
		if err != nil {
			return nil, err
		}
		if existingTx != nil {
			slog.Info("Idempotency key hit", "key", idempotencyKey)
			return &models.ChargeResponse{
				TransactionID: existingTx.ID,
				Status:        existingTx.Status,
			}, ErrDuplicateRequest
		}
	}

	// 2. Fetch Accounts
	userAcc, err := s.repo.GetAccountByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if userAcc == nil {
		return nil, ErrAccountNotFound
	}

	merchantAcc, err := s.repo.GetMerchantAccount(ctx)
	if err != nil {
		// Mock dynamic creation of merchant if missing (for demo purposes)
		// Usually you'd fail here.
		slog.Warn("Merchant account not found, payment would fail in production")
		return nil, errors.New("merchant account missing")
	}

	// 3. Create Pending Transaction
	tx := &models.Transaction{
		ID:             uuid.New(),
		OrderID:        req.OrderID,
		Amount:         req.Amount,
		Currency:       userAcc.Currency,
		Status:         models.StatusPending,
		IdempotencyKey: idempotencyKey,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.repo.CreateTransaction(ctx, tx); err != nil {
		return nil, err
	}

	// 4. Process Payment (ACID Transaction)
	err = s.repo.ProcessPayment(ctx, tx, userAcc.ID, merchantAcc.ID)
	if err != nil {
		errStr := err.Error()
		slog.Error("Payment processing failed", "transaction_id", tx.ID, "error", errStr)

		updErr := s.repo.UpdateTransactionStatus(ctx, tx.ID, models.StatusFailed, errStr)
		if updErr != nil {
			slog.Error("Failed to update transaction status to FAILED", "transaction_id", tx.ID, "error", updErr)
		}

		if errStr == "insufficient funds" {
			return &models.ChargeResponse{TransactionID: tx.ID, Status: models.StatusFailed}, ErrInsufficientFund
		}
		return &models.ChargeResponse{TransactionID: tx.ID, Status: models.StatusFailed}, err
	}

	// Success
	return &models.ChargeResponse{
		TransactionID: tx.ID,
		Status:        models.StatusSucceeded,
	}, nil
}
