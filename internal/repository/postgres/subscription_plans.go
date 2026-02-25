// internal/repository/postgres/subscription_plan_repository.go
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

	"github.com/jackc/pgx/v5/pgxpool"
)

type SubscriptionPlanRepository struct {
	db *pgxpool.Pool
}

func NewSubscriptionPlanRepository(db *pgxpool.Pool) *SubscriptionPlanRepository {
	return &SubscriptionPlanRepository{db: db}
}

// Create creates a new subscription plan
func (r *SubscriptionPlanRepository) Create(ctx context.Context, plan *subscription.SubscriptionPlan) error {
	query := `
		INSERT INTO subscription_plans (
			plan_code, name, description, price, currency, setup_fee,
			billing_usage, billing_cycle, overage_charge,
			max_offers, max_customers, features,
			status, is_public, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, created_at, updated_at
	`

	var featuresJSON, metadataJSON []byte
	var err error

	if plan.Features != nil {
		featuresJSON, err = json.Marshal(plan.Features)
		if err != nil {
			return fmt.Errorf("failed to marshal features: %w", err)
		}
	}

	if plan.Metadata != nil {
		metadataJSON, err = json.Marshal(plan.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	err = r.db.QueryRow(
		ctx, query,
		plan.PlanCode, plan.Name, plan.Description, plan.Price, plan.Currency, plan.SetupFee,
		plan.BillingUsage, plan.BillingCycle, plan.OverageCharge,
		plan.MaxOffers, plan.MaxCustomers, featuresJSON,
		plan.Status, plan.IsPublic, metadataJSON,
	).Scan(&plan.ID, &plan.CreatedAt, &plan.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create subscription plan: %w", err)
	}

	return nil
}

// FindByID retrieves a subscription plan by ID
func (r *SubscriptionPlanRepository) FindByID(ctx context.Context, id int64) (*subscription.SubscriptionPlan, error) {
	query := `
		SELECT id, plan_code, name, description, price, currency, setup_fee,
		       billing_usage, billing_cycle, overage_charge,
		       max_offers, max_customers, features,
		       status, is_public, metadata, created_at, updated_at
		FROM subscription_plans
		WHERE id = $1
	`

	var plan subscription.SubscriptionPlan
	var featuresJSON, metadataJSON []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
		&plan.ID, &plan.PlanCode, &plan.Name, &plan.Description, &plan.Price, &plan.Currency, &plan.SetupFee,
		&plan.BillingUsage, &plan.BillingCycle, &plan.OverageCharge,
		&plan.MaxOffers, &plan.MaxCustomers, &featuresJSON,
		&plan.Status, &plan.IsPublic, &metadataJSON, &plan.CreatedAt, &plan.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, xerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find subscription plan: %w", err)
	}

	// Unmarshal JSON fields
	if len(featuresJSON) > 0 {
		if err := json.Unmarshal(featuresJSON, &plan.Features); err != nil {
			return nil, fmt.Errorf("failed to unmarshal features: %w", err)
		}
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &plan.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &plan, nil
}

// FindByPlanCode retrieves a subscription plan by plan code
func (r *SubscriptionPlanRepository) FindByPlanCode(ctx context.Context, planCode string) (*subscription.SubscriptionPlan, error) {
	query := `
		SELECT id, plan_code, name, description, price, currency, setup_fee,
		       billing_usage, billing_cycle, overage_charge,
		       max_offers, max_customers, features,
		       status, is_public, metadata, created_at, updated_at
		FROM subscription_plans
		WHERE plan_code = $1
	`

	var plan subscription.SubscriptionPlan
	var featuresJSON, metadataJSON []byte

	err := r.db.QueryRow(ctx, query, planCode).Scan(
		&plan.ID, &plan.PlanCode, &plan.Name, &plan.Description, &plan.Price, &plan.Currency, &plan.SetupFee,
		&plan.BillingUsage, &plan.BillingCycle, &plan.OverageCharge,
		&plan.MaxOffers, &plan.MaxCustomers, &featuresJSON,
		&plan.Status, &plan.IsPublic, &metadataJSON, &plan.CreatedAt, &plan.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, xerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find subscription plan: %w", err)
	}

	// Unmarshal JSON fields
	if len(featuresJSON) > 0 {
		if err := json.Unmarshal(featuresJSON, &plan.Features); err != nil {
			return nil, fmt.Errorf("failed to unmarshal features: %w", err)
		}
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &plan.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &plan, nil
}

// Update updates a subscription plan
func (r *SubscriptionPlanRepository) Update(ctx context.Context, id int64, plan *subscription.SubscriptionPlan) error {
	query := `
		UPDATE subscription_plans
		SET name = $1, description = $2, price = $3, setup_fee = $4,
		    billing_usage = $5, billing_cycle = $6, overage_charge = $7,
		    max_offers = $8, max_customers = $9, features = $10,
		    is_public = $11, metadata = $12, updated_at = $13
		WHERE id = $14
	`

	var featuresJSON, metadataJSON []byte
	var err error

	if plan.Features != nil {
		featuresJSON, err = json.Marshal(plan.Features)
		if err != nil {
			return fmt.Errorf("failed to marshal features: %w", err)
		}
	}

	if plan.Metadata != nil {
		metadataJSON, err = json.Marshal(plan.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	result, err := r.db.Exec(
		ctx, query,
		plan.Name, plan.Description, plan.Price, plan.SetupFee,
		plan.BillingUsage, plan.BillingCycle, plan.OverageCharge,
		plan.MaxOffers, plan.MaxCustomers, featuresJSON,
		plan.IsPublic, metadataJSON, time.Now(), id,
	)

	if err != nil {
		return fmt.Errorf("failed to update subscription plan: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// UpdateStatus updates the status of a subscription plan
func (r *SubscriptionPlanRepository) UpdateStatus(ctx context.Context, id int64, status subscription.SubscriptionStatus) error {
	query := `UPDATE subscription_plans SET status = $1, updated_at = $2 WHERE id = $3`

	result, err := r.db.Exec(ctx, query, status, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update plan status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// Delete deletes a subscription plan
func (r *SubscriptionPlanRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM subscription_plans WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete subscription plan: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// List retrieves subscription plans with filters
func (r *SubscriptionPlanRepository) List(ctx context.Context, filters *subscription.PlanListFilters) ([]subscription.SubscriptionPlan, int64, error) {
	// Build WHERE clause
	conditions := []string{}
	args := []interface{}{}
	argPos := 1

	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argPos))
		args = append(args, *filters.Status)
		argPos++
	}

	if filters.IsPublic != nil {
		conditions = append(conditions, fmt.Sprintf("is_public = $%d", argPos))
		args = append(args, *filters.IsPublic)
		argPos++
	}

	if filters.MinPrice != nil {
		conditions = append(conditions, fmt.Sprintf("price >= $%d", argPos))
		args = append(args, *filters.MinPrice)
		argPos++
	}

	if filters.MaxPrice != nil {
		conditions = append(conditions, fmt.Sprintf("price <= $%d", argPos))
		args = append(args, *filters.MaxPrice)
		argPos++
	}

	if filters.BillingCycle != nil {
		conditions = append(conditions, fmt.Sprintf("billing_cycle = $%d", argPos))
		args = append(args, *filters.BillingCycle)
		argPos++
	}

	if filters.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d OR plan_code ILIKE $%d)", argPos, argPos, argPos))
		args = append(args, "%"+filters.Search+"%")
		argPos++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM subscription_plans %s", whereClause)
	var total int64
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count plans: %w", err)
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

	// Query plans
	query := fmt.Sprintf(`
		SELECT id, plan_code, name, description, price, currency, setup_fee,
		       billing_usage, billing_cycle, overage_charge,
		       max_offers, max_customers, features,
		       status, is_public, metadata, created_at, updated_at
		FROM subscription_plans
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortBy, sortOrder, argPos, argPos+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list plans: %w", err)
	}
	defer rows.Close()

	plans := []subscription.SubscriptionPlan{}
	for rows.Next() {
		var plan subscription.SubscriptionPlan
		var featuresJSON, metadataJSON []byte

		err := rows.Scan(
			&plan.ID, &plan.PlanCode, &plan.Name, &plan.Description, &plan.Price, &plan.Currency, &plan.SetupFee,
			&plan.BillingUsage, &plan.BillingCycle, &plan.OverageCharge,
			&plan.MaxOffers, &plan.MaxCustomers, &featuresJSON,
			&plan.Status, &plan.IsPublic, &metadataJSON, &plan.CreatedAt, &plan.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan plan: %w", err)
		}

		// Unmarshal JSON fields
		if len(featuresJSON) > 0 {
			json.Unmarshal(featuresJSON, &plan.Features)
		}
		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &plan.Metadata)
		}

		plans = append(plans, plan)
	}

	return plans, total, nil
}

// GetStats retrieves statistics about subscription plans
func (r *SubscriptionPlanRepository) GetStats(ctx context.Context) (*subscription.SubscriptionPlanStats, error) {
	query := `
		SELECT 
			COUNT(*) as total_plans,
			COUNT(CASE WHEN status = 'active' THEN 1 END) as active_plans,
			COUNT(CASE WHEN status = 'inactive' THEN 1 END) as inactive_plans,
			COALESCE(AVG(price), 0) as average_price,
			(SELECT COUNT(*) FROM agent_subscriptions WHERE status = 'active') as total_subscribers
		FROM subscription_plans
	`

	var stats subscription.SubscriptionPlanStats
	err := r.db.QueryRow(ctx, query).Scan(
		&stats.TotalPlans,
		&stats.ActivePlans,
		&stats.InactivePlans,
		&stats.AveragePrice,
		&stats.TotalSubscribers,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	// Get most popular plan
	popularQuery := `
		SELECT sp.id, sp.name, COUNT(as2.id) as subscriber_count
		FROM subscription_plans sp
		LEFT JOIN agent_subscriptions as2 ON sp.id = as2.subscription_plan_id
		WHERE as2.status = 'active'
		GROUP BY sp.id, sp.name
		ORDER BY subscriber_count DESC
		LIMIT 1
	`

	err = r.db.QueryRow(ctx, popularQuery).Scan(&stats.MostPopularPlanID, &stats.MostPopularPlanName, &sql.NullInt64{})
	if err != nil && err != sql.ErrNoRows {
		// Don't fail if no popular plan found
		return &stats, nil
	}

	return &stats, nil
}

// ExistsByPlanCode checks if a plan with the given code exists
func (r *SubscriptionPlanRepository) ExistsByPlanCode(ctx context.Context, planCode string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM subscription_plans WHERE plan_code = $1)`
	var exists bool
	err := r.db.QueryRow(ctx, query, planCode).Scan(&exists)
	return exists, err
}