package repository

import (
	"context"
	"errors"
	"payment-service/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PaymentRepository interface {
	CheckIdempotency(ctx context.Context, key string) (*models.Transaction, error)
	GetAccountByUserID(ctx context.Context, userID uuid.UUID) (*models.Account, error)
	GetMerchantAccount(ctx context.Context) (*models.Account, error)
	ProcessPayment(ctx context.Context, tx *models.Transaction, srcAccountID uuid.UUID, destAccountID uuid.UUID) error
	CreateTransaction(ctx context.Context, tx *models.Transaction) error
	UpdateTransactionStatus(ctx context.Context, id uuid.UUID, status models.TransactionStatus, errMsg string) error
}

type paymentRepository struct {
	db *pgxpool.Pool
}

func NewPaymentRepository(db *pgxpool.Pool) PaymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) CheckIdempotency(ctx context.Context, key string) (*models.Transaction, error) {
	query := `
		SELECT id, order_id, amount, currency, status, idempotency_key, error_message, created_at, updated_at
		FROM transactions
		WHERE idempotency_key = $1
	`
	row := r.db.QueryRow(ctx, query, key)

	var tx models.Transaction
	var errMsg *string
	err := row.Scan(&tx.ID, &tx.OrderID, &tx.Amount, &tx.Currency, &tx.Status, &tx.IdempotencyKey, &errMsg, &tx.CreatedAt, &tx.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Not found
		}
		return nil, err
	}
	if errMsg != nil {
		tx.ErrorMessage = *errMsg
	}
	return &tx, nil
}

func (r *paymentRepository) GetAccountByUserID(ctx context.Context, userID uuid.UUID) (*models.Account, error) {
	query := `SELECT id, user_id, balance, currency, created_at, updated_at FROM accounts WHERE user_id = $1`
	row := r.db.QueryRow(ctx, query, userID)

	var acc models.Account
	err := row.Scan(&acc.ID, &acc.UserID, &acc.Balance, &acc.Currency, &acc.CreatedAt, &acc.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Not found
		}
		return nil, err
	}
	return &acc, nil
}

func (r *paymentRepository) GetMerchantAccount(ctx context.Context) (*models.Account, error) {
	// For simplicity, assuming a fixed user_id or special role for the merchant.
	// In a real system, you'd fetch the specific merchant's account.
	// We'll just fetch a single specific account assuming it's the main platform's account.
	// Let's assume user_id = '00000000-0000-0000-0000-000000000000' is the platform.
	platformID := uuid.Nil
	query := `SELECT id, user_id, balance, currency, created_at, updated_at FROM accounts WHERE user_id = $1`
	row := r.db.QueryRow(ctx, query, platformID)

	var acc models.Account
	err := row.Scan(&acc.ID, &acc.UserID, &acc.Balance, &acc.Currency, &acc.CreatedAt, &acc.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Create it if it doesn't exist for testing purposes
			return nil, errors.New("merchant account not found")
		}
		return nil, err
	}
	return &acc, nil
}

func (r *paymentRepository) CreateTransaction(ctx context.Context, tx *models.Transaction) error {
	query := `
		INSERT INTO transactions (id, order_id, amount, currency, status, idempotency_key, error_message, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	var errMsg *string
	if tx.ErrorMessage != "" {
		errMsg = &tx.ErrorMessage
	}
	_, err := r.db.Exec(ctx, query, tx.ID, tx.OrderID, tx.Amount, tx.Currency, tx.Status, tx.IdempotencyKey, errMsg, tx.CreatedAt, tx.UpdatedAt)
	return err
}

func (r *paymentRepository) UpdateTransactionStatus(ctx context.Context, id uuid.UUID, status models.TransactionStatus, errMsgStr string) error {
	query := `
		UPDATE transactions
		SET status = $1, error_message = $2, updated_at = NOW()
		WHERE id = $3
	`
	var errMsg *string
	if errMsgStr != "" {
		errMsg = &errMsgStr
	}
	_, err := r.db.Exec(ctx, query, status, errMsg, id)
	return err
}

func (r *paymentRepository) ProcessPayment(ctx context.Context, tx *models.Transaction, srcAccountID uuid.UUID, destAccountID uuid.UUID) error {
	// Execute a database transaction for double-entry bookkeeping
	dbTx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer dbTx.Rollback(ctx)

	// 1. Debit Source Wallet
	var srcBalance int64
	err = dbTx.QueryRow(ctx, `SELECT balance FROM accounts WHERE id = $1 FOR UPDATE`, srcAccountID).Scan(&srcBalance)
	if err != nil {
		return err
	}
	if srcBalance < tx.Amount {
		return errors.New("insufficient funds")
	}

	_, err = dbTx.Exec(ctx, `UPDATE accounts SET balance = balance - $1, updated_at = NOW() WHERE id = $2`, tx.Amount, srcAccountID)
	if err != nil {
		return err
	}

	// 2. Credit Destination Wallet (Merchant)
	_, err = dbTx.Exec(ctx, `UPDATE accounts SET balance = balance + $1, updated_at = NOW() WHERE id = $2`, tx.Amount, destAccountID)
	if err != nil {
		return err
	}

	// 3. Insert Ledger Entries
	debitID, creditID := uuid.New(), uuid.New()
	_, err = dbTx.Exec(ctx, `
		INSERT INTO ledger_entries (id, transaction_id, account_id, amount, type, created_at)
		VALUES ($1, $2, $3, $4, 'DEBIT', NOW())
	`, debitID, tx.ID, srcAccountID, tx.Amount)
	if err != nil {
		return err
	}

	_, err = dbTx.Exec(ctx, `
		INSERT INTO ledger_entries (id, transaction_id, account_id, amount, type, created_at)
		VALUES ($1, $2, $3, $4, 'CREDIT', NOW())
	`, creditID, tx.ID, destAccountID, tx.Amount)
	if err != nil {
		return err
	}

	// 4. Update Transaction Status
	_, err = dbTx.Exec(ctx, `
		UPDATE transactions
		SET status = 'SUCCEEDED', updated_at = NOW()
		WHERE id = $1
	`, tx.ID)
	if err != nil {
		return err
	}

	return dbTx.Commit(ctx)
}
