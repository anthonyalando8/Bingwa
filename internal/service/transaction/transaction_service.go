// internal/usecase/transaction/transaction_service.go
package transaction

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"bingwa-service/internal/domain/transaction"
	"bingwa-service/internal/repository/postgres"

	//"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

type TransactionService struct {
	requestRepo    *postgres.OfferRequestRepository
	redemptionRepo *postgres.OfferRedemptionRepository
	offerRepo      *postgres.AgentOfferRepository
	customerRepo   *postgres.AgentCustomerRepository
	db             *postgres.DB // For transaction management
	logger         *zap.Logger
	
	// Configuration
	requireSubscription bool // Toggle subscription check
}

func NewTransactionService(
	requestRepo *postgres.OfferRequestRepository,
	redemptionRepo *postgres.OfferRedemptionRepository,
	offerRepo *postgres.AgentOfferRepository,
	customerRepo *postgres.AgentCustomerRepository,
	db *postgres.DB,
	logger *zap.Logger,
) *TransactionService {
	return &TransactionService{
		requestRepo:         requestRepo,
		redemptionRepo:      redemptionRepo,
		offerRepo:           offerRepo,
		customerRepo:        customerRepo,
		db:                  db,
		logger:              logger,
		requireSubscription: false, // Default: don't require subscription (can be configured)
	}
}

// SetRequireSubscription configures whether to check subscription
func (s *TransactionService) SetRequireSubscription(require bool) {
	s.requireSubscription = require
}

// CreateOfferRequest creates an offer request and redemption in a transaction
func (s *TransactionService) CreateOfferRequest(ctx context.Context, agentID int64, input *transaction.CreateOfferRequestInput) (*transaction.OfferRequest, *transaction.OfferRedemption, error) {
	// Get offer details
	offer, err := s.offerRepo.FindByID(ctx, input.OfferID)
	if err != nil {
		return nil, nil, fmt.Errorf("offer not found: %w", err)
	}

	// Verify offer belongs to agent
	if offer.AgentIdentityID != agentID {
		return nil, nil, fmt.Errorf("unauthorized: offer does not belong to agent")
	}

	// Check if offer is available
	if offer.Status != "active" {
		return nil, nil, fmt.Errorf("offer is not active")
	}

	// Get or create customer
	customerID, err := s.getOrCreateCustomer(ctx, agentID, input.CustomerPhone, input.CustomerName)
	if err != nil {
		s.logger.Warn("failed to get/create customer", zap.Error(err))
		// Continue without customer_id
	}

	// Default to M-Pesa if not provided
	paymentMethod := input.PaymentMethod
	if paymentMethod == "" {
		paymentMethod = transaction.PaymentMethodMpesa
	}

	// Determine initial status based on input completeness
	isCompleted := s.isRequestCompleted(input)
	initialStatus := transaction.TransactionStatusPending
	if isCompleted {
		initialStatus = transaction.TransactionStatusSuccess
	}

	// Check subscription if required (only for non-completed requests)
	if !isCompleted && s.requireSubscription {
		hasSubscription, err := s.checkActiveSubscription(ctx, agentID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to check subscription: %w", err)
		}
		if !hasSubscription {
			// Save details with failed status and return error
			return s.createFailedRequest(ctx, agentID, input, offer, customerID, "No active subscription available")
		}
	}

	// Generate references
	requestRef := s.generateRequestReference()
	redemptionRef := s.generateRedemptionReference()

	// Create request entity
	offerRequest := &transaction.OfferRequest{
		RequestReference:  requestRef,
		OfferID:           input.OfferID,
		AgentIdentityID:   agentID,
		CustomerPhone:     input.CustomerPhone,
		PaymentMethod:     paymentMethod,
		AmountPaid:        input.AmountPaid,
		Currency:          strings.ToUpper(input.Currency),
		RequestTime:       time.Now(),
		Status:            initialStatus,
		RetryCount:        0,
		DeviceInfo:        input.DeviceInfo,
		Metadata:          input.Metadata,
	}

	if customerID != nil {
		offerRequest.CustomerID = sql.NullInt64{Int64: *customerID, Valid: true}
	}
	if input.CustomerName != "" {
		offerRequest.CustomerName = sql.NullString{String: input.CustomerName, Valid: true}
	}

	// M-Pesa details
	if input.MpesaTransactionID != "" {
		offerRequest.MpesaTransactionID = sql.NullString{String: input.MpesaTransactionID, Valid: true}
	}
	if input.MpesaReceiptNumber != "" {
		offerRequest.MpesaReceiptNumber = sql.NullString{String: input.MpesaReceiptNumber, Valid: true}
	}
	if !input.MpesaTransactionDate.IsZero() {
		offerRequest.MpesaTransactionDate = sql.NullTime{Time: input.MpesaTransactionDate, Valid: true}
	}
	if input.MpesaPhoneNumber != "" {
		offerRequest.MpesaPhoneNumber = sql.NullString{String: input.MpesaPhoneNumber, Valid: true}
	}
	if input.MpesaMessage != "" {
		offerRequest.MpesaMessage = sql.NullString{String: input.MpesaMessage, Valid: true}
	}

	if isCompleted {
		offerRequest.ProcessedAt = sql.NullTime{Time: time.Now(), Valid: true}
	}

	// Generate USSD code
	ussdCode := s.generateUSSDCode(offer, input.CustomerPhone)

	// Calculate validity dates
	validFrom := time.Now()
	validUntil := validFrom.AddDate(0, 0, offer.ValidityDays)

	// Create redemption entity
	redemption := &transaction.OfferRedemption{
		RedemptionReference: redemptionRef,
		OfferID:             input.OfferID,
		AgentIdentityID:     agentID,
		CustomerPhone:       input.CustomerPhone,
		Amount:              input.AmountPaid,
		Currency:            strings.ToUpper(input.Currency),
		USSDCodeUsed:        ussdCode,
		RedemptionTime:      time.Now(),
		Status:              initialStatus,
		RetryCount:          0,
		MaxRetries:          3,
		ValidFrom:           sql.NullTime{Time: validFrom, Valid: true},
		ValidUntil:          sql.NullTime{Time: validUntil, Valid: true},
	}

	if customerID != nil {
		redemption.CustomerID = sql.NullInt64{Int64: *customerID, Valid: true}
	}

	if isCompleted {
		redemption.CompletedAt = sql.NullTime{Time: time.Now(), Valid: true}
	}

	// Execute in transaction
	tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Create offer request
	if err := s.requestRepo.CreateWithTx(ctx, tx, offerRequest); err != nil {
		return nil, nil, fmt.Errorf("failed to create offer request: %w", err)
	}

	// Link redemption to request
	redemption.OfferRequestID = offerRequest.ID

	// Create redemption
	if err := s.redemptionRepo.CreateWithTx(ctx, tx, redemption); err != nil {
		return nil, nil, fmt.Errorf("failed to create redemption: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Info("offer request created",
		zap.Int64("request_id", offerRequest.ID),
		zap.Int64("redemption_id", redemption.ID),
		zap.String("status", string(offerRequest.Status)),
		zap.Int64("agent_id", agentID),
	)

	return offerRequest, redemption, nil
}

// UpdateOfferRequestStatus updates both request and redemption status in transaction
func (s *TransactionService) UpdateOfferRequestStatus(ctx context.Context, agentID, requestID int64, status transaction.TransactionStatus, ussdResponse *transaction.UpdateUSSDResponseInput) error {
	// Get request to verify ownership
	request, err := s.requestRepo.FindByID(ctx, requestID)
	if err != nil {
		return err
	}

	if request.AgentIdentityID != agentID {
		return fmt.Errorf("unauthorized: request does not belong to agent")
	}

	// Execute in transaction
	tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Update request status
	failureReason := ""
	if ussdResponse != nil && ussdResponse.FailureReason != "" {
		failureReason = ussdResponse.FailureReason
	}

	if err := s.requestRepo.UpdateStatusWithTx(ctx, tx, requestID, status, failureReason); err != nil {
		return fmt.Errorf("failed to update request status: %w", err)
	}

	// Find and update redemption
	// Note: We need to get redemption by request_id
	// For now, update via direct query (or add method to repo)
	redemptionQuery := `SELECT id FROM offer_redemptions WHERE offer_request_id = $1`
	var redemptionID int64
	if err := tx.QueryRow(ctx, redemptionQuery, requestID).Scan(&redemptionID); err != nil {
		return fmt.Errorf("failed to find redemption: %w", err)
	}

	// Update redemption status
	if ussdResponse != nil {
		if err := s.redemptionRepo.UpdateUSSDResponse(ctx, redemptionID, ussdResponse); err != nil {
			return fmt.Errorf("failed to update redemption: %w", err)
		}
	} else {
		if err := s.redemptionRepo.UpdateStatusWithTx(ctx, tx, redemptionID, status, failureReason); err != nil {
			return fmt.Errorf("failed to update redemption status: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Info("offer request status updated",
		zap.Int64("request_id", requestID),
		zap.String("status", string(status)),
	)

	return nil
}

// GetPendingRequests retrieves pending offer requests for retry
func (s *TransactionService) GetPendingRequests(ctx context.Context, agentID int64, limit int) ([]transaction.OfferRequest, error) {
	if limit < 1 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	status := transaction.TransactionStatusPending
	filters := &transaction.OfferRequestListFilters{
		Status:   &status,
		Page:     1,
		PageSize: limit,
		SortBy:   "created_at",
		SortOrder: "asc",
	}

	requests, _, err := s.requestRepo.List(ctx, agentID, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending requests: %w", err)
	}

	return requests, nil
}

// GetFailedRequests retrieves failed offer requests for retry
func (s *TransactionService) GetFailedRequests(ctx context.Context, agentID int64, limit int) ([]transaction.OfferRequest, error) {
	if limit < 1 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	status := transaction.TransactionStatusFailed
	filters := &transaction.OfferRequestListFilters{
		Status:   &status,
		Page:     1,
		PageSize: limit,
		SortBy:   "created_at",
		SortOrder: "asc",
	}

	requests, _, err := s.requestRepo.List(ctx, agentID, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get failed requests: %w", err)
	}

	return requests, nil
}

// GetProcessingRequests retrieves processing offer requests
func (s *TransactionService) GetProcessingRequests(ctx context.Context, agentID int64, limit int) ([]transaction.OfferRequest, error) {
	if limit < 1 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	status := transaction.TransactionStatusProcessing
	filters := &transaction.OfferRequestListFilters{
		Status:   &status,
		Page:     1,
		PageSize: limit,
		SortBy:   "created_at",
		SortOrder: "asc",
	}

	requests, _, err := s.requestRepo.List(ctx, agentID, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get processing requests: %w", err)
	}

	return requests, nil
}

// RetryFailedRequest retries a failed request
func (s *TransactionService) RetryFailedRequest(ctx context.Context, agentID, requestID int64) error {
	// Get request
	request, err := s.requestRepo.FindByID(ctx, requestID)
	if err != nil {
		return err
	}

	if request.AgentIdentityID != agentID {
		return fmt.Errorf("unauthorized: request does not belong to agent")
	}

	if request.Status != transaction.TransactionStatusFailed {
		return fmt.Errorf("can only retry failed requests")
	}

	// Increment retry count
	if err := s.requestRepo.IncrementRetryCount(ctx, requestID); err != nil {
		return fmt.Errorf("failed to increment retry count: %w", err)
	}

	// Update status to pending
	if err := s.UpdateOfferRequestStatus(ctx, agentID, requestID, transaction.TransactionStatusPending, nil); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	s.logger.Info("offer request retry initiated",
		zap.Int64("request_id", requestID),
		zap.Int("retry_count", request.RetryCount+1),
	)

	return nil
}

// GetOfferRequest retrieves an offer request by ID
func (s *TransactionService) GetOfferRequest(ctx context.Context, agentID, requestID int64) (*transaction.OfferRequest, error) {
	request, err := s.requestRepo.FindByID(ctx, requestID)
	if err != nil {
		return nil, err
	}

	if request.AgentIdentityID != agentID {
		return nil, fmt.Errorf("unauthorized: request does not belong to agent")
	}

	return request, nil
}

// ListOfferRequests retrieves offer requests with filters
func (s *TransactionService) ListOfferRequests(ctx context.Context, agentID int64, filters *transaction.OfferRequestListFilters) (*transaction.OfferRequestListResponse, error) {
	// Set defaults
	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.PageSize < 1 {
		filters.PageSize = 20
	}
	if filters.PageSize > 100 {
		filters.PageSize = 100
	}

	requests, total, err := s.requestRepo.List(ctx, agentID, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list requests: %w", err)
	}

	totalPages := int(total) / filters.PageSize
	if int(total)%filters.PageSize > 0 {
		totalPages++
	}

	return &transaction.OfferRequestListResponse{
		Requests:   requests,
		Total:      total,
		Page:       filters.Page,
		PageSize:   filters.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetOfferRedemption retrieves a redemption by ID
func (s *TransactionService) GetOfferRedemption(ctx context.Context, agentID, redemptionID int64) (*transaction.OfferRedemption, error) {
	redemption, err := s.redemptionRepo.FindByID(ctx, redemptionID)
	if err != nil {
		return nil, err
	}

	if redemption.AgentIdentityID != agentID {
		return nil, fmt.Errorf("unauthorized: redemption does not belong to agent")
	}

	return redemption, nil
}

// ListOfferRedemptions retrieves redemptions with filters
func (s *TransactionService) ListOfferRedemptions(ctx context.Context, agentID int64, filters *transaction.RedemptionListFilters) (*transaction.RedemptionListResponse, error) {
	// Set defaults
	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.PageSize < 1 {
		filters.PageSize = 20
	}
	if filters.PageSize > 100 {
		filters.PageSize = 100
	}

	redemptions, total, err := s.redemptionRepo.List(ctx, agentID, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list redemptions: %w", err)
	}

	totalPages := int(total) / filters.PageSize
	if int(total)%filters.PageSize > 0 {
		totalPages++
	}

	return &transaction.RedemptionListResponse{
		Redemptions: redemptions,
		Total:       total,
		Page:        filters.Page,
		PageSize:    filters.PageSize,
		TotalPages:  totalPages,
	}, nil
}

// GetTransactionStats retrieves transaction statistics
func (s *TransactionService) GetTransactionStats(ctx context.Context, agentID int64) (*transaction.TransactionStats, error) {
	stats, err := s.requestRepo.GetStats(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	return stats, nil
}

// ========== Helper Methods ==========

// isRequestCompleted checks if request is already completed (has USSD response)
func (s *TransactionService) isRequestCompleted(input *transaction.CreateOfferRequestInput) bool {
	// Consider completed if has M-Pesa transaction details
	return input.MpesaTransactionID != "" && input.MpesaReceiptNumber != ""
}

// generateRequestReference generates unique request reference
func (s *TransactionService) generateRequestReference() string {
	timestamp := time.Now().Format("20060102150405")
	random := generateRandomString(6)
	return fmt.Sprintf("REQ-%s-%s", timestamp, random)
}

// generateRedemptionReference generates unique redemption reference
func (s *TransactionService) generateRedemptionReference() string {
	timestamp := time.Now().Format("20060102150405")
	random := generateRandomString(6)
	return fmt.Sprintf("RED-%s-%s", timestamp, random)
}

// generateUSSDCode generates USSD code from template
func (s *TransactionService) generateUSSDCode(offer interface{}, phoneNumber string) string {
	// This would use the offer's USSD template
	// For now, return placeholder
	return fmt.Sprintf("*181*%s#", phoneNumber)
}

// getOrCreateCustomer gets existing customer or creates new one
func (s *TransactionService) getOrCreateCustomer(ctx context.Context, agentID int64, phone, name string) (*int64, error) {
	// Try to find existing customer
	customer, err := s.customerRepo.FindByAgentAndPhone(ctx, agentID, phone)
	if err == nil {
		return &customer.ID, nil
	}

	// Customer doesn't exist, create new one
	// This is simplified - in production you'd create properly
	// For now, return nil (customer creation can be optional)
	return nil, nil
}

// checkActiveSubscription checks if agent has active subscription
func (s *TransactionService) checkActiveSubscription(ctx context.Context, agentID int64) (bool, error) {
	// TODO: Implement subscription check
	// Placeholder for now - always return true
	return true, nil
}

// createFailedRequest creates a failed request when subscription check fails
func (s *TransactionService) createFailedRequest(ctx context.Context, agentID int64, input *transaction.CreateOfferRequestInput, offer interface{}, customerID *int64, reason string) (*transaction.OfferRequest, *transaction.OfferRedemption, error) {
	// Similar to CreateOfferRequest but with failed status
	requestRef := s.generateRequestReference()
	redemptionRef := s.generateRedemptionReference()

	offerRequest := &transaction.OfferRequest{
		RequestReference: requestRef,
		OfferID:          input.OfferID,
		AgentIdentityID:  agentID,
		CustomerPhone:    input.CustomerPhone,
		PaymentMethod:    input.PaymentMethod,
		AmountPaid:       input.AmountPaid,
		Currency:         strings.ToUpper(input.Currency),
		RequestTime:      time.Now(),
		Status:           transaction.TransactionStatusFailed,
		FailureReason:    sql.NullString{String: reason, Valid: true},
		RetryCount:       0,
	}

	if customerID != nil {
		offerRequest.CustomerID = sql.NullInt64{Int64: *customerID, Valid: true}
	}

	redemption := &transaction.OfferRedemption{
		RedemptionReference: redemptionRef,
		OfferID:             input.OfferID,
		AgentIdentityID:     agentID,
		CustomerPhone:       input.CustomerPhone,
		Amount:              input.AmountPaid,
		Currency:            strings.ToUpper(input.Currency),
		USSDCodeUsed:        "",
		RedemptionTime:      time.Now(),
		Status:              transaction.TransactionStatusFailed,
		FailureReason:       sql.NullString{String: reason, Valid: true},
		RetryCount:          0,
		MaxRetries:          3,
	}

	// Save in transaction
	tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.requestRepo.CreateWithTx(ctx, tx, offerRequest); err != nil {
		return nil, nil, err
	}

	redemption.OfferRequestID = offerRequest.ID

	if err := s.redemptionRepo.CreateWithTx(ctx, tx, redemption); err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}

	return offerRequest, redemption, fmt.Errorf("%s", reason)
}

func generateRandomString(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(time.Nanosecond)
	}
	return string(result)
}