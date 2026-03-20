-- User Service Tables
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

-- Order Service Tables
CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    amount BIGINT NOT NULL,
    currency VARCHAR(10) NOT NULL,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

-- Payment Service Tables
CREATE TABLE IF NOT EXISTS accounts (
    id UUID PRIMARY KEY,
    user_id UUID UNIQUE NOT NULL,
    balance BIGINT NOT NULL,
    currency VARCHAR(10) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL,
    amount BIGINT NOT NULL,
    currency VARCHAR(10) NOT NULL,
    status VARCHAR(50) NOT NULL,
    idempotency_key VARCHAR(255) UNIQUE NOT NULL,
    error_message TEXT,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS ledger_entries (
    id UUID PRIMARY KEY,
    transaction_id UUID NOT NULL,
    account_id UUID NOT NULL,
    amount BIGINT NOT NULL,
    type VARCHAR(20) NOT NULL, -- 'CREDIT' or 'DEBIT'
    created_at TIMESTAMP NOT NULL
);

-- Analytics Service Tables
CREATE TABLE IF NOT EXISTS daily_metrics (
    metric_date TIMESTAMP PRIMARY KEY,
    revenue_cents BIGINT NOT NULL DEFAULT 0,
    failed_payments BIGINT NOT NULL DEFAULT 0,
    order_volume BIGINT NOT NULL DEFAULT 0
);

-- Note: In a real system, you'd seed specific data dynamically upon User registration, 
-- but for testing the Double-Entry ledger, we'll quickly seed a merchant platform account.
INSERT INTO accounts (id, user_id, balance, currency, created_at, updated_at)
VALUES (
    '00000000-0000-0000-0000-000000000000', -- internal ID
    '00000000-0000-0000-0000-000000000000', -- Platform "user_id"
    0,
    'USD',
    NOW(),
    NOW()
) ON CONFLICT (user_id) DO NOTHING;
