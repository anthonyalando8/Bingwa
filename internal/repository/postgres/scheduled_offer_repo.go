// internal/repository/postgres/scheduled_offer_repository.go
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"bingwa-service/internal/domain/schedule"
	xerrors "bingwa-service/internal/pkg/errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ScheduledOfferRepository struct {
	db *pgxpool.Pool
}

func NewScheduledOfferRepository(db *pgxpool.Pool) *ScheduledOfferRepository {
	return &ScheduledOfferRepository{db: db}
}

// CreateWithTx creates a scheduled offer within a transaction
func (r *ScheduledOfferRepository) CreateWithTx(ctx context.Context, tx pgx.Tx, schedule *schedule.ScheduledOffer) error {
	query := `
		INSERT INTO scheduled_offers (
			schedule_reference, offer_id, agent_identity_id, customer_id, customer_phone,
			scheduled_time, next_renewal_date, auto_renew, renewal_period,
			renewal_limit, renew_until, status, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, created_at, updated_at
	`

	var metadataJSON []byte
	var err error

	if schedule.Metadata != nil {
		metadataJSON, err = json.Marshal(schedule.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	err = tx.QueryRow(
		ctx, query,
		schedule.ScheduleReference, schedule.OfferID, schedule.AgentIdentityID, schedule.CustomerID, schedule.CustomerPhone,
		schedule.ScheduledTime, schedule.NextRenewalDate, schedule.AutoRenew, schedule.RenewalPeriod,
		schedule.RenewalLimit, schedule.RenewUntil, schedule.Status, metadataJSON,
	).Scan(&schedule.ID, &schedule.CreatedAt, &schedule.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create scheduled offer: %w", err)
	}

	return nil
}

// FindByID retrieves a scheduled offer by ID
func (r *ScheduledOfferRepository) FindByID(ctx context.Context, id int64) (*schedule.ScheduledOffer, error) {
	query := `
		SELECT id, schedule_reference, offer_id, agent_identity_id, customer_id, customer_phone,
		       scheduled_time, next_renewal_date, last_renewal_date,
		       auto_renew, renewal_period, renewal_count, renewal_limit, renew_until,
		       status, paused_at, cancelled_at, cancellation_reason,
		       metadata, created_at, updated_at
		FROM scheduled_offers
		WHERE id = $1
	`

	var s schedule.ScheduledOffer
	var metadataJSON []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.ScheduleReference, &s.OfferID, &s.AgentIdentityID, &s.CustomerID, &s.CustomerPhone,
		&s.ScheduledTime, &s.NextRenewalDate, &s.LastRenewalDate,
		&s.AutoRenew, &s.RenewalPeriod, &s.RenewalCount, &s.RenewalLimit, &s.RenewUntil,
		&s.Status, &s.PausedAt, &s.CancelledAt, &s.CancellationReason,
		&metadataJSON, &s.CreatedAt, &s.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, xerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find scheduled offer: %w", err)
	}

	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &s.Metadata)
	}

	return &s, nil
}

// Update updates a scheduled offer
func (r *ScheduledOfferRepository) Update(ctx context.Context, id int64, schedule *schedule.ScheduledOffer) error {
	query := `
		UPDATE scheduled_offers
		SET scheduled_time = $1, next_renewal_date = $2, auto_renew = $3, renewal_period = $4,
		    renewal_limit = $5, renew_until = $6, metadata = $7, updated_at = $8
		WHERE id = $9
	`

	var metadataJSON []byte
	var err error

	if schedule.Metadata != nil {
		metadataJSON, err = json.Marshal(schedule.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	result, err := r.db.Exec(
		ctx, query,
		schedule.ScheduledTime, schedule.NextRenewalDate, schedule.AutoRenew, schedule.RenewalPeriod,
		schedule.RenewalLimit, schedule.RenewUntil, metadataJSON, time.Now(), id,
	)

	if err != nil {
		return fmt.Errorf("failed to update scheduled offer: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// UpdateStatusWithTx updates status within a transaction
func (r *ScheduledOfferRepository) UpdateStatusWithTx(ctx context.Context, tx pgx.Tx, id int64, status schedule.ScheduleStatus) error {
	query := `UPDATE scheduled_offers SET status = $1, updated_at = $2 WHERE id = $3`

	result, err := tx.Exec(ctx, query, status, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// UpdateRenewalInfoWithTx updates renewal information within a transaction
func (r *ScheduledOfferRepository) UpdateRenewalInfoWithTx(ctx context.Context, tx pgx.Tx, id int64, nextRenewal, lastRenewal time.Time, renewalCount int) error {
	query := `
		UPDATE scheduled_offers
		SET next_renewal_date = $1, last_renewal_date = $2, renewal_count = $3, updated_at = $4
		WHERE id = $5
	`

	result, err := tx.Exec(
		ctx, query,
		sql.NullTime{Time: nextRenewal, Valid: true},
		sql.NullTime{Time: lastRenewal, Valid: true},
		renewalCount,
		time.Now(),
		id,
	)

	if err != nil {
		return fmt.Errorf("failed to update renewal info: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// PauseSchedule pauses a schedule
func (r *ScheduledOfferRepository) PauseSchedule(ctx context.Context, id int64) error {
	query := `UPDATE scheduled_offers SET status = $1, paused_at = $2, updated_at = $3 WHERE id = $4`

	result, err := r.db.Exec(ctx, query, schedule.ScheduleStatusPaused, time.Now(), time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to pause schedule: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// ResumeSchedule resumes a paused schedule
func (r *ScheduledOfferRepository) ResumeSchedule(ctx context.Context, id int64) error {
	query := `UPDATE scheduled_offers SET status = $1, paused_at = NULL, updated_at = $2 WHERE id = $3`

	result, err := r.db.Exec(ctx, query, schedule.ScheduleStatusActive, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to resume schedule: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// CancelSchedule cancels a schedule
func (r *ScheduledOfferRepository) CancelSchedule(ctx context.Context, id int64, reason string) error {
	query := `
		UPDATE scheduled_offers
		SET status = $1, cancelled_at = $2, cancellation_reason = $3, updated_at = $4
		WHERE id = $5
	`

	result, err := r.db.Exec(
		ctx, query,
		schedule.ScheduleStatusCancelled,
		time.Now(),
		sql.NullString{String: reason, Valid: reason != ""},
		time.Now(),
		id,
	)

	if err != nil {
		return fmt.Errorf("failed to cancel schedule: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// List retrieves scheduled offers with filters
func (r *ScheduledOfferRepository) List(ctx context.Context, agentID int64, filters *schedule.ScheduledOfferListFilters) ([]schedule.ScheduledOffer, int64, error) {
	conditions := []string{"agent_identity_id = $1"}
	args := []interface{}{agentID}
	argPos := 2

	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argPos))
		args = append(args, *filters.Status)
		argPos++
	}

	if filters.OfferID != nil {
		conditions = append(conditions, fmt.Sprintf("offer_id = $%d", argPos))
		args = append(args, *filters.OfferID)
		argPos++
	}

	if filters.CustomerPhone != "" {
		conditions = append(conditions, fmt.Sprintf("customer_phone = $%d", argPos))
		args = append(args, filters.CustomerPhone)
		argPos++
	}

	if filters.AutoRenew != nil {
		conditions = append(conditions, fmt.Sprintf("auto_renew = $%d", argPos))
		args = append(args, *filters.AutoRenew)
		argPos++
	}

	if filters.DueToday {
		conditions = append(conditions, fmt.Sprintf("DATE(next_renewal_date) = $%d", argPos))
		args = append(args, time.Now().Format("2006-01-02"))
		argPos++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM scheduled_offers WHERE %s", whereClause)
	var total int64
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count schedules: %w", err)
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
	sortBy := "next_renewal_date"
	if filters.SortBy != "" {
		sortBy = filters.SortBy
	}
	sortOrder := "ASC"
	if filters.SortOrder != "" {
		sortOrder = strings.ToUpper(filters.SortOrder)
	}

	// Query schedules
	query := fmt.Sprintf(`
		SELECT id, schedule_reference, offer_id, agent_identity_id, customer_id, customer_phone,
		       scheduled_time, next_renewal_date, last_renewal_date,
		       auto_renew, renewal_period, renewal_count, renewal_limit, renew_until,
		       status, paused_at, cancelled_at, cancellation_reason,
		       metadata, created_at, updated_at
		FROM scheduled_offers
		WHERE %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortBy, sortOrder, argPos, argPos+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list schedules: %w", err)
	}
	defer rows.Close()

	schedules := []schedule.ScheduledOffer{}
	for rows.Next() {
		var s schedule.ScheduledOffer
		var metadataJSON []byte

		err := rows.Scan(
			&s.ID, &s.ScheduleReference, &s.OfferID, &s.AgentIdentityID, &s.CustomerID, &s.CustomerPhone,
			&s.ScheduledTime, &s.NextRenewalDate, &s.LastRenewalDate,
			&s.AutoRenew, &s.RenewalPeriod, &s.RenewalCount, &s.RenewalLimit, &s.RenewUntil,
			&s.Status, &s.PausedAt, &s.CancelledAt, &s.CancellationReason,
			&metadataJSON, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan schedule: %w", err)
		}

		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &s.Metadata)
		}

		schedules = append(schedules, s)
	}

	return schedules, total, nil
}

// GetDueSchedules retrieves schedules due for renewal
func (r *ScheduledOfferRepository) GetDueSchedules(ctx context.Context) ([]schedule.ScheduledOffer, error) {
	query := `
		SELECT id, schedule_reference, offer_id, agent_identity_id, customer_id, customer_phone,
		       scheduled_time, next_renewal_date, last_renewal_date,
		       auto_renew, renewal_period, renewal_count, renewal_limit, renew_until,
		       status, paused_at, cancelled_at, cancellation_reason,
		       metadata, created_at, updated_at
		FROM scheduled_offers
		WHERE status = 'active' AND next_renewal_date <= $1
		ORDER BY next_renewal_date ASC
	`

	rows, err := r.db.Query(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to get due schedules: %w", err)
	}
	defer rows.Close()

	schedules := []schedule.ScheduledOffer{}
	for rows.Next() {
		var s schedule.ScheduledOffer
		var metadataJSON []byte

		err := rows.Scan(
			&s.ID, &s.ScheduleReference, &s.OfferID, &s.AgentIdentityID, &s.CustomerID, &s.CustomerPhone,
			&s.ScheduledTime, &s.NextRenewalDate, &s.LastRenewalDate,
			&s.AutoRenew, &s.RenewalPeriod, &s.RenewalCount, &s.RenewalLimit, &s.RenewUntil,
			&s.Status, &s.PausedAt, &s.CancelledAt, &s.CancellationReason,
			&metadataJSON, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan schedule: %w", err)
		}

		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &s.Metadata)
		}

		schedules = append(schedules, s)
	}

	return schedules, nil
}

// GetStats retrieves statistics
func (r *ScheduledOfferRepository) GetStats(ctx context.Context, agentID int64) (*schedule.ScheduleStats, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'active' THEN 1 END) as active,
			COUNT(CASE WHEN status = 'paused' THEN 1 END) as paused
		FROM scheduled_offers
		WHERE agent_identity_id = $1
	`

	var stats schedule.ScheduleStats
	err := r.db.QueryRow(ctx, query, agentID).Scan(
		&stats.TotalSchedules,
		&stats.ActiveSchedules,
		&stats.PausedSchedules,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	return &stats, nil
}

// ExistsByScheduleReference checks if schedule reference exists
func (r *ScheduledOfferRepository) ExistsByScheduleReference(ctx context.Context, reference string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM scheduled_offers WHERE schedule_reference = $1)`
	var exists bool
	err := r.db.QueryRow(ctx, query, reference).Scan(&exists)
	return exists, err
}