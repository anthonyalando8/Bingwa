// internal/repository/postgres/offer_ussd_code_repository.go
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"bingwa-service/internal/domain/offer"
	xerrors "bingwa-service/internal/pkg/errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OfferUSSDCodeRepository struct {
	db *pgxpool.Pool
}

func NewOfferUSSDCodeRepository(db *pgxpool.Pool) *OfferUSSDCodeRepository {
	return &OfferUSSDCodeRepository{db: db}
}

// Create creates a new USSD code for an offer
func (r *OfferUSSDCodeRepository) Create(ctx context.Context, code *offer.OfferUSSDCode) error {
	query := `
		INSERT INTO offer_ussd_codes (
			offer_id, ussd_code, signature_pattern, priority, is_active,
			expected_response, error_pattern, processing_type, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`

	var metadataJSON []byte
	var err error

	if code.Metadata != nil {
		metadataJSON, err = json.Marshal(code.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	err = r.db.QueryRow(
		ctx, query,
		code.OfferID, code.USSDCode, code.SignaturePattern, code.Priority, code.IsActive,
		code.ExpectedResponse, code.ErrorPattern, code.ProcessingType, metadataJSON,
	).Scan(&code.ID, &code.CreatedAt, &code.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create USSD code: %w", err)
	}

	return nil
}

// CreateWithTx creates a USSD code within a transaction
func (r *OfferUSSDCodeRepository) CreateWithTx(ctx context.Context, tx pgx.Tx, code *offer.OfferUSSDCode) error {
	query := `
		INSERT INTO offer_ussd_codes (
			offer_id, ussd_code, signature_pattern, priority, is_active,
			expected_response, error_pattern, processing_type, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`

	var metadataJSON []byte
	var err error

	if code.Metadata != nil {
		metadataJSON, err = json.Marshal(code.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	err = tx.QueryRow(
		ctx, query,
		code.OfferID, code.USSDCode, code.SignaturePattern, code.Priority, code.IsActive,
		code.ExpectedResponse, code.ErrorPattern, code.ProcessingType, metadataJSON,
	).Scan(&code.ID, &code.CreatedAt, &code.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create USSD code: %w", err)
	}

	return nil
}

// FindByID retrieves a USSD code by ID
func (r *OfferUSSDCodeRepository) FindByID(ctx context.Context, id int64) (*offer.OfferUSSDCode, error) {
	query := `
		SELECT id, offer_id, ussd_code, signature_pattern, priority, is_active,
		       expected_response, error_pattern, processing_type,
		       success_count, failure_count, last_used_at, last_success_at, last_failure_at,
		       metadata, created_at, updated_at
		FROM offer_ussd_codes
		WHERE id = $1
	`

	var code offer.OfferUSSDCode
	var metadataJSON []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
		&code.ID, &code.OfferID, &code.USSDCode, &code.SignaturePattern, &code.Priority, &code.IsActive,
		&code.ExpectedResponse, &code.ErrorPattern, &code.ProcessingType,
		&code.SuccessCount, &code.FailureCount, &code.LastUsedAt, &code.LastSuccessAt, &code.LastFailureAt,
		&metadataJSON, &code.CreatedAt, &code.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, xerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find USSD code: %w", err)
	}

	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &code.Metadata)
	}

	return &code, nil
}

// ListByOfferID retrieves all USSD codes for an offer
func (r *OfferUSSDCodeRepository) ListByOfferID(ctx context.Context, offerID int64) ([]offer.OfferUSSDCode, error) {
	query := `
		SELECT id, offer_id, ussd_code, signature_pattern, priority, is_active,
		       expected_response, error_pattern, processing_type,
		       success_count, failure_count, last_used_at, last_success_at, last_failure_at,
		       metadata, created_at, updated_at
		FROM offer_ussd_codes
		WHERE offer_id = $1
		ORDER BY priority ASC, created_at ASC
	`

	rows, err := r.db.Query(ctx, query, offerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list USSD codes: %w", err)
	}
	defer rows.Close()

	codes := []offer.OfferUSSDCode{}
	for rows.Next() {
		var code offer.OfferUSSDCode
		var metadataJSON []byte

		err := rows.Scan(
			&code.ID, &code.OfferID, &code.USSDCode, &code.SignaturePattern, &code.Priority, &code.IsActive,
			&code.ExpectedResponse, &code.ErrorPattern, &code.ProcessingType,
			&code.SuccessCount, &code.FailureCount, &code.LastUsedAt, &code.LastSuccessAt, &code.LastFailureAt,
			&metadataJSON, &code.CreatedAt, &code.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan USSD code: %w", err)
		}

		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &code.Metadata)
		}

		codes = append(codes, code)
	}

	return codes, nil
}

// GetActiveCodesByPriority retrieves active USSD codes sorted by priority
func (r *OfferUSSDCodeRepository) GetActiveCodesByPriority(ctx context.Context, offerID int64) ([]offer.OfferUSSDCode, error) {
	query := `
		SELECT id, offer_id, ussd_code, signature_pattern, priority, is_active,
		       expected_response, error_pattern, processing_type,
		       success_count, failure_count, last_used_at, last_success_at, last_failure_at,
		       metadata, created_at, updated_at
		FROM offer_ussd_codes
		WHERE offer_id = $1 AND is_active = TRUE
		ORDER BY priority ASC, success_count DESC
	`

	rows, err := r.db.Query(ctx, query, offerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active codes: %w", err)
	}
	defer rows.Close()

	codes := []offer.OfferUSSDCode{}
	for rows.Next() {
		var code offer.OfferUSSDCode
		var metadataJSON []byte

		err := rows.Scan(
			&code.ID, &code.OfferID, &code.USSDCode, &code.SignaturePattern, &code.Priority, &code.IsActive,
			&code.ExpectedResponse, &code.ErrorPattern, &code.ProcessingType,
			&code.SuccessCount, &code.FailureCount, &code.LastUsedAt, &code.LastSuccessAt, &code.LastFailureAt,
			&metadataJSON, &code.CreatedAt, &code.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan USSD code: %w", err)
		}

		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &code.Metadata)
		}

		codes = append(codes, code)
	}

	return codes, nil
}

// Update updates a USSD code
func (r *OfferUSSDCodeRepository) Update(ctx context.Context, id int64, code *offer.OfferUSSDCode) error {
	query := `
		UPDATE offer_ussd_codes
		SET ussd_code = $1, signature_pattern = $2, priority = $3, is_active = $4,
		    expected_response = $5, error_pattern = $6, processing_type = $7,
		    metadata = $8, updated_at = $9
		WHERE id = $10
	`

	var metadataJSON []byte
	var err error

	if code.Metadata != nil {
		metadataJSON, err = json.Marshal(code.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	result, err := r.db.Exec(
		ctx, query,
		code.USSDCode, code.SignaturePattern, code.Priority, code.IsActive,
		code.ExpectedResponse, code.ErrorPattern, code.ProcessingType,
		metadataJSON, time.Now(), id,
	)

	if err != nil {
		return fmt.Errorf("failed to update USSD code: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// UpdatePriority updates the priority of a USSD code
func (r *OfferUSSDCodeRepository) UpdatePriority(ctx context.Context, id int64, priority int) error {
	query := `UPDATE offer_ussd_codes SET priority = $1, updated_at = $2 WHERE id = $3`

	result, err := r.db.Exec(ctx, query, priority, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update priority: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// ToggleActive toggles the active status of a USSD code
func (r *OfferUSSDCodeRepository) ToggleActive(ctx context.Context, id int64, isActive bool) error {
	query := `UPDATE offer_ussd_codes SET is_active = $1, updated_at = $2 WHERE id = $3`

	result, err := r.db.Exec(ctx, query, isActive, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to toggle active status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// RecordSuccess records a successful USSD execution
func (r *OfferUSSDCodeRepository) RecordSuccess(ctx context.Context, id int64) error {
	query := `
		UPDATE offer_ussd_codes
		SET success_count = success_count + 1,
		    last_used_at = $1,
		    last_success_at = $1,
		    updated_at = $1
		WHERE id = $2
	`

	result, err := r.db.Exec(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to record success: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// RecordFailure records a failed USSD execution
func (r *OfferUSSDCodeRepository) RecordFailure(ctx context.Context, id int64) error {
	query := `
		UPDATE offer_ussd_codes
		SET failure_count = failure_count + 1,
		    last_used_at = $1,
		    last_failure_at = $1,
		    updated_at = $1
		WHERE id = $2
	`

	result, err := r.db.Exec(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to record failure: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// Delete deletes a USSD code
func (r *OfferUSSDCodeRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM offer_ussd_codes WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete USSD code: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// GetStats retrieves statistics for USSD codes of an offer
func (r *OfferUSSDCodeRepository) GetStats(ctx context.Context, offerID int64) (*offer.USSDCodeStats, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN is_active = TRUE THEN 1 END) as active,
			COALESCE(SUM(success_count), 0) as total_success,
			COALESCE(SUM(failure_count), 0) as total_failure
		FROM offer_ussd_codes
		WHERE offer_id = $1
	`

	var stats offer.USSDCodeStats
	var totalSuccess, totalFailure int

	err := r.db.QueryRow(ctx, query, offerID).Scan(
		&stats.TotalCodes,
		&stats.ActiveCodes,
		&totalSuccess,
		&totalFailure,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	stats.TotalAttempts = totalSuccess + totalFailure
	if stats.TotalAttempts > 0 {
		stats.SuccessRate = (float64(totalSuccess) / float64(stats.TotalAttempts)) * 100
	}

	return &stats, nil
}

// ExistsByOfferAndCode checks if a USSD code exists for an offer
func (r *OfferUSSDCodeRepository) ExistsByOfferAndCode(ctx context.Context, offerID int64, ussdCode string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM offer_ussd_codes WHERE offer_id = $1 AND ussd_code = $2)`
	var exists bool
	err := r.db.QueryRow(ctx, query, offerID, ussdCode).Scan(&exists)
	return exists, err
}

// GetPrimaryUSSDCode gets the highest priority active USSD code
func (r *OfferUSSDCodeRepository) GetPrimaryUSSDCode(ctx context.Context, offerID int64) (*offer.OfferUSSDCode, error) {
	query := `
		SELECT id, offer_id, ussd_code, signature_pattern, priority, is_active,
		       expected_response, error_pattern, processing_type,
		       success_count, failure_count, last_used_at, last_success_at, last_failure_at,
		       metadata, created_at, updated_at
		FROM offer_ussd_codes
		WHERE offer_id = $1 AND is_active = TRUE
		ORDER BY priority ASC, success_count DESC
		LIMIT 1
	`

	var code offer.OfferUSSDCode
	var metadataJSON []byte

	err := r.db.QueryRow(ctx, query, offerID).Scan(
		&code.ID, &code.OfferID, &code.USSDCode, &code.SignaturePattern, &code.Priority, &code.IsActive,
		&code.ExpectedResponse, &code.ErrorPattern, &code.ProcessingType,
		&code.SuccessCount, &code.FailureCount, &code.LastUsedAt, &code.LastSuccessAt, &code.LastFailureAt,
		&metadataJSON, &code.CreatedAt, &code.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, xerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get primary USSD code: %w", err)
	}

	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &code.Metadata)
	}

	return &code, nil
}