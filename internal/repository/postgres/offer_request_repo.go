// internal/repository/postgres/offer_request_repository.go
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

type OfferRequestRepository struct {
	db *pgxpool.Pool
}

func NewOfferRequestRepository(db *pgxpool.Pool) *OfferRequestRepository {
	return &OfferRequestRepository{db: db}
}

// CreateWithTx creates an offer request within a transaction
func (r *OfferRequestRepository) CreateWithTx(ctx context.Context, tx pgx.Tx, req *transaction.OfferRequest) error {
	query := `
		INSERT INTO offer_requests (
			request_reference, offer_id, agent_identity_id, customer_id,
			customer_phone, customer_name, payment_method, amount_paid, currency,
			mpesa_transaction_id, mpesa_receipt_number, mpesa_transaction_date,
			mpesa_phone_number, mpesa_message, request_time, status,
			device_info, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		RETURNING id, created_at, updated_at
	`

	var deviceInfoJSON, metadataJSON []byte
	var err error

	if req.DeviceInfo != nil {
		deviceInfoJSON, err = json.Marshal(req.DeviceInfo)
		if err != nil {
			return fmt.Errorf("failed to marshal device_info: %w", err)
		}
	}

	if req.Metadata != nil {
		metadataJSON, err = json.Marshal(req.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	err = tx.QueryRow(
		ctx, query,
		req.RequestReference, req.OfferID, req.AgentIdentityID, req.CustomerID,
		req.CustomerPhone, req.CustomerName, req.PaymentMethod, req.AmountPaid, req.Currency,
		req.MpesaTransactionID, req.MpesaReceiptNumber, req.MpesaTransactionDate,
		req.MpesaPhoneNumber, req.MpesaMessage, req.RequestTime, req.Status,
		deviceInfoJSON, metadataJSON,
	).Scan(&req.ID, &req.CreatedAt, &req.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create offer request: %w", err)
	}

	return nil
}

// FindByID retrieves an offer request by ID
func (r *OfferRequestRepository) FindByID(ctx context.Context, id int64) (*transaction.OfferRequest, error) {
	query := `
		SELECT id, request_reference, offer_id, agent_identity_id, customer_id,
		       customer_phone, customer_name, payment_method, amount_paid, currency,
		       mpesa_transaction_id, mpesa_receipt_number, mpesa_transaction_date,
		       mpesa_phone_number, mpesa_message, request_time, processed_at,
		       status, failure_reason, retry_count, device_info, metadata,
		       created_at, updated_at
		FROM offer_requests
		WHERE id = $1
	`

	var req transaction.OfferRequest
	var deviceInfoJSON, metadataJSON []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
		&req.ID, &req.RequestReference, &req.OfferID, &req.AgentIdentityID, &req.CustomerID,
		&req.CustomerPhone, &req.CustomerName, &req.PaymentMethod, &req.AmountPaid, &req.Currency,
		&req.MpesaTransactionID, &req.MpesaReceiptNumber, &req.MpesaTransactionDate,
		&req.MpesaPhoneNumber, &req.MpesaMessage, &req.RequestTime, &req.ProcessedAt,
		&req.Status, &req.FailureReason, &req.RetryCount, &deviceInfoJSON, &metadataJSON,
		&req.CreatedAt, &req.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, xerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find offer request: %w", err)
	}

	// Unmarshal JSON fields
	if len(deviceInfoJSON) > 0 {
		json.Unmarshal(deviceInfoJSON, &req.DeviceInfo)
	}
	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &req.Metadata)
	}

	return &req, nil
}

// UpdateStatusWithTx updates offer request status within a transaction
func (r *OfferRequestRepository) UpdateStatusWithTx(ctx context.Context, tx pgx.Tx, id int64, status transaction.TransactionStatus, failureReason string) error {
	query := `
		UPDATE offer_requests
		SET status = $1, failure_reason = $2, processed_at = $3, updated_at = $4
		WHERE id = $5
	`

	var processedAt sql.NullTime
	if status == transaction.TransactionStatusSuccess || status == transaction.TransactionStatusFailed {
		processedAt = sql.NullTime{Time: time.Now(), Valid: true}
	}

	var failureReasonNull sql.NullString
	if failureReason != "" {
		failureReasonNull = sql.NullString{String: failureReason, Valid: true}
	}

	result, err := tx.Exec(ctx, query, status, failureReasonNull, processedAt, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// IncrementRetryCount increments retry count
func (r *OfferRequestRepository) IncrementRetryCount(ctx context.Context, id int64) error {
	query := `UPDATE offer_requests SET retry_count = retry_count + 1, updated_at = $1 WHERE id = $2`

	result, err := r.db.Exec(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to increment retry count: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// List retrieves offer requests with filters
func (r *OfferRequestRepository) List(ctx context.Context, agentID int64, filters *transaction.OfferRequestListFilters) ([]transaction.OfferRequest, int64, error) {
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

	if filters.PaymentMethod != nil {
		conditions = append(conditions, fmt.Sprintf("payment_method = $%d", argPos))
		args = append(args, *filters.PaymentMethod)
		argPos++
	}

	if filters.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("request_time >= $%d", argPos))
		args = append(args, *filters.DateFrom)
		argPos++
	}

	if filters.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("request_time <= $%d", argPos))
		args = append(args, *filters.DateTo)
		argPos++
	}

	if filters.Search != "" {
		conditions = append(conditions, fmt.Sprintf(
			"(customer_phone ILIKE $%d OR customer_name ILIKE $%d OR mpesa_transaction_id ILIKE $%d)",
			argPos, argPos, argPos,
		))
		args = append(args, "%"+filters.Search+"%")
		argPos++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM offer_requests WHERE %s", whereClause)
	var total int64
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count requests: %w", err)
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

	// Query requests
	query := fmt.Sprintf(`
		SELECT id, request_reference, offer_id, agent_identity_id, customer_id,
		       customer_phone, customer_name, payment_method, amount_paid, currency,
		       mpesa_transaction_id, mpesa_receipt_number, mpesa_transaction_date,
		       mpesa_phone_number, mpesa_message, request_time, processed_at,
		       status, failure_reason, retry_count, device_info, metadata,
		       created_at, updated_at
		FROM offer_requests
		WHERE %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortBy, sortOrder, argPos, argPos+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list requests: %w", err)
	}
	defer rows.Close()

	requests := []transaction.OfferRequest{}
	for rows.Next() {
		var req transaction.OfferRequest
		var deviceInfoJSON, metadataJSON []byte

		err := rows.Scan(
			&req.ID, &req.RequestReference, &req.OfferID, &req.AgentIdentityID, &req.CustomerID,
			&req.CustomerPhone, &req.CustomerName, &req.PaymentMethod, &req.AmountPaid, &req.Currency,
			&req.MpesaTransactionID, &req.MpesaReceiptNumber, &req.MpesaTransactionDate,
			&req.MpesaPhoneNumber, &req.MpesaMessage, &req.RequestTime, &req.ProcessedAt,
			&req.Status, &req.FailureReason, &req.RetryCount, &deviceInfoJSON, &metadataJSON,
			&req.CreatedAt, &req.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan request: %w", err)
		}

		if len(deviceInfoJSON) > 0 {
			json.Unmarshal(deviceInfoJSON, &req.DeviceInfo)
		}
		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &req.Metadata)
		}

		requests = append(requests, req)
	}

	return requests, total, nil
}

// GetStats retrieves statistics
func (r *OfferRequestRepository) GetStats(ctx context.Context, agentID int64) (*transaction.TransactionStats, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'success' THEN 1 END) as successful,
			COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed,
			COALESCE(SUM(CASE WHEN status = 'success' THEN amount_paid ELSE 0 END), 0) as revenue
		FROM offer_requests
		WHERE agent_identity_id = $1
	`

	var stats transaction.TransactionStats
	err := r.db.QueryRow(ctx, query, agentID).Scan(
		&stats.TotalRequests,
		&stats.SuccessfulRequests,
		&stats.PendingRequests,
		&stats.FailedRequests,
		&stats.TotalRevenue,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	// Calculate success rate
	if stats.TotalRequests > 0 {
		stats.SuccessRate = (float64(stats.SuccessfulRequests) / float64(stats.TotalRequests)) * 100
	}

	return &stats, nil
}

// ExistsByRequestReference checks if request reference exists
func (r *OfferRequestRepository) ExistsByRequestReference(ctx context.Context, reference string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM offer_requests WHERE request_reference = $1)`
	var exists bool
	err := r.db.QueryRow(ctx, query, reference).Scan(&exists)
	return exists, err
}