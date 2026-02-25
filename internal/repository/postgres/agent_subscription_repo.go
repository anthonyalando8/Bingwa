// internal/repository/postgres/agent_subscription_repository.go
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"bingwa-service/internal/domain/subscription"
	xerrors "bingwa-service/internal/pkg/errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AgentSubscriptionRepository struct {
	db *pgxpool.Pool
}

func NewAgentSubscriptionRepository(db *pgxpool.Pool) *AgentSubscriptionRepository {
	return &AgentSubscriptionRepository{db: db}
}

// CreateWithTx creates a subscription within a transaction
func (r *AgentSubscriptionRepository) CreateWithTx(ctx context.Context, tx pgx.Tx, sub *subscription.AgentSubscription) error {
	query := `
		INSERT INTO agent_subscriptions (
			subscription_reference, agent_identity_id, subscription_plan_id, promotional_campaign_id,
			start_date, end_date, current_period_start, current_period_end,
			auto_renew, next_billing_date, requests_limit,
			plan_price, discount_applied, amount_paid, currency,
			status, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id, created_at, updated_at
	`

	var metadataJSON []byte
	var err error

	if sub.Metadata != nil {
		metadataJSON, err = json.Marshal(sub.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	err = tx.QueryRow(
		ctx, query,
		sub.SubscriptionReference, sub.AgentIdentityID, sub.SubscriptionPlanID, sub.PromotionalCampaignID,
		sub.StartDate, sub.EndDate, sub.CurrentPeriodStart, sub.CurrentPeriodEnd,
		sub.AutoRenew, sub.NextBillingDate, sub.RequestsLimit,
		sub.PlanPrice, sub.DiscountApplied, sub.AmountPaid, sub.Currency,
		sub.Status, metadataJSON,
	).Scan(&sub.ID, &sub.CreatedAt, &sub.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create subscription: %w", err)
	}

	return nil
}

// FindByID retrieves a subscription by ID
func (r *AgentSubscriptionRepository) FindByID(ctx context.Context, id int64) (*subscription.AgentSubscription, error) {
	query := `
		SELECT id, subscription_reference, agent_identity_id, subscription_plan_id, promotional_campaign_id,
		       start_date, end_date, current_period_start, current_period_end,
		       auto_renew, renewal_count, next_billing_date,
		       requests_used, requests_limit,
		       plan_price, discount_applied, amount_paid, currency,
		       status, cancelled_at, cancellation_reason,
		       metadata, created_at, updated_at
		FROM agent_subscriptions
		WHERE id = $1
	`

	var sub subscription.AgentSubscription
	var metadataJSON []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
		&sub.ID, &sub.SubscriptionReference, &sub.AgentIdentityID, &sub.SubscriptionPlanID, &sub.PromotionalCampaignID,
		&sub.StartDate, &sub.EndDate, &sub.CurrentPeriodStart, &sub.CurrentPeriodEnd,
		&sub.AutoRenew, &sub.RenewalCount, &sub.NextBillingDate,
		&sub.RequestsUsed, &sub.RequestsLimit,
		&sub.PlanPrice, &sub.DiscountApplied, &sub.AmountPaid, &sub.Currency,
		&sub.Status, &sub.CancelledAt, &sub.CancellationReason,
		&metadataJSON, &sub.CreatedAt, &sub.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, xerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find subscription: %w", err)
	}

	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &sub.Metadata)
	}

	return &sub, nil
}

// FindActiveByAgent retrieves active subscription for an agent
func (r *AgentSubscriptionRepository) FindActiveByAgent(ctx context.Context, agentID int64) (*subscription.AgentSubscription, error) {
	query := `
		SELECT id, subscription_reference, agent_identity_id, subscription_plan_id, promotional_campaign_id,
		       start_date, end_date, current_period_start, current_period_end,
		       auto_renew, renewal_count, next_billing_date,
		       requests_used, requests_limit,
		       plan_price, discount_applied, amount_paid, currency,
		       status, cancelled_at, cancellation_reason,
		       metadata, created_at, updated_at
		FROM agent_subscriptions
		WHERE agent_identity_id = $1 AND status = 'active' AND current_period_end > NOW()
		ORDER BY current_period_end DESC
		LIMIT 1
	`

	var sub subscription.AgentSubscription
	var metadataJSON []byte

	err := r.db.QueryRow(ctx, query, agentID).Scan(
		&sub.ID, &sub.SubscriptionReference, &sub.AgentIdentityID, &sub.SubscriptionPlanID, &sub.PromotionalCampaignID,
		&sub.StartDate, &sub.EndDate, &sub.CurrentPeriodStart, &sub.CurrentPeriodEnd,
		&sub.AutoRenew, &sub.RenewalCount, &sub.NextBillingDate,
		&sub.RequestsUsed, &sub.RequestsLimit,
		&sub.PlanPrice, &sub.DiscountApplied, &sub.AmountPaid, &sub.Currency,
		&sub.Status, &sub.CancelledAt, &sub.CancellationReason,
		&metadataJSON, &sub.CreatedAt, &sub.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, xerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find active subscription: %w", err)
	}

	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &sub.Metadata)
	}

	return &sub, nil
}

// Update updates a subscription
func (r *AgentSubscriptionRepository) Update(ctx context.Context, id int64, sub *subscription.AgentSubscription) error {
	query := `
		UPDATE agent_subscriptions
		SET auto_renew = $1, metadata = $2, updated_at = $3
		WHERE id = $4
	`

	var metadataJSON []byte
	var err error

	if sub.Metadata != nil {
		metadataJSON, err = json.Marshal(sub.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	result, err := r.db.Exec(ctx, query, sub.AutoRenew, metadataJSON, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// UpdateStatusWithTx updates status within a transaction
func (r *AgentSubscriptionRepository) UpdateStatusWithTx(ctx context.Context, tx pgx.Tx, id int64, status subscription.SubscriptionStatus) error {
	query := `UPDATE agent_subscriptions SET status = $1, updated_at = $2 WHERE id = $3`

	result, err := tx.Exec(ctx, query, status, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// UpdateRenewalInfoWithTx updates renewal information
func (r *AgentSubscriptionRepository) UpdateRenewalInfoWithTx(ctx context.Context, tx pgx.Tx, id int64, periodStart, periodEnd, nextBilling time.Time, renewalCount int) error {
	query := `
		UPDATE agent_subscriptions
		SET current_period_start = $1, current_period_end = $2, next_billing_date = $3, 
		    renewal_count = $4, updated_at = $5
		WHERE id = $6
	`

	result, err := tx.Exec(
		ctx, query,
		periodStart, periodEnd,
		sql.NullTime{Time: nextBilling, Valid: true},
		renewalCount, time.Now(), id,
	)

	if err != nil {
		return fmt.Errorf("failed to update renewal info: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// IncrementRequestUsage increments request usage counter
func (r *AgentSubscriptionRepository) IncrementRequestUsage(ctx context.Context, id int64) error {
	query := `UPDATE agent_subscriptions SET requests_used = requests_used + 1, updated_at = $1 WHERE id = $2`

	result, err := r.db.Exec(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to increment request usage: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// CancelSubscription cancels a subscription
func (r *AgentSubscriptionRepository) CancelSubscription(ctx context.Context, id int64, reason string, immediately bool) error {
	var query string

	if immediately {
		// Cancel immediately
		query = `
			UPDATE agent_subscriptions
			SET status = $1, cancelled_at = $2, cancellation_reason = $3, 
			    current_period_end = $2, updated_at = $4
			WHERE id = $5
		`
	} else {
		// Cancel at period end
		query = `
			UPDATE agent_subscriptions
			SET cancelled_at = $1, cancellation_reason = $2, auto_renew = FALSE, updated_at = $3
			WHERE id = $4
		`
	}

	now := time.Now()
	var err error

	var result pgconn.CommandTag
	if immediately {
		result, err = r.db.Exec(
			ctx, query,
			subscription.SubscriptionStatusCancelled, now,
			sql.NullString{String: reason, Valid: reason != ""},
			now, id,
		)
	} else {
		result, err = r.db.Exec(
			ctx, query,
			now,
			sql.NullString{String: reason, Valid: reason != ""},
			now, id,
		)
	}

	if err != nil {
		return fmt.Errorf("failed to cancel subscription: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// List retrieves subscriptions with filters
func (r *AgentSubscriptionRepository) List(ctx context.Context, agentID int64, filters *subscription.SubscriptionListFilters) ([]subscription.AgentSubscription, int64, error) {
	conditions := []string{"agent_identity_id = $1"}
	args := []interface{}{agentID}
	argPos := 2

	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argPos))
		args = append(args, *filters.Status)
		argPos++
	}

	if filters.PlanID != nil {
		conditions = append(conditions, fmt.Sprintf("subscription_plan_id = $%d", argPos))
		args = append(args, *filters.PlanID)
		argPos++
	}

	if filters.IsExpiring {
		conditions = append(conditions, fmt.Sprintf("current_period_end <= $%d AND status = 'active'", argPos))
		args = append(args, time.Now().AddDate(0, 0, 7)) // Next 7 days
		argPos++
	}

	if filters.HasCampaign != nil {
		if *filters.HasCampaign {
			conditions = append(conditions, "promotional_campaign_id IS NOT NULL")
		} else {
			conditions = append(conditions, "promotional_campaign_id IS NULL")
		}
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM agent_subscriptions WHERE %s", whereClause)
	var total int64
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count subscriptions: %w", err)
	}

	// Pagination
	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.PageSize < 1 {
		filters.PageSize = 20
	}

	offset := (filters.Page - 1) * filters.PageSize
	limit := filters.PageSize

	// Sorting
	sortBy := "created_at"
	if filters.SortBy != "" {
		sortBy = filters.SortBy
	}
	sortOrder := "DESC"
	if filters.SortOrder != "" {
		sortOrder = strings.ToUpper(filters.SortOrder)
	}

	// Query subscriptions
	query := fmt.Sprintf(`
		SELECT id, subscription_reference, agent_identity_id, subscription_plan_id, promotional_campaign_id,
		       start_date, end_date, current_period_start, current_period_end,
		       auto_renew, renewal_count, next_billing_date,
		       requests_used, requests_limit,
		       plan_price, discount_applied, amount_paid, currency,
		       status, cancelled_at, cancellation_reason,
		       metadata, created_at, updated_at
		FROM agent_subscriptions
		WHERE %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortBy, sortOrder, argPos, argPos+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list subscriptions: %w", err)
	}
	defer rows.Close()

	subscriptions := []subscription.AgentSubscription{}
	for rows.Next() {
		var sub subscription.AgentSubscription
		var metadataJSON []byte

		err := rows.Scan(
			&sub.ID, &sub.SubscriptionReference, &sub.AgentIdentityID, &sub.SubscriptionPlanID, &sub.PromotionalCampaignID,
			&sub.StartDate, &sub.EndDate, &sub.CurrentPeriodStart, &sub.CurrentPeriodEnd,
			&sub.AutoRenew, &sub.RenewalCount, &sub.NextBillingDate,
			&sub.RequestsUsed, &sub.RequestsLimit,
			&sub.PlanPrice, &sub.DiscountApplied, &sub.AmountPaid, &sub.Currency,
			&sub.Status, &sub.CancelledAt, &sub.CancellationReason,
			&metadataJSON, &sub.CreatedAt, &sub.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan subscription: %w", err)
		}

		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &sub.Metadata)
		}

		subscriptions = append(subscriptions, sub)
	}

	return subscriptions, total, nil
}

// GetStats retrieves subscription statistics
func (r *AgentSubscriptionRepository) GetStats(ctx context.Context, agentID int64) (*subscription.SubscriptionStats, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'active' THEN 1 END) as active,
			COUNT(CASE WHEN status = 'expired' THEN 1 END) as expired,
			COUNT(CASE WHEN status = 'cancelled' THEN 1 END) as cancelled,
			COALESCE(SUM(amount_paid), 0) as revenue,
			COALESCE(AVG(amount_paid), 0) as avg_value
		FROM agent_subscriptions
		WHERE agent_identity_id = $1
	`

	var stats subscription.SubscriptionStats
	err := r.db.QueryRow(ctx, query, agentID).Scan(
		&stats.TotalSubscriptions,
		&stats.ActiveSubscriptions,
		&stats.ExpiredSubscriptions,
		&stats.CancelledSubscriptions,
		&stats.TotalRevenue,
		&stats.AverageSubscriptionValue,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	return &stats, nil
}

// GetExpiringSubscriptions retrieves subscriptions expiring soon
func (r *AgentSubscriptionRepository) GetExpiringSubscriptions(ctx context.Context, days int) ([]subscription.AgentSubscription, error) {
	query := `
		SELECT id, subscription_reference, agent_identity_id, subscription_plan_id, promotional_campaign_id,
		       start_date, end_date, current_period_start, current_period_end,
		       auto_renew, renewal_count, next_billing_date,
		       requests_used, requests_limit,
		       plan_price, discount_applied, amount_paid, currency,
		       status, cancelled_at, cancellation_reason,
		       metadata, created_at, updated_at
		FROM agent_subscriptions
		WHERE status = 'active' AND current_period_end <= $1 AND current_period_end > NOW()
		ORDER BY current_period_end ASC
	`

	expiryDate := time.Now().AddDate(0, 0, days)
	rows, err := r.db.Query(ctx, query, expiryDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get expiring subscriptions: %w", err)
	}
	defer rows.Close()

	subscriptions := []subscription.AgentSubscription{}
	for rows.Next() {
		var sub subscription.AgentSubscription
		var metadataJSON []byte

		err := rows.Scan(
			&sub.ID, &sub.SubscriptionReference, &sub.AgentIdentityID, &sub.SubscriptionPlanID, &sub.PromotionalCampaignID,
			&sub.StartDate, &sub.EndDate, &sub.CurrentPeriodStart, &sub.CurrentPeriodEnd,
			&sub.AutoRenew, &sub.RenewalCount, &sub.NextBillingDate,
			&sub.RequestsUsed, &sub.RequestsLimit,
			&sub.PlanPrice, &sub.DiscountApplied, &sub.AmountPaid, &sub.Currency,
			&sub.Status, &sub.CancelledAt, &sub.CancellationReason,
			&metadataJSON, &sub.CreatedAt, &sub.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan subscription: %w", err)
		}

		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &sub.Metadata)
		}

		subscriptions = append(subscriptions, sub)
	}

	return subscriptions, nil
}

// ExistsBySubscriptionReference checks if reference exists
func (r *AgentSubscriptionRepository) ExistsBySubscriptionReference(ctx context.Context, reference string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM agent_subscriptions WHERE subscription_reference = $1)`
	var exists bool
	err := r.db.QueryRow(ctx, query, reference).Scan(&exists)
	return exists, err
}

// ResetRequestUsage resets the request usage counter (for new billing cycle)
func (r *AgentSubscriptionRepository) ResetRequestUsage(ctx context.Context, id int64) error {
	query := `UPDATE agent_subscriptions SET requests_used = 0, updated_at = $1 WHERE id = $2`

	result, err := r.db.Exec(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to reset request usage: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}