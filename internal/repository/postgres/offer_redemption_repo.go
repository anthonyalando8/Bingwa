// internal/repository/postgres/offer_redemption_repository.go
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"bingwa-service/internal/domain/transaction"
	xerrors "bingwa-service/internal/pkg/errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OfferRedemptionRepository struct {
	db *pgxpool.Pool
}

func NewOfferRedemptionRepository(db *pgxpool.Pool) *OfferRedemptionRepository {
	return &OfferRedemptionRepository{db: db}
}

// CreateWithTx creates an offer redemption within a transaction
func (r *OfferRedemptionRepository) CreateWithTx(ctx context.Context, tx pgx.Tx, redemption *transaction.OfferRedemption) error {
	query := `
		INSERT INTO offer_redemptions (
			redemption_reference, offer_id, offer_request_id, agent_identity_id,
			customer_id, customer_phone, amount, currency, ussd_code_used,
			redemption_time, status, max_retries, valid_from, valid_until, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, created_at, updated_at
	`

	var metadataJSON []byte
	var err error

	if redemption.Metadata != nil {
		metadataJSON, err = json.Marshal(redemption.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	err = tx.QueryRow(
		ctx, query,
		redemption.RedemptionReference, redemption.OfferID, redemption.OfferRequestID, redemption.AgentIdentityID,
		redemption.CustomerID, redemption.CustomerPhone, redemption.Amount, redemption.Currency, redemption.USSDCodeUsed,
		redemption.RedemptionTime, redemption.Status, redemption.MaxRetries, redemption.ValidFrom, redemption.ValidUntil, metadataJSON,
	).Scan(&redemption.ID, &redemption.CreatedAt, &redemption.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create redemption: %w", err)
	}

	return nil
}

// FindByID retrieves a redemption by ID
func (r *OfferRedemptionRepository) FindByID(ctx context.Context, id int64) (*transaction.OfferRedemption, error) {
	query := `
		SELECT id, redemption_reference, offer_id, offer_request_id, agent_identity_id,
		       customer_id, customer_phone, amount, currency, ussd_code_used,
		       ussd_response, ussd_session_id, ussd_processing_time,
		       redemption_time, completed_at, status, failure_reason, retry_count, max_retries,
		       valid_from, valid_until, metadata, created_at, updated_at
		FROM offer_redemptions
		WHERE id = $1
	`

	var redemption transaction.OfferRedemption
	var metadataJSON []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
		&redemption.ID, &redemption.RedemptionReference, &redemption.OfferID, &redemption.OfferRequestID, &redemption.AgentIdentityID,
		&redemption.CustomerID, &redemption.CustomerPhone, &redemption.Amount, &redemption.Currency, &redemption.USSDCodeUsed,
		&redemption.USSDResponse, &redemption.USSDSessionID, &redemption.USSDProcessingTime,
		&redemption.RedemptionTime, &redemption.CompletedAt, &redemption.Status, &redemption.FailureReason, &redemption.RetryCount, &redemption.MaxRetries,
		&redemption.ValidFrom, &redemption.ValidUntil, &metadataJSON, &redemption.CreatedAt, &redemption.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, xerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find redemption: %w", err)
	}

	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &redemption.Metadata)
	}

	return &redemption, nil
}

// UpdateStatusWithTx updates redemption status within a transaction
func (r *OfferRedemptionRepository) UpdateStatusWithTx(ctx context.Context, tx pgx.Tx, id int64, status transaction.TransactionStatus, failureReason string) error {
	query := `
		UPDATE offer_redemptions
		SET status = $1, failure_reason = $2, completed_at = $3, updated_at = $4
		WHERE id = $5
	`

	var completedAt sql.NullTime
	if status == transaction.TransactionStatusSuccess || status == transaction.TransactionStatusFailed {
		completedAt = sql.NullTime{Time: time.Now(), Valid: true}
	}

	var failureReasonNull sql.NullString
	if failureReason != "" {
		failureReasonNull = sql.NullString{String: failureReason, Valid: true}
	}

	result, err := tx.Exec(ctx, query, status, failureReasonNull, completedAt, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// UpdateUSSDResponse updates USSD response details
func (r *OfferRedemptionRepository) UpdateUSSDResponse(ctx context.Context, id int64, input *transaction.UpdateUSSDResponseInput) error {
	query := `
		UPDATE offer_redemptions
		SET ussd_response = $1, ussd_session_id = $2, ussd_processing_time = $3,
		    status = $4, failure_reason = $5, completed_at = $6, updated_at = $7
		WHERE id = $8
	`

	var completedAt sql.NullTime
	if input.Status == transaction.TransactionStatusSuccess || input.Status == transaction.TransactionStatusFailed {
		completedAt = sql.NullTime{Time: time.Now(), Valid: true}
	}

	result, err := r.db.Exec(
		ctx, query,
		sql.NullString{String: input.USSDResponse, Valid: input.USSDResponse != ""},
		sql.NullString{String: input.USSDSessionID, Valid: input.USSDSessionID != ""},
		sql.NullInt32{Int32: input.USSDProcessingTime, Valid: input.USSDProcessingTime > 0},
		input.Status,
		sql.NullString{String: input.FailureReason, Valid: input.FailureReason != ""},
		completedAt,
		time.Now(),
		id,
	)

	if err != nil {
		return fmt.Errorf("failed to update USSD response: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// List retrieves redemptions with filters
func (r *OfferRedemptionRepository) List(ctx context.Context, agentID int64, filters *transaction.RedemptionListFilters) ([]transaction.OfferRedemption, int64, error) {
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

	if filters.OfferRequestID != nil {
		conditions = append(conditions, fmt.Sprintf("offer_request_id = $%d", argPos))
		args = append(args, *filters.OfferRequestID)
		argPos++
	}

	if filters.CustomerPhone != "" {
		conditions = append(conditions, fmt.Sprintf("customer_phone = $%d", argPos))
		args = append(args, filters.CustomerPhone)
		argPos++
	}

	if filters.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("redemption_time >= $%d", argPos))
		args = append(args, *filters.DateFrom)
		argPos++
	}

	if filters.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("redemption_time <= $%d", argPos))
		args = append(args, *filters.DateTo)
		argPos++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM offer_redemptions WHERE %s", whereClause)
	var total int64
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count redemptions: %w", err)
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

	// Query redemptions
	query := fmt.Sprintf(`
		SELECT id, redemption_reference, offer_id, offer_request_id, agent_identity_id,
		       customer_id, customer_phone, amount, currency, ussd_code_used,
		       ussd_response, ussd_session_id, ussd_processing_time,
		       redemption_time, completed_at, status, failure_reason, retry_count, max_retries,
		       valid_from, valid_until, metadata, created_at, updated_at
		FROM offer_redemptions
		WHERE %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortBy, sortOrder, argPos, argPos+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list redemptions: %w", err)
	}
	defer rows.Close()

	redemptions := []transaction.OfferRedemption{}
	for rows.Next() {
		var redemption transaction.OfferRedemption
		var metadataJSON []byte

		err := rows.Scan(
			&redemption.ID, &redemption.RedemptionReference, &redemption.OfferID, &redemption.OfferRequestID, &redemption.AgentIdentityID,
			&redemption.CustomerID, &redemption.CustomerPhone, &redemption.Amount, &redemption.Currency, &redemption.USSDCodeUsed,
			&redemption.USSDResponse, &redemption.USSDSessionID, &redemption.USSDProcessingTime,
			&redemption.RedemptionTime, &redemption.CompletedAt, &redemption.Status, &redemption.FailureReason, &redemption.RetryCount, &redemption.MaxRetries,
			&redemption.ValidFrom, &redemption.ValidUntil, &metadataJSON, &redemption.CreatedAt, &redemption.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan redemption: %w", err)
		}

		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &redemption.Metadata)
		}

		redemptions = append(redemptions, redemption)
	}

	return redemptions, total, nil
}

// ExistsByRedemptionReference checks if redemption reference exists
func (r *OfferRedemptionRepository) ExistsByRedemptionReference(ctx context.Context, reference string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM offer_redemptions WHERE redemption_reference = $1)`
	var exists bool
	err := r.db.QueryRow(ctx, query, reference).Scan(&exists)
	return exists, err
}