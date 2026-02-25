// internal/usecase/schedule/schedule_service.go
package schedule

import (
	"context"
	"database/sql"
	"fmt"

	//"strings"
	"time"

	"bingwa-service/internal/domain/schedule"
	"bingwa-service/internal/domain/transaction"
	xerrors "bingwa-service/internal/pkg/errors"
	"bingwa-service/internal/service/offer"
	customer "bingwa-service/internal/service/customer"
	domainoffer "bingwa-service/internal/domain/offer"
	"bingwa-service/internal/repository/postgres"

	"go.uber.org/zap"
)

type ScheduleService struct {
	scheduleRepo       *postgres.ScheduledOfferRepository
	historyRepo        *postgres.ScheduledOfferHistoryRepository
	redemptionRepo     *postgres.OfferRedemptionRepository
	offerRepo          *postgres.AgentOfferRepository
	customerRepo       *postgres.AgentCustomerRepository
	customerSvc        *customer.CustomerService
	db                 *postgres.DB

	offerSvc 		   *offer.OfferService
	logger             *zap.Logger
}

func NewScheduleService(
	scheduleRepo *postgres.ScheduledOfferRepository,
	historyRepo *postgres.ScheduledOfferHistoryRepository,
	redemptionRepo *postgres.OfferRedemptionRepository,
	offerRepo *postgres.AgentOfferRepository,
	customerRepo *postgres.AgentCustomerRepository,
	customerSvc *customer.CustomerService,
	db *postgres.DB,
	offerSvc *offer.OfferService,
	logger *zap.Logger,
) *ScheduleService {
	return &ScheduleService{
		scheduleRepo:   scheduleRepo,
		historyRepo:    historyRepo,
		redemptionRepo: redemptionRepo,
		offerRepo:      offerRepo,
		customerRepo:   customerRepo,
		customerSvc:    customerSvc,
		db:             db,
		offerSvc:       offerSvc,
		logger:         logger,
	}
}

// CreateScheduledOffer creates a new scheduled offer
func (s *ScheduleService) CreateScheduledOffer(ctx context.Context, agentID int64, req *schedule.CreateScheduledOfferRequest) (*schedule.ScheduledOffer, error) {
	// Get offer details
	offer, err := s.offerRepo.FindByID(ctx, req.OfferID)
	if err != nil {
		return nil, fmt.Errorf("offer not found: %w", err)
	}

	// Verify offer belongs to agent
	if offer.AgentIdentityID != agentID {
		return nil, fmt.Errorf("unauthorized: offer does not belong to agent")
	}

	// Check if offer is active
	if offer.Status != "active" {
		return nil, fmt.Errorf("offer is not active")
	}

	// Validate scheduled time
	if req.ScheduledTime.Before(time.Now()) {
		return nil, fmt.Errorf("scheduled time must be in the future")
	}

	// Get or find customer
	customerID, err := s.customerSvc.GetOrCreateCustomer(ctx, agentID, req.CustomerPhone)
	if err != nil {
		s.logger.Warn("failed to get/create customer", zap.Error(err))
	}

	// Generate unique reference
	scheduleRef := s.generateScheduleReference()

	// Calculate next renewal date
	var nextRenewal sql.NullTime
	if req.AutoRenew && req.RenewalPeriod != "" {
		nextDate := s.calculateNextRenewal(req.ScheduledTime, req.RenewalPeriod)
		nextRenewal = sql.NullTime{Time: nextDate, Valid: true}
	}

	// Create scheduled offer entity
	scheduledOffer := &schedule.ScheduledOffer{
		ScheduleReference: scheduleRef,
		OfferID:           req.OfferID,
		AgentIdentityID:   agentID,
		CustomerPhone:     req.CustomerPhone,
		ScheduledTime:     req.ScheduledTime,
		NextRenewalDate:   nextRenewal,
		AutoRenew:         req.AutoRenew,
		Status:            schedule.ScheduleStatusActive,
		RenewalCount:      0,
		Metadata:          req.Metadata,
	}

	if customerID != nil {
		scheduledOffer.CustomerID = sql.NullInt64{Int64: *customerID, Valid: true}
	}

	if req.RenewalPeriod != "" {
		scheduledOffer.RenewalPeriod = sql.NullString{String: string(req.RenewalPeriod), Valid: true}
	}

	if req.RenewalLimit != nil {
		scheduledOffer.RenewalLimit = sql.NullInt32{Int32: *req.RenewalLimit, Valid: true}
	}

	if req.RenewUntil != nil {
		scheduledOffer.RenewUntil = sql.NullTime{Time: *req.RenewUntil, Valid: true}
	}

	// Execute in transaction
	tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Create scheduled offer
	if err := s.scheduleRepo.CreateWithTx(ctx, tx, scheduledOffer); err != nil {
		return nil, fmt.Errorf("failed to create scheduled offer: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Info("scheduled offer created",
		zap.Int64("schedule_id", scheduledOffer.ID),
		zap.String("schedule_reference", scheduledOffer.ScheduleReference),
		zap.Int64("agent_id", agentID),
	)

	return scheduledOffer, nil
}

// ExecuteScheduledOffer executes a scheduled offer and creates redemption + history
func (s *ScheduleService) ExecuteScheduledOffer(ctx context.Context, agentID, scheduleID int64, input *schedule.ExecuteScheduledOfferInput) (*transaction.OfferRedemption, *schedule.ScheduledOfferHistory, error) {
	// Get scheduled offer
	scheduledOffer, err := s.scheduleRepo.FindByID(ctx, scheduleID)
	if err != nil {
		return nil, nil, err
	}

	// Verify ownership
	if scheduledOffer.AgentIdentityID != agentID {
		return nil, nil, fmt.Errorf("unauthorized: scheduled offer does not belong to agent")
	}

	// Check if active
	if scheduledOffer.Status != schedule.ScheduleStatusActive {
		return nil, nil, fmt.Errorf("scheduled offer is not active")
	}

	// Get offer details
	offer, err := s.offerRepo.FindByID(ctx, scheduledOffer.OfferID)
	if err != nil {
		return nil, nil, fmt.Errorf("offer not found: %w", err)
	}

	// Determine execution status
	executionStatus := transaction.TransactionStatusSuccess
	if input.Status == "failed" {
		executionStatus = transaction.TransactionStatusFailed
	}

	// Generate redemption reference
	redemptionRef := s.generateRedemptionReference()

	// Generate USSD code
	ussdCodeInfo, err := s.generateUSSDCode(ctx, offer, scheduledOffer.CustomerPhone)
	if err != nil {
		s.logger.Error("failed to generate USSD code", zap.Error(err))
		//return nil, nil, fmt.Errorf("failed to generate USSD code: %w", err)
	}


	ussdCode := ussdCodeInfo.USSDCode
	if ussdCode == "" {
		ussdCode = offer.USSDCodeTemplate
	}

	// Calculate validity
	validFrom := time.Now()
	validUntil := validFrom.AddDate(0, 0, offer.ValidityDays)

	// Create redemption entity
	redemption := &transaction.OfferRedemption{
		RedemptionReference: redemptionRef,
		OfferID:             scheduledOffer.OfferID,
		OfferRequestID:      nil, // No request for scheduled offers
		AgentIdentityID:     agentID,
		CustomerID:          scheduledOffer.CustomerID,
		CustomerPhone:       scheduledOffer.CustomerPhone,
		Amount:              offer.Price,
		Currency:            offer.Currency,
		USSDCodeUsed:        ussdCode,
		RedemptionTime:      time.Now(),
		Status:              executionStatus,
		RetryCount:          0,
		MaxRetries:          3,
		ValidFrom:           sql.NullTime{Time: validFrom, Valid: true},
		ValidUntil:          sql.NullTime{Time: validUntil, Valid: true},
	}
	if ussdCodeInfo != nil {
		redemption.USSDCodeExecutionInfo = ussdCodeInfo
	}

	if input.USSDResponse != "" {
		redemption.USSDResponse = sql.NullString{String: input.USSDResponse, Valid: true}
	}
	if input.USSDSessionID != "" {
		redemption.USSDSessionID = sql.NullString{String: input.USSDSessionID, Valid: true}
	}
	if input.USSDProcessingTime > 0 {
		redemption.USSDProcessingTime = sql.NullInt32{Int32: input.USSDProcessingTime, Valid: true}
	}
	if input.FailureReason != "" {
		redemption.FailureReason = sql.NullString{String: input.FailureReason, Valid: true}
	}

	if executionStatus == transaction.TransactionStatusSuccess {
		redemption.CompletedAt = sql.NullTime{Time: time.Now(), Valid: true}
	}

	// Increment renewal count
	newRenewalCount := scheduledOffer.RenewalCount + 1

	// Create history entry
	history := &schedule.ScheduledOfferHistory{
		ScheduledOfferID: scheduleID,
		CustomerID:       scheduledOffer.CustomerID,
		CustomerPhone:    scheduledOffer.CustomerPhone,
		RenewalTime:      time.Now(),
		RenewalNumber:    newRenewalCount,
		Status:           string(executionStatus),
	}

	if input.FailureReason != "" {
		history.FailureReason = sql.NullString{String: input.FailureReason, Valid: true}
	}

	// Execute in transaction
	tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Create redemption
	if err := s.redemptionRepo.CreateWithTx(ctx, tx, redemption); err != nil {
		return nil, nil, fmt.Errorf("failed to create redemption: %w", err)
	}

	// Link redemption to history
	history.OfferRedemptionID = sql.NullInt64{Int64: redemption.ID, Valid: true}

	// Create history
	if err := s.historyRepo.CreateWithTx(ctx, tx, history); err != nil {
		return nil, nil, fmt.Errorf("failed to create history: %w", err)
	}

	// Update scheduled offer renewal info
	lastRenewal := time.Now()
	var nextRenewal time.Time

	// Calculate next renewal if auto-renew is enabled
	shouldContinue := s.shouldContinueRenewal(scheduledOffer, newRenewalCount)
	
	if scheduledOffer.AutoRenew && shouldContinue && executionStatus == transaction.TransactionStatusSuccess {
		if scheduledOffer.RenewalPeriod.Valid {
			nextRenewal = s.calculateNextRenewal(lastRenewal, schedule.RenewalPeriod(scheduledOffer.RenewalPeriod.String))
		}
		
		// Update renewal info
		if err := s.scheduleRepo.UpdateRenewalInfoWithTx(ctx, tx, scheduleID, nextRenewal, lastRenewal, newRenewalCount); err != nil {
			return nil, nil, fmt.Errorf("failed to update renewal info: %w", err)
		}
	} else if !shouldContinue {
		// Reached limit or end date, mark as completed
		if err := s.scheduleRepo.UpdateStatusWithTx(ctx, tx, scheduleID, schedule.ScheduleStatusCompleted); err != nil {
			return nil, nil, fmt.Errorf("failed to update status: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Info("scheduled offer executed",
		zap.Int64("schedule_id", scheduleID),
		zap.Int64("redemption_id", redemption.ID),
		zap.Int64("history_id", history.ID),
		zap.String("status", string(executionStatus)),
		zap.Int("renewal_number", newRenewalCount),
	)

	return redemption, history, nil
}

// GetScheduledOffer retrieves a scheduled offer by ID
func (s *ScheduleService) GetScheduledOffer(ctx context.Context, agentID, scheduleID int64) (*schedule.ScheduledOffer, error) {
	scheduledOffer, err := s.scheduleRepo.FindByID(ctx, scheduleID)
	if err != nil {
		return nil, err
	}

	if scheduledOffer.AgentIdentityID != agentID {
		return nil, xerrors.ErrUnauthorized
	}

	return scheduledOffer, nil
}

// ListScheduledOffers retrieves scheduled offers with filters
func (s *ScheduleService) ListScheduledOffers(ctx context.Context, agentID int64, filters *schedule.ScheduledOfferListFilters) (*schedule.ScheduledOfferListResponse, error) {
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

	schedules, total, err := s.scheduleRepo.List(ctx, agentID, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list scheduled offers: %w", err)
	}

	totalPages := int(total) / filters.PageSize
	if int(total)%filters.PageSize > 0 {
		totalPages++
	}

	return &schedule.ScheduledOfferListResponse{
		Schedules:  schedules,
		Total:      total,
		Page:       filters.Page,
		PageSize:   filters.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetDueSchedules retrieves schedules due for execution
func (s *ScheduleService) GetDueSchedules(ctx context.Context, agentID int64) ([]schedule.ScheduledOffer, error) {
	// Get all due schedules
	allDue, err := s.scheduleRepo.GetDueSchedules(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get due schedules: %w", err)
	}

	// Filter by agent
	agentSchedules := []schedule.ScheduledOffer{}
	for _, sched := range allDue {
		if sched.AgentIdentityID == agentID {
			agentSchedules = append(agentSchedules, sched)
		}
	}

	return agentSchedules, nil
}

// UpdateScheduledOffer updates a scheduled offer
func (s *ScheduleService) UpdateScheduledOffer(ctx context.Context, agentID, scheduleID int64, req *schedule.UpdateScheduledOfferRequest) (*schedule.ScheduledOffer, error) {
	// Get existing schedule
	scheduledOffer, err := s.scheduleRepo.FindByID(ctx, scheduleID)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if scheduledOffer.AgentIdentityID != agentID {
		return nil, xerrors.ErrUnauthorized
	}

	// Update fields
	if req.ScheduledTime != nil {
		scheduledOffer.ScheduledTime = *req.ScheduledTime
	}
	if req.AutoRenew != nil {
		scheduledOffer.AutoRenew = *req.AutoRenew
	}
	if req.RenewalPeriod != nil {
		scheduledOffer.RenewalPeriod = sql.NullString{String: string(*req.RenewalPeriod), Valid: true}
	}
	if req.RenewalLimit != nil {
		scheduledOffer.RenewalLimit = sql.NullInt32{Int32: *req.RenewalLimit, Valid: true}
	}
	if req.RenewUntil != nil {
		scheduledOffer.RenewUntil = sql.NullTime{Time: *req.RenewUntil, Valid: true}
	}
	if req.Metadata != nil {
		scheduledOffer.Metadata = req.Metadata
	}

	// Recalculate next renewal if renewal settings changed
	if (req.AutoRenew != nil || req.RenewalPeriod != nil) && scheduledOffer.AutoRenew && scheduledOffer.RenewalPeriod.Valid {
		nextDate := s.calculateNextRenewal(scheduledOffer.ScheduledTime, schedule.RenewalPeriod(scheduledOffer.RenewalPeriod.String))
		scheduledOffer.NextRenewalDate = sql.NullTime{Time: nextDate, Valid: true}
	}

	// Update in database
	if err := s.scheduleRepo.Update(ctx, scheduleID, scheduledOffer); err != nil {
		s.logger.Error("failed to update scheduled offer", zap.Error(err))
		return nil, fmt.Errorf("failed to update scheduled offer: %w", err)
	}

	s.logger.Info("scheduled offer updated", zap.Int64("schedule_id", scheduleID))

	return s.scheduleRepo.FindByID(ctx, scheduleID)
}

// PauseScheduledOffer pauses a scheduled offer
func (s *ScheduleService) PauseScheduledOffer(ctx context.Context, agentID, scheduleID int64) error {
	// Verify ownership
	scheduledOffer, err := s.scheduleRepo.FindByID(ctx, scheduleID)
	if err != nil {
		return err
	}
	if scheduledOffer.AgentIdentityID != agentID {
		return xerrors.ErrUnauthorized
	}

	if err := s.scheduleRepo.PauseSchedule(ctx, scheduleID); err != nil {
		return fmt.Errorf("failed to pause schedule: %w", err)
	}

	s.logger.Info("scheduled offer paused", zap.Int64("schedule_id", scheduleID))
	return nil
}

// ResumeScheduledOffer resumes a paused scheduled offer
func (s *ScheduleService) ResumeScheduledOffer(ctx context.Context, agentID, scheduleID int64) error {
	// Verify ownership
	scheduledOffer, err := s.scheduleRepo.FindByID(ctx, scheduleID)
	if err != nil {
		return err
	}
	if scheduledOffer.AgentIdentityID != agentID {
		return xerrors.ErrUnauthorized
	}

	if err := s.scheduleRepo.ResumeSchedule(ctx, scheduleID); err != nil {
		return fmt.Errorf("failed to resume schedule: %w", err)
	}

	s.logger.Info("scheduled offer resumed", zap.Int64("schedule_id", scheduleID))
	return nil
}

// CancelScheduledOffer cancels a scheduled offer
func (s *ScheduleService) CancelScheduledOffer(ctx context.Context, agentID, scheduleID int64, reason string) error {
	// Verify ownership
	scheduledOffer, err := s.scheduleRepo.FindByID(ctx, scheduleID)
	if err != nil {
		return err
	}
	if scheduledOffer.AgentIdentityID != agentID {
		return xerrors.ErrUnauthorized
	}

	if err := s.scheduleRepo.CancelSchedule(ctx, scheduleID, reason); err != nil {
		return fmt.Errorf("failed to cancel schedule: %w", err)
	}

	s.logger.Info("scheduled offer cancelled", zap.Int64("schedule_id", scheduleID), zap.String("reason", reason))
	return nil
}

// GetScheduleHistory retrieves history for a scheduled offer
func (s *ScheduleService) GetScheduleHistory(ctx context.Context, agentID, scheduleID int64, filters *schedule.ScheduleHistoryListFilters) (*schedule.ScheduleHistoryListResponse, error) {
	// Verify ownership
	scheduledOffer, err := s.scheduleRepo.FindByID(ctx, scheduleID)
	if err != nil {
		return nil, err
	}
	if scheduledOffer.AgentIdentityID != agentID {
		return nil, xerrors.ErrUnauthorized
	}

	// Set schedule ID filter
	filters.ScheduledOfferID = &scheduleID

	// Set defaults
	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.PageSize < 1 {
		filters.PageSize = 20
	}

	histories, total, err := s.historyRepo.List(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get history: %w", err)
	}

	totalPages := int(total) / filters.PageSize
	if int(total)%filters.PageSize > 0 {
		totalPages++
	}

	return &schedule.ScheduleHistoryListResponse{
		History:    histories,
		Total:      total,
		Page:       filters.Page,
		PageSize:   filters.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetScheduleStats retrieves statistics
func (s *ScheduleService) GetScheduleStats(ctx context.Context, agentID int64) (*schedule.ScheduleStats, error) {
	stats, err := s.scheduleRepo.GetStats(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	// Get renewal stats from history
	// This is simplified - you might want to optimize this
	filters := &schedule.ScheduledOfferListFilters{Page: 1, PageSize: 1000}
	schedules, _, _ := s.scheduleRepo.List(ctx, agentID, filters)
	
	for _, sched := range schedules {
		total, successful, failed, _ := s.historyRepo.GetRenewalStats(ctx, sched.ID)
		stats.TotalRenewals += total
		stats.SuccessfulRenewals += successful
		stats.FailedRenewals += failed
	}

	return stats, nil
}

// ========== Helper Methods ==========

// generateScheduleReference generates unique schedule reference
func (s *ScheduleService) generateScheduleReference() string {
	timestamp := time.Now().Format("20060102150405")
	random := generateRandomString(6)
	return fmt.Sprintf("SCH-%s-%s", timestamp, random)
}

// generateRedemptionReference generates unique redemption reference
func (s *ScheduleService) generateRedemptionReference() string {
	timestamp := time.Now().Format("20060102150405")
	random := generateRandomString(6)
	return fmt.Sprintf("RED-%s-%s", timestamp, random)
}

// generateUSSDCode generates USSD code from offer template
func (s *ScheduleService) generateUSSDCode(ctx context.Context, offer *domainoffer.AgentOffer, phoneNumber string) (*domainoffer.USSDCodeExecutionInfo, error) {
	return s.offerSvc.GetUSSDCodeForExecution(ctx, offer.AgentIdentityID, offer.ID, phoneNumber)
}

// calculateNextRenewal calculates next renewal date based on period
func (s *ScheduleService) calculateNextRenewal(from time.Time, period schedule.RenewalPeriod) time.Time {
	switch period {
	case schedule.RenewalDaily:
		return from.AddDate(0, 0, 1)
	case schedule.RenewalWeekly:
		return from.AddDate(0, 0, 7)
	case schedule.RenewalMonthly:
		return from.AddDate(0, 1, 0)
	case schedule.RenewalQuarterly:
		return from.AddDate(0, 3, 0)
	case schedule.RenewalYearly:
		return from.AddDate(1, 0, 0)
	default:
		return from.AddDate(0, 1, 0) // Default to monthly
	}
}

// shouldContinueRenewal checks if renewal should continue
func (s *ScheduleService) shouldContinueRenewal(scheduledOffer *schedule.ScheduledOffer, newCount int) bool {
	// Check renewal limit
	if scheduledOffer.RenewalLimit.Valid && newCount >= int(scheduledOffer.RenewalLimit.Int32) {
		return false
	}

	// Check end date
	if scheduledOffer.RenewUntil.Valid && time.Now().After(scheduledOffer.RenewUntil.Time) {
		return false
	}

	return true
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