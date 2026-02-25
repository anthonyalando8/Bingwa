// internal/repository/postgres/promotional_campaign_repository.go
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"bingwa-service/internal/domain/campaign"
	xerrors "bingwa-service/internal/pkg/errors"

	"github.com/jackc/pgx/v5/pgxpool"
	//"github.com/lib/pq"
)

type PromotionalCampaignRepository struct {
	db *pgxpool.Pool
}

func NewPromotionalCampaignRepository(db *pgxpool.Pool) *PromotionalCampaignRepository {
	return &PromotionalCampaignRepository{db: db}
}

// Create creates a new promotional campaign
func (r *PromotionalCampaignRepository) Create(ctx context.Context, c *campaign.PromotionalCampaign) error {
	query := `
		INSERT INTO promotional_campaigns (
			campaign_code, name, description, promotional_code,
			discount_type, discount_value, max_discount_amount,
			start_date, end_date, max_uses, uses_per_user,
			applicable_plans, target_user_types, status, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, created_at, updated_at
	`

	var metadataJSON []byte
	var err error

	if c.Metadata != nil {
		metadataJSON, err = json.Marshal(c.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	err = r.db.QueryRow(
		ctx, query,
		c.CampaignCode, c.Name, c.Description, c.PromotionalCode,
		c.DiscountType, c.DiscountValue, c.MaxDiscountAmount,
		c.StartDate, c.EndDate, c.MaxUses, c.UsesPerUser,
		c.ApplicablePlans, c.TargetUserTypes, c.Status, metadataJSON,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create campaign: %w", err)
	}

	return nil
}

// FindByID retrieves a campaign by ID
func (r *PromotionalCampaignRepository) FindByID(ctx context.Context, id int64) (*campaign.PromotionalCampaign, error) {
	query := `
		SELECT id, campaign_code, name, description, promotional_code,
		       discount_type, discount_value, max_discount_amount,
		       start_date, end_date, max_uses, uses_per_user, current_uses,
		       applicable_plans, target_user_types, status, metadata,
		       created_at, updated_at
		FROM promotional_campaigns
		WHERE id = $1
	`

	var c campaign.PromotionalCampaign
	var metadataJSON []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.CampaignCode, &c.Name, &c.Description, &c.PromotionalCode,
		&c.DiscountType, &c.DiscountValue, &c.MaxDiscountAmount,
		&c.StartDate, &c.EndDate, &c.MaxUses, &c.UsesPerUser, &c.CurrentUses,
		&c.ApplicablePlans, &c.TargetUserTypes, &c.Status, &metadataJSON,
		&c.CreatedAt, &c.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, xerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find campaign: %w", err)
	}

	// Unmarshal metadata
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &c.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &c, nil
}

// FindByPromotionalCode retrieves a campaign by promotional code
func (r *PromotionalCampaignRepository) FindByPromotionalCode(ctx context.Context, promoCode string) (*campaign.PromotionalCampaign, error) {
	query := `
		SELECT id, campaign_code, name, description, promotional_code,
		       discount_type, discount_value, max_discount_amount,
		       start_date, end_date, max_uses, uses_per_user, current_uses,
		       applicable_plans, target_user_types, status, metadata,
		       created_at, updated_at
		FROM promotional_campaigns
		WHERE promotional_code = $1
	`

	var c campaign.PromotionalCampaign
	var metadataJSON []byte

	err := r.db.QueryRow(ctx, query, promoCode).Scan(
		&c.ID, &c.CampaignCode, &c.Name, &c.Description, &c.PromotionalCode,
		&c.DiscountType, &c.DiscountValue, &c.MaxDiscountAmount,
		&c.StartDate, &c.EndDate, &c.MaxUses, &c.UsesPerUser, &c.CurrentUses,
		&c.ApplicablePlans, &c.TargetUserTypes, &c.Status, &metadataJSON,
		&c.CreatedAt, &c.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, xerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find campaign: %w", err)
	}

	// Unmarshal metadata
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &c.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &c, nil
}

// Update updates a campaign
func (r *PromotionalCampaignRepository) Update(ctx context.Context, id int64, c *campaign.PromotionalCampaign) error {
	query := `
		UPDATE promotional_campaigns
		SET name = $1, description = $2, discount_value = $3, max_discount_amount = $4,
		    start_date = $5, end_date = $6, max_uses = $7, uses_per_user = $8,
		    applicable_plans = $9, target_user_types = $10, metadata = $11, updated_at = $12
		WHERE id = $13
	`

	var metadataJSON []byte
	var err error

	if c.Metadata != nil {
		metadataJSON, err = json.Marshal(c.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	result, err := r.db.Exec(
		ctx, query,
		c.Name, c.Description, c.DiscountValue, c.MaxDiscountAmount,
		c.StartDate, c.EndDate, c.MaxUses, c.UsesPerUser,
		c.ApplicablePlans, c.TargetUserTypes, metadataJSON, time.Now(), id,
	)

	if err != nil {
		return fmt.Errorf("failed to update campaign: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// UpdateStatus updates campaign status
func (r *PromotionalCampaignRepository) UpdateStatus(ctx context.Context, id int64, status campaign.CampaignStatus) error {
	query := `UPDATE promotional_campaigns SET status = $1, updated_at = $2 WHERE id = $3`

	result, err := r.db.Exec(ctx, query, status, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// IncrementUses increments the current_uses counter
func (r *PromotionalCampaignRepository) IncrementUses(ctx context.Context, id int64) error {
	query := `UPDATE promotional_campaigns SET current_uses = current_uses + 1, updated_at = $1 WHERE id = $2`

	result, err := r.db.Exec(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to increment uses: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// Delete deletes a campaign
func (r *PromotionalCampaignRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM promotional_campaigns WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete campaign: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// List retrieves campaigns with filters
func (r *PromotionalCampaignRepository) List(ctx context.Context, filters *campaign.CampaignListFilters) ([]campaign.PromotionalCampaign, int64, error) {
	// Build WHERE clause
	conditions := []string{}
	args := []interface{}{}
	argPos := 1

	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argPos))
		args = append(args, *filters.Status)
		argPos++
	}

	if filters.DiscountType != nil {
		conditions = append(conditions, fmt.Sprintf("discount_type = $%d", argPos))
		args = append(args, *filters.DiscountType)
		argPos++
	}

	if filters.IsActive != nil && *filters.IsActive {
		conditions = append(conditions, fmt.Sprintf("status = 'active' AND start_date <= $%d AND end_date >= $%d", argPos, argPos))
		now := time.Now()
		args = append(args, now)
		argPos++
	}

	if filters.Search != "" {
		conditions = append(conditions, fmt.Sprintf(
			"(name ILIKE $%d OR description ILIKE $%d OR promotional_code ILIKE $%d)",
			argPos, argPos, argPos,
		))
		args = append(args, "%"+filters.Search+"%")
		argPos++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM promotional_campaigns %s", whereClause)
	var total int64
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count campaigns: %w", err)
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

	// Query campaigns
	query := fmt.Sprintf(`
		SELECT id, campaign_code, name, description, promotional_code,
		       discount_type, discount_value, max_discount_amount,
		       start_date, end_date, max_uses, uses_per_user, current_uses,
		       applicable_plans, target_user_types, status, metadata,
		       created_at, updated_at
		FROM promotional_campaigns
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortBy, sortOrder, argPos, argPos+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list campaigns: %w", err)
	}
	defer rows.Close()

	campaigns := []campaign.PromotionalCampaign{}
	for rows.Next() {
		var c campaign.PromotionalCampaign
		var metadataJSON []byte

		err := rows.Scan(
			&c.ID, &c.CampaignCode, &c.Name, &c.Description, &c.PromotionalCode,
			&c.DiscountType, &c.DiscountValue, &c.MaxDiscountAmount,
			&c.StartDate, &c.EndDate, &c.MaxUses, &c.UsesPerUser, &c.CurrentUses,
			&c.ApplicablePlans, &c.TargetUserTypes, &c.Status, &metadataJSON,
			&c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan campaign: %w", err)
		}

		// Unmarshal metadata
		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &c.Metadata)
		}

		campaigns = append(campaigns, c)
	}

	return campaigns, total, nil
}

// GetActiveCampaigns retrieves currently active campaigns
func (r *PromotionalCampaignRepository) GetActiveCampaigns(ctx context.Context) ([]campaign.PromotionalCampaign, error) {
	query := `
		SELECT id, campaign_code, name, description, promotional_code,
		       discount_type, discount_value, max_discount_amount,
		       start_date, end_date, max_uses, uses_per_user, current_uses,
		       applicable_plans, target_user_types, status, metadata,
		       created_at, updated_at
		FROM promotional_campaigns
		WHERE status = 'active' AND start_date <= $1 AND end_date >= $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to get active campaigns: %w", err)
	}
	defer rows.Close()

	campaigns := []campaign.PromotionalCampaign{}
	for rows.Next() {
		var c campaign.PromotionalCampaign
		var metadataJSON []byte

		err := rows.Scan(
			&c.ID, &c.CampaignCode, &c.Name, &c.Description, &c.PromotionalCode,
			&c.DiscountType, &c.DiscountValue, &c.MaxDiscountAmount,
			&c.StartDate, &c.EndDate, &c.MaxUses, &c.UsesPerUser, &c.CurrentUses,
			&c.ApplicablePlans, &c.TargetUserTypes, &c.Status, &metadataJSON,
			&c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan campaign: %w", err)
		}

		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &c.Metadata)
		}

		campaigns = append(campaigns, c)
	}

	return campaigns, nil
}

// GetStats retrieves campaign statistics
func (r *PromotionalCampaignRepository) GetStats(ctx context.Context) (*campaign.CampaignStats, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'active' AND start_date <= NOW() AND end_date >= NOW() THEN 1 END) as active,
			COUNT(CASE WHEN status = 'expired' OR end_date < NOW() THEN 1 END) as expired,
			COALESCE(SUM(current_uses), 0) as total_uses
		FROM promotional_campaigns
	`

	var stats campaign.CampaignStats
	err := r.db.QueryRow(ctx, query).Scan(
		&stats.TotalCampaigns,
		&stats.ActiveCampaigns,
		&stats.ExpiredCampaigns,
		&stats.TotalUses,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	// TODO: Calculate total discount given from agent_subscriptions table when implemented
	stats.TotalDiscount = 0

	return &stats, nil
}

// ExistsByPromotionalCode checks if promotional code exists
func (r *PromotionalCampaignRepository) ExistsByPromotionalCode(ctx context.Context, promoCode string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM promotional_campaigns WHERE promotional_code = $1)`
	var exists bool
	err := r.db.QueryRow(ctx, query, promoCode).Scan(&exists)
	return exists, err
}

// ExistsByCampaignCode checks if campaign code exists
func (r *PromotionalCampaignRepository) ExistsByCampaignCode(ctx context.Context, campaignCode string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM promotional_campaigns WHERE campaign_code = $1)`
	var exists bool
	err := r.db.QueryRow(ctx, query, campaignCode).Scan(&exists)
	return exists, err
}