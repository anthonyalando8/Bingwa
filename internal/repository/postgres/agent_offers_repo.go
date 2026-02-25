// internal/repository/postgres/agent_offer_repository.go
package postgres

import (
	"context"
	//"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"bingwa-service/internal/domain/offer"
	xerrors "bingwa-service/internal/pkg/errors"

	//"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq"
)

type AgentOfferRepository struct {
	db           *pgxpool.Pool
	ussdCodeRepo *OfferUSSDCodeRepository
	dbWrapper    *DB
}

func NewAgentOfferRepository(db *pgxpool.Pool, ussdCodeRepo *OfferUSSDCodeRepository, dbWrapper *DB) *AgentOfferRepository {
	return &AgentOfferRepository{
		db:           db,
		ussdCodeRepo: ussdCodeRepo,
		dbWrapper:    dbWrapper,
	}
}

// scanOfferRow is a helper function to scan a single offer row
func (r *AgentOfferRepository) scanOfferRow(scanner interface {
	Scan(dest ...interface{}) error
}) (*offer.AgentOffer, error) {
	var o offer.AgentOffer
	var metadataJSON []byte
	var tags []string

	err := scanner.Scan(
		&o.ID, &o.AgentIdentityID, &o.OfferCode, &o.Name, &o.Description, &o.Type, &o.Amount, &o.Units,
		&o.Price, &o.Currency, &o.DiscountPercentage, &o.ValidityDays, &o.ValidityLabel,
		&o.USSDCodeTemplate, &o.USSDProcessingType, &o.USSDExpectedResponse, &o.USSDErrorPattern,
		&o.IsFeatured, &o.IsRecurring, &o.MaxPurchasesPerCustomer,
		&o.Status, &o.AvailableFrom, &o.AvailableUntil, pq.Array(&tags), &metadataJSON,
		&o.CreatedAt, &o.UpdatedAt, &o.DeletedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan offer row: %w", err)
	}

	// Convert []string to pq.StringArray
	o.Tags = pq.StringArray(tags)

	// Unmarshal metadata
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &o.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &o, nil
}

// loadPrimaryUSSDCode loads the primary USSD code for an offer
func (r *AgentOfferRepository) loadPrimaryUSSDCode(ctx context.Context, o *offer.AgentOffer) error {
	primaryCode, err := r.ussdCodeRepo.GetPrimaryUSSDCode(ctx, o.ID)
	if err != nil && err != xerrors.ErrNotFound {
		return fmt.Errorf("failed to load primary USSD code: %w", err)
	}
	
	if primaryCode != nil {
		o.PrimaryUSSDCode = primaryCode
	}
	
	return nil
}

// Create creates a new agent offer (now with transaction support for USSD code)
func (r *AgentOfferRepository) Create(ctx context.Context, o *offer.AgentOffer) error {
	// Begin transaction
	tx, err := r.dbWrapper.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Create offer
	query := `
		INSERT INTO agent_offers (
			agent_identity_id, offer_code, name, description, type, amount, units,
			price, currency, discount_percentage, validity_days, validity_label,
			ussd_code_template, ussd_processing_type, ussd_expected_response, ussd_error_pattern,
			is_featured, is_recurring, max_purchases_per_customer,
			status, available_from, available_until, tags, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24)
		RETURNING id, created_at, updated_at
	`

	var metadataJSON []byte
	if o.Metadata != nil {
		metadataJSON, err = json.Marshal(o.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	err = tx.QueryRow(
		ctx, query,
		o.AgentIdentityID, o.OfferCode, o.Name, o.Description, o.Type, o.Amount, o.Units,
		o.Price, o.Currency, o.DiscountPercentage, o.ValidityDays, o.ValidityLabel,
		o.USSDCodeTemplate, o.USSDProcessingType, o.USSDExpectedResponse, o.USSDErrorPattern,
		o.IsFeatured, o.IsRecurring, o.MaxPurchasesPerCustomer,
		o.Status, o.AvailableFrom, o.AvailableUntil, pq.Array(o.Tags), metadataJSON,
	).Scan(&o.ID, &o.CreatedAt, &o.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create offer: %w", err)
	}

	// Create initial USSD code (priority 1)
	ussdCode := &offer.OfferUSSDCode{
		OfferID:          o.ID,
		USSDCode:         o.USSDCodeTemplate,
		Priority:         1,
		IsActive:         true,
		ProcessingType:   o.USSDProcessingType,
		ExpectedResponse: o.USSDExpectedResponse,
		ErrorPattern:     o.USSDErrorPattern,
	}

	if err := r.ussdCodeRepo.CreateWithTx(ctx, tx, ussdCode); err != nil {
		return fmt.Errorf("failed to create USSD code: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// FindByID retrieves an offer by ID (now loads primary USSD code)
func (r *AgentOfferRepository) FindByID(ctx context.Context, id int64) (*offer.AgentOffer, error) {
	query := `
		SELECT id, agent_identity_id, offer_code, name, description, type, amount, units,
		       price, currency, discount_percentage, validity_days, validity_label,
		       ussd_code_template, ussd_processing_type, ussd_expected_response, ussd_error_pattern,
		       is_featured, is_recurring, max_purchases_per_customer,
		       status, available_from, available_until, tags, metadata,
		       created_at, updated_at, deleted_at
		FROM agent_offers
		WHERE id = $1 AND deleted_at IS NULL
	`

	row := r.db.QueryRow(ctx, query, id)
	o, err := r.scanOfferRow(row)
	
	if err != nil {
		if err.Error() == "failed to scan offer row: no rows in result set" {
			return nil, xerrors.ErrNotFound
		}
		return nil, fmt.Errorf("failed to find offer: %w", err)
	}

	// Load primary USSD code
	if err := r.loadPrimaryUSSDCode(ctx, o); err != nil {
		// Log but don't fail
		// In production, use logger here
	}

	return o, nil
}

// FindByOfferCode retrieves an offer by offer code (now loads primary USSD code)
func (r *AgentOfferRepository) FindByOfferCode(ctx context.Context, offerCode string) (*offer.AgentOffer, error) {
	query := `
		SELECT id, agent_identity_id, offer_code, name, description, type, amount, units,
		       price, currency, discount_percentage, validity_days, validity_label,
		       ussd_code_template, ussd_processing_type, ussd_expected_response, ussd_error_pattern,
		       is_featured, is_recurring, max_purchases_per_customer,
		       status, available_from, available_until, tags, metadata,
		       created_at, updated_at, deleted_at
		FROM agent_offers
		WHERE offer_code = $1 AND deleted_at IS NULL
	`

	row := r.db.QueryRow(ctx, query, offerCode)
	o, err := r.scanOfferRow(row)
	
	if err != nil {
		if err.Error() == "failed to scan offer row: no rows in result set" {
			return nil, xerrors.ErrNotFound
		}
		return nil, fmt.Errorf("failed to find offer by code: %w", err)
	}

	// Load primary USSD code
	if err := r.loadPrimaryUSSDCode(ctx, o); err != nil {
		// Log but don't fail
	}

	return o, nil
}

// Update updates an offer
func (r *AgentOfferRepository) Update(ctx context.Context, id int64, o *offer.AgentOffer) error {
	query := `
		UPDATE agent_offers
		SET name = $1, description = $2, type = $3, amount = $4, units = $5,
		    price = $6, discount_percentage = $7, validity_days = $8, validity_label = $9,
		    ussd_code_template = $10, ussd_processing_type = $11, ussd_expected_response = $12, ussd_error_pattern = $13,
		    is_featured = $14, is_recurring = $15, max_purchases_per_customer = $16,
		    available_from = $17, available_until = $18, tags = $19, metadata = $20, updated_at = $21
		WHERE id = $22 AND deleted_at IS NULL
	`

	var metadataJSON []byte
	var err error

	if o.Metadata != nil {
		metadataJSON, err = json.Marshal(o.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	result, err := r.db.Exec(
		ctx, query,
		o.Name, o.Description, o.Type, o.Amount, o.Units,
		o.Price, o.DiscountPercentage, o.ValidityDays, o.ValidityLabel,
		o.USSDCodeTemplate, o.USSDProcessingType, o.USSDExpectedResponse, o.USSDErrorPattern,
		o.IsFeatured, o.IsRecurring, o.MaxPurchasesPerCustomer,
		o.AvailableFrom, o.AvailableUntil, pq.Array(o.Tags), metadataJSON, time.Now(), id,
	)

	if err != nil {
		return fmt.Errorf("failed to update offer: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// UpdateStatus updates offer status
func (r *AgentOfferRepository) UpdateStatus(ctx context.Context, id int64, status offer.OfferStatus) error {
	query := `UPDATE agent_offers SET status = $1, updated_at = $2 WHERE id = $3 AND deleted_at IS NULL`

	result, err := r.db.Exec(ctx, query, status, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// SoftDelete soft deletes an offer
func (r *AgentOfferRepository) SoftDelete(ctx context.Context, id int64) error {
	query := `UPDATE agent_offers SET deleted_at = $1, updated_at = $2 WHERE id = $3 AND deleted_at IS NULL`

	result, err := r.db.Exec(ctx, query, time.Now(), time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to delete offer: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// List retrieves offers with filters (now loads primary USSD codes)
func (r *AgentOfferRepository) List(ctx context.Context, agentID int64, filters *offer.OfferListFilters) ([]offer.AgentOffer, int64, error) {
	// Build WHERE clause
	conditions := []string{"agent_identity_id = $1", "deleted_at IS NULL"}
	args := []interface{}{agentID}
	argPos := 2

	if filters.Type != nil {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argPos))
		args = append(args, *filters.Type)
		argPos++
	}

	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argPos))
		args = append(args, *filters.Status)
		argPos++
	}

	if filters.IsFeatured != nil {
		conditions = append(conditions, fmt.Sprintf("is_featured = $%d", argPos))
		args = append(args, *filters.IsFeatured)
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

	if filters.Search != "" {
		conditions = append(conditions, fmt.Sprintf(
			"(name ILIKE $%d OR description ILIKE $%d OR offer_code ILIKE $%d)",
			argPos, argPos, argPos,
		))
		args = append(args, "%"+filters.Search+"%")
		argPos++
	}

	if len(filters.Tags) > 0 {
		conditions = append(conditions, fmt.Sprintf("tags && $%d", argPos))
		args = append(args, pq.Array(filters.Tags))
		argPos++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM agent_offers WHERE %s", whereClause)
	var total int64
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count offers: %w", err)
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

	// Query offers
	query := fmt.Sprintf(`
		SELECT id, agent_identity_id, offer_code, name, description, type, amount, units,
		       price, currency, discount_percentage, validity_days, validity_label,
		       ussd_code_template, ussd_processing_type, ussd_expected_response, ussd_error_pattern,
		       is_featured, is_recurring, max_purchases_per_customer,
		       status, available_from, available_until, tags, metadata,
		       created_at, updated_at, deleted_at
		FROM agent_offers
		WHERE %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortBy, sortOrder, argPos, argPos+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list offers: %w", err)
	}
	defer rows.Close()

	offers := []offer.AgentOffer{}
	for rows.Next() {
		o, err := r.scanOfferRow(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan offer in list: %w", err)
		}
		offers = append(offers, *o)
	}

	// Load primary USSD codes for all offers
	for i := range offers {
		if err := r.loadPrimaryUSSDCode(ctx, &offers[i]); err != nil {
			// Log but don't fail
			continue
		}
	}

	return offers, total, nil
}

// GetStats retrieves offer statistics for an agent
func (r *AgentOfferRepository) GetStats(ctx context.Context, agentID int64) (*offer.OfferStats, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'active' THEN 1 END) as active,
			COUNT(CASE WHEN is_featured = TRUE THEN 1 END) as featured,
			COALESCE(AVG(price), 0) as avg_price
		FROM agent_offers
		WHERE agent_identity_id = $1 AND deleted_at IS NULL
	`

	var stats offer.OfferStats
	err := r.db.QueryRow(ctx, query, agentID).Scan(
		&stats.TotalOffers,
		&stats.ActiveOffers,
		&stats.FeaturedOffers,
		&stats.AveragePrice,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	// TODO: Calculate total revenue from offer_redemptions table when implemented
	stats.TotalRevenue = 0

	return &stats, nil
}

// ExistsByOfferCode checks if offer code exists
func (r *AgentOfferRepository) ExistsByOfferCode(ctx context.Context, offerCode string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM agent_offers WHERE offer_code = $1 AND deleted_at IS NULL)`
	var exists bool
	err := r.db.QueryRow(ctx, query, offerCode).Scan(&exists)
	return exists, err
}

// GetFeaturedOffers retrieves featured offers for an agent (now loads primary USSD codes)
func (r *AgentOfferRepository) GetFeaturedOffers(ctx context.Context, agentID int64, limit int) ([]offer.AgentOffer, error) {
	query := `
		SELECT id, agent_identity_id, offer_code, name, description, type, amount, units,
		       price, currency, discount_percentage, validity_days, validity_label,
		       ussd_code_template, ussd_processing_type, ussd_expected_response, ussd_error_pattern,
		       is_featured, is_recurring, max_purchases_per_customer,
		       status, available_from, available_until, tags, metadata,
		       created_at, updated_at, deleted_at
		FROM agent_offers
		WHERE agent_identity_id = $1 AND is_featured = TRUE AND status = 'active' AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, agentID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get featured offers: %w", err)
	}
	defer rows.Close()

	offers := []offer.AgentOffer{}
	for rows.Next() {
		o, err := r.scanOfferRow(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan featured offer: %w", err)
		}
		offers = append(offers, *o)
	}

	// Load primary USSD codes for all offers
	for i := range offers {
		if err := r.loadPrimaryUSSDCode(ctx, &offers[i]); err != nil {
			// Log but don't fail
			continue
		}
	}

	return offers, nil
}