// internal/repository/postgres/scheduled_offer_history_repository.go
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	//"time"

	"bingwa-service/internal/domain/schedule"
	xerrors "bingwa-service/internal/pkg/errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ScheduledOfferHistoryRepository struct {
	db *pgxpool.Pool
}

func NewScheduledOfferHistoryRepository(db *pgxpool.Pool) *ScheduledOfferHistoryRepository {
	return &ScheduledOfferHistoryRepository{db: db}
}

// CreateWithTx creates history entry within a transaction
func (r *ScheduledOfferHistoryRepository) CreateWithTx(ctx context.Context, tx pgx.Tx, history *schedule.ScheduledOfferHistory) error {
	query := `
		INSERT INTO scheduled_offer_history (
			scheduled_offer_id, offer_redemption_id, customer_id, customer_phone,
			renewal_time, renewal_number, status, failure_reason, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`

	var metadataJSON []byte
	var err error

	if history.Metadata != nil {
		metadataJSON, err = json.Marshal(history.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	err = tx.QueryRow(
		ctx, query,
		history.ScheduledOfferID, history.OfferRedemptionID, history.CustomerID, history.CustomerPhone,
		history.RenewalTime, history.RenewalNumber, history.Status, history.FailureReason, metadataJSON,
	).Scan(&history.ID, &history.CreatedAt, &history.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create history: %w", err)
	}

	return nil
}

// FindByID retrieves history by ID
func (r *ScheduledOfferHistoryRepository) FindByID(ctx context.Context, id int64) (*schedule.ScheduledOfferHistory, error) {
	query := `
		SELECT id, scheduled_offer_id, offer_redemption_id, customer_id, customer_phone,
		       renewal_time, renewal_number, status, failure_reason,
		       metadata, created_at, updated_at
		FROM scheduled_offer_history
		WHERE id = $1
	`

	var h schedule.ScheduledOfferHistory
	var metadataJSON []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
		&h.ID, &h.ScheduledOfferID, &h.OfferRedemptionID, &h.CustomerID, &h.CustomerPhone,
		&h.RenewalTime, &h.RenewalNumber, &h.Status, &h.FailureReason,
		&metadataJSON, &h.CreatedAt, &h.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, xerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find history: %w", err)
	}

	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &h.Metadata)
	}

	return &h, nil
}

// List retrieves history with filters
func (r *ScheduledOfferHistoryRepository) List(ctx context.Context, filters *schedule.ScheduleHistoryListFilters) ([]schedule.ScheduledOfferHistory, int64, error) {
	conditions := []string{}
	args := []interface{}{}
	argPos := 1

	if filters.ScheduledOfferID != nil {
		conditions = append(conditions, fmt.Sprintf("scheduled_offer_id = $%d", argPos))
		args = append(args, *filters.ScheduledOfferID)
		argPos++
	}

	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argPos))
		args = append(args, *filters.Status)
		argPos++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM scheduled_offer_history %s", whereClause)
	var total int64
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count history: %w", err)
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

	// Query history
	query := fmt.Sprintf(`
		SELECT id, scheduled_offer_id, offer_redemption_id, customer_id, customer_phone,
		       renewal_time, renewal_number, status, failure_reason,
		       metadata, created_at, updated_at
		FROM scheduled_offer_history
		%s
		ORDER BY renewal_time DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argPos, argPos+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list history: %w", err)
	}
	defer rows.Close()

	histories := []schedule.ScheduledOfferHistory{}
	for rows.Next() {
		var h schedule.ScheduledOfferHistory
		var metadataJSON []byte

		err := rows.Scan(
			&h.ID, &h.ScheduledOfferID, &h.OfferRedemptionID, &h.CustomerID, &h.CustomerPhone,
			&h.RenewalTime, &h.RenewalNumber, &h.Status, &h.FailureReason,
			&metadataJSON, &h.CreatedAt, &h.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan history: %w", err)
		}

		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &h.Metadata)
		}

		histories = append(histories, h)
	}

	return histories, total, nil
}

// GetRenewalStats retrieves renewal statistics for a schedule
func (r *ScheduledOfferHistoryRepository) GetRenewalStats(ctx context.Context, scheduledOfferID int64) (int64, int64, int64, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'success' THEN 1 END) as successful,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed
		FROM scheduled_offer_history
		WHERE scheduled_offer_id = $1
	`

	var total, successful, failed int64
	err := r.db.QueryRow(ctx, query, scheduledOfferID).Scan(&total, &successful, &failed)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get renewal stats: %w", err)
	}

	return total, successful, failed, nil
}