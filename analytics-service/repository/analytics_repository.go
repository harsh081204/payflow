package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AnalyticsRepository interface {
	RecordDailyRevenue(ctx context.Context, date time.Time, amount int64) error
	RecordFailedPayment(ctx context.Context, date time.Time) error
	RecordOrderVolume(ctx context.Context, date time.Time) error
	GetMetrics(ctx context.Context) (map[string]interface{}, error)
}

type analyticsRepository struct {
	db *pgxpool.Pool
}

func NewAnalyticsRepository(db *pgxpool.Pool) AnalyticsRepository {
	return &analyticsRepository{db: db}
}

func (r *analyticsRepository) RecordDailyRevenue(ctx context.Context, date time.Time, amount int64) error {
	query := `
		INSERT INTO daily_metrics (metric_date, revenue_cents, failed_payments, order_volume)
		VALUES ($1, $2, 0, 0)
		ON CONFLICT (metric_date) 
		DO UPDATE SET revenue_cents = daily_metrics.revenue_cents + EXCLUDED.revenue_cents
	`
	_, err := r.db.Exec(ctx, query, date.Truncate(24*time.Hour), amount)
	return err
}

func (r *analyticsRepository) RecordFailedPayment(ctx context.Context, date time.Time) error {
	query := `
		INSERT INTO daily_metrics (metric_date, revenue_cents, failed_payments, order_volume)
		VALUES ($1, 0, 1, 0)
		ON CONFLICT (metric_date) 
		DO UPDATE SET failed_payments = daily_metrics.failed_payments + 1
	`
	_, err := r.db.Exec(ctx, query, date.Truncate(24*time.Hour))
	return err
}

func (r *analyticsRepository) RecordOrderVolume(ctx context.Context, date time.Time) error {
	query := `
		INSERT INTO daily_metrics (metric_date, revenue_cents, failed_payments, order_volume)
		VALUES ($1, 0, 0, 1)
		ON CONFLICT (metric_date) 
		DO UPDATE SET order_volume = daily_metrics.order_volume + 1
	`
	_, err := r.db.Exec(ctx, query, date.Truncate(24*time.Hour))
	return err
}

func (r *analyticsRepository) GetMetrics(ctx context.Context) (map[string]interface{}, error) {
	query := `
		SELECT 
			COALESCE(SUM(revenue_cents), 0) as total_revenue,
			COALESCE(SUM(failed_payments), 0) as total_failed,
			COALESCE(SUM(order_volume), 0) as total_orders
		FROM daily_metrics
	`
	var totalRevenue, totalFailed, totalOrders int64
	err := r.db.QueryRow(ctx, query).Scan(&totalRevenue, &totalFailed, &totalOrders)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_revenue_cents":   totalRevenue,
		"total_failed_payments": totalFailed,
		"total_order_volume":    totalOrders,
	}, nil
}
