// internal/usecase/subscription/subscription_service.go
package subscription

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"bingwa-service/internal/domain/subscription"
	xerrors "bingwa-service/internal/pkg/errors"
	"bingwa-service/internal/repository/postgres"

	"go.uber.org/zap"
)

type SubscriptionService struct {
	subscriptionRepo *postgres.AgentSubscriptionRepository
	planRepo         *postgres.SubscriptionPlanRepository
	campaignRepo     *postgres.PromotionalCampaignRepository
	db               *postgres.DB
	logger           *zap.Logger
}

func NewSubscriptionService(
	subscriptionRepo *postgres.AgentSubscriptionRepository,
	planRepo *postgres.SubscriptionPlanRepository,
	campaignRepo *postgres.PromotionalCampaignRepository,
	db *postgres.DB,
	logger *zap.Logger,
) *SubscriptionService {
	return &SubscriptionService{
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		campaignRepo:     campaignRepo,
		db:               db,
		logger:           logger,
	}
}

// CreateSubscription creates a new subscription (from mobile USSD payment)
func (s *SubscriptionService) CreateSubscription(ctx context.Context, agentID int64, req *subscription.CreateSubscriptionRequest) (*subscription.AgentSubscription, error) {
	// Get plan details
	plan, err := s.planRepo.FindByID(ctx, req.SubscriptionPlanID)
	if err != nil {
		return nil, fmt.Errorf("subscription plan not found: %w", err)
	}

	// Check if plan is active and public
	if plan.Status != subscription.StatusActive {
		return nil, fmt.Errorf("subscription plan is not active")
	}

	if !plan.IsPublic {
		return nil, fmt.Errorf("subscription plan is not available for subscription")
	}

	// Check if agent already has active subscription
	existingSub, _ := s.subscriptionRepo.FindActiveByAgent(ctx, agentID)
	if existingSub != nil {
		return nil, fmt.Errorf("agent already has an active subscription")
	}

	// Calculate pricing
	planPrice := plan.Price
	setupFee := plan.SetupFee
	totalPrice := planPrice + setupFee
	discountAmount := 0.0
	var campaignID *int64

	// Apply promotional code if provided
	if req.PromotionalCode != "" {
		discount, campID, err := s.applyPromotionalCode(ctx, req.PromotionalCode, plan.ID, planPrice)
		if err != nil {
			s.logger.Warn("failed to apply promotional code", zap.Error(err))
		} else {
			discountAmount = discount
			campaignID = campID
		}
	}

	finalPrice := totalPrice - discountAmount

	// Validate payment amount
	if req.AmountPaid < finalPrice {
		return nil, fmt.Errorf("insufficient payment: expected %.2f, received %.2f", finalPrice, req.AmountPaid)
	}

	// Calculate subscription period based on billing cycle
	startDate := time.Now()
	periodEnd := s.calculatePeriodEnd(startDate, plan.BillingCycle)
	
	// Calculate next billing date if auto-renew
	var nextBilling sql.NullTime
	if req.AutoRenew {
		nextBilling = sql.NullTime{Time: periodEnd, Valid: true}
	}

	// Generate unique reference
	subRef := s.generateSubscriptionReference()

	// Create subscription entity
	sub := &subscription.AgentSubscription{
		SubscriptionReference: subRef,
		AgentIdentityID:       agentID,
		SubscriptionPlanID:    req.SubscriptionPlanID,
		StartDate:             startDate,
		CurrentPeriodStart:    startDate,
		CurrentPeriodEnd:      periodEnd,
		AutoRenew:             req.AutoRenew,
		RenewalCount:          0,
		NextBillingDate:       nextBilling,
		RequestsUsed:          0,
		PlanPrice:             planPrice,
		DiscountApplied:       discountAmount,
		AmountPaid:            req.AmountPaid,
		Currency:              strings.ToUpper(req.Currency),
		Status:                subscription.SubscriptionStatusActive,
		Metadata:              req.Metadata,
	}

	// Set campaign if discount applied
	if campaignID != nil {
		sub.PromotionalCampaignID = sql.NullInt64{Int64: *campaignID, Valid: true}
	}

	// Set requests limit from plan (billing_usage)
	sub.RequestsLimit = sql.NullInt32{Int32: int32(plan.BillingUsage), Valid: true}

	// Add payment reference to metadata
	if req.PaymentReference != "" {
		if sub.Metadata == nil {
			sub.Metadata = make(map[string]interface{})
		}
		sub.Metadata["payment_reference"] = req.PaymentReference
		sub.Metadata["payment_method"] = req.PaymentMethod
		sub.Metadata["setup_fee"] = setupFee
	}

	// Execute in transaction
	tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Create subscription
	if err := s.subscriptionRepo.CreateWithTx(ctx, tx, sub); err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	// Increment campaign usage if applicable
	if campaignID != nil {
		if err := s.campaignRepo.IncrementUses(ctx, *campaignID); err != nil {
			s.logger.Warn("failed to increment campaign uses", zap.Error(err))
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Info("subscription created",
		zap.Int64("subscription_id", sub.ID),
		zap.String("subscription_reference", sub.SubscriptionReference),
		zap.Int64("agent_id", agentID),
		zap.Int64("plan_id", req.SubscriptionPlanID),
		zap.String("billing_cycle", string(plan.BillingCycle)),
		zap.Int("usage_limit", plan.BillingUsage),
	)

	return sub, nil
}

// RenewSubscription renews an existing subscription (from mobile USSD payment)
func (s *SubscriptionService) RenewSubscription(ctx context.Context, agentID int64, req *subscription.RenewSubscriptionRequest) (*subscription.AgentSubscription, error) {
	// Get current active subscription
	currentSub, err := s.subscriptionRepo.FindActiveByAgent(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("no active subscription found: %w", err)
	}

	// Get plan details
	plan, err := s.planRepo.FindByID(ctx, currentSub.SubscriptionPlanID)
	if err != nil {
		return nil, fmt.Errorf("subscription plan not found: %w", err)
	}

	// Calculate pricing (no setup fee for renewals)
	planPrice := plan.Price
	discountAmount := 0.0
	var campaignID *int64

	// Apply promotional code if provided
	if req.PromotionalCode != "" {
		discount, campID, err := s.applyPromotionalCode(ctx, req.PromotionalCode, plan.ID, planPrice)
		if err != nil {
			s.logger.Warn("failed to apply promotional code", zap.Error(err))
		} else {
			discountAmount = discount
			campaignID = campID
		}
	}

	finalPrice := planPrice - discountAmount

	// Validate payment amount
	if req.AmountPaid < finalPrice {
		return nil, fmt.Errorf("insufficient payment: expected %.2f, received %.2f", finalPrice, req.AmountPaid)
	}

	// Calculate new period
	newPeriodStart := currentSub.CurrentPeriodEnd
	if newPeriodStart.Before(time.Now()) {
		newPeriodStart = time.Now()
	}
	newPeriodEnd := s.calculatePeriodEnd(newPeriodStart, plan.BillingCycle)
	newRenewalCount := currentSub.RenewalCount + 1

	// Calculate next billing date
	nextBilling := newPeriodEnd

	// Execute in transaction
	tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Update renewal info
	if err := s.subscriptionRepo.UpdateRenewalInfoWithTx(ctx, tx, currentSub.ID, newPeriodStart, newPeriodEnd, nextBilling, newRenewalCount); err != nil {
		return nil, fmt.Errorf("failed to update renewal info: %w", err)
	}

	// Reset request usage counter for new billing cycle
	if err := s.subscriptionRepo.ResetRequestUsage(ctx, currentSub.ID); err != nil {
		s.logger.Warn("failed to reset request usage", zap.Error(err))
	}

	// Increment campaign usage if applicable
	if campaignID != nil {
		if err := s.campaignRepo.IncrementUses(ctx, *campaignID); err != nil {
			s.logger.Warn("failed to increment campaign uses", zap.Error(err))
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Info("subscription renewed",
		zap.Int64("subscription_id", currentSub.ID),
		zap.Int64("agent_id", agentID),
		zap.Int("renewal_count", newRenewalCount),
	)

	// Return updated subscription
	return s.subscriptionRepo.FindByID(ctx, currentSub.ID)
}

// GetSubscription retrieves a subscription by ID
func (s *SubscriptionService) GetSubscription(ctx context.Context, agentID, subscriptionID int64, isAdmin bool) (*subscription.AgentSubscription, error) {
	sub, err := s.subscriptionRepo.FindByID(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}

	// Verify ownership (unless admin)
	if !isAdmin && sub.AgentIdentityID != agentID {
		return nil, xerrors.ErrUnauthorized
	}

	return sub, nil
}

// GetActiveSubscription retrieves active subscription for an agent
func (s *SubscriptionService) GetActiveSubscription(ctx context.Context, agentID int64) (*subscription.AgentSubscription, error) {
	sub, err := s.subscriptionRepo.FindActiveByAgent(ctx, agentID)
	if err != nil {
		return nil, err
	}

	return sub, nil
}

// ListSubscriptions retrieves subscriptions with filters
func (s *SubscriptionService) ListSubscriptions(ctx context.Context, agentID int64, filters *subscription.SubscriptionListFilters, isAdmin bool) (*subscription.SubscriptionListResponse, error) {
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

	subscriptions, total, err := s.subscriptionRepo.List(ctx, agentID, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscriptions: %w", err)
	}

	totalPages := int(total) / filters.PageSize
	if int(total)%filters.PageSize > 0 {
		totalPages++
	}

	return &subscription.SubscriptionListResponse{
		Subscriptions: subscriptions,
		Total:         total,
		Page:          filters.Page,
		PageSize:      filters.PageSize,
		TotalPages:    totalPages,
	}, nil
}

// UpdateSubscription updates a subscription
func (s *SubscriptionService) UpdateSubscription(ctx context.Context, agentID, subscriptionID int64, req *subscription.UpdateSubscriptionRequest, isAdmin bool) (*subscription.AgentSubscription, error) {
	// Get existing subscription
	sub, err := s.subscriptionRepo.FindByID(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}

	// Verify ownership (unless admin)
	if !isAdmin && sub.AgentIdentityID != agentID {
		return nil, xerrors.ErrUnauthorized
	}

	// Update fields
	if req.AutoRenew != nil {
		sub.AutoRenew = *req.AutoRenew
	}
	if req.Metadata != nil {
		sub.Metadata = req.Metadata
	}

	// Update in database
	if err := s.subscriptionRepo.Update(ctx, subscriptionID, sub); err != nil {
		s.logger.Error("failed to update subscription", zap.Error(err))
		return nil, fmt.Errorf("failed to update subscription: %w", err)
	}

	s.logger.Info("subscription updated", zap.Int64("subscription_id", subscriptionID))

	return s.subscriptionRepo.FindByID(ctx, subscriptionID)
}

// CancelSubscription cancels a subscription
func (s *SubscriptionService) CancelSubscription(ctx context.Context, agentID, subscriptionID int64, req *subscription.CancelSubscriptionRequest, isAdmin bool) error {
	// Get subscription
	sub, err := s.subscriptionRepo.FindByID(ctx, subscriptionID)
	if err != nil {
		return err
	}

	// Verify ownership (unless admin)
	if !isAdmin && sub.AgentIdentityID != agentID {
		return xerrors.ErrUnauthorized
	}

	// Check if already cancelled
	if sub.Status == subscription.SubscriptionStatusCancelled {
		return fmt.Errorf("subscription is already cancelled")
	}

	// Cancel subscription
	if err := s.subscriptionRepo.CancelSubscription(ctx, subscriptionID, req.Reason, req.CancelImmediately); err != nil {
		s.logger.Error("failed to cancel subscription", zap.Error(err))
		return fmt.Errorf("failed to cancel subscription: %w", err)
	}

	cancelType := "at period end"
	if req.CancelImmediately {
		cancelType = "immediately"
	}

	s.logger.Info("subscription cancelled",
		zap.Int64("subscription_id", subscriptionID),
		zap.String("cancel_type", cancelType),
		zap.String("reason", req.Reason),
	)

	return nil
}

// GetSubscriptionUsage retrieves usage information for active subscription
func (s *SubscriptionService) GetSubscriptionUsage(ctx context.Context, agentID int64) (*subscription.SubscriptionUsageInfo, error) {
	sub, err := s.subscriptionRepo.FindActiveByAgent(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("no active subscription found: %w", err)
	}

	// Get plan to check for overage charges
	plan, err := s.planRepo.FindByID(ctx, sub.SubscriptionPlanID)
	if err != nil {
		return nil, fmt.Errorf("plan not found: %w", err)
	}

	usage := &subscription.SubscriptionUsageInfo{
		SubscriptionID: sub.ID,
		RequestsUsed:   sub.RequestsUsed,
	}

	// Calculate remaining requests
	if sub.RequestsLimit.Valid {
		usage.RequestsLimit = int(sub.RequestsLimit.Int32)
		usage.RequestsRemaining = usage.RequestsLimit - usage.RequestsUsed
		
		if usage.RequestsRemaining < 0 {
			// Over limit - check if overage is allowed
			usage.RequestsRemaining = 0
			if plan.OverageCharge.Valid {
				// Calculate overage charges
				overageCount := usage.RequestsUsed - usage.RequestsLimit
				overageAmount := float64(overageCount) * plan.OverageCharge.Float64
				
				if usage.Metadata == nil {
					usage.Metadata = make(map[string]interface{})
				}
				usage.Metadata["overage_count"] = overageCount
				usage.Metadata["overage_charge"] = overageAmount
				usage.Metadata["overage_rate"] = plan.OverageCharge.Float64
			}
		}
		
		usage.UsagePercentage = (float64(usage.RequestsUsed) / float64(usage.RequestsLimit)) * 100
		if usage.UsagePercentage > 100 {
			usage.UsagePercentage = 100
		}
	} else {
		// Unlimited
		usage.RequestsLimit = -1
		usage.RequestsRemaining = -1
		usage.UsagePercentage = 0
	}

	// Calculate days remaining
	now := time.Now()
	if sub.CurrentPeriodEnd.After(now) {
		duration := sub.CurrentPeriodEnd.Sub(now)
		usage.DaysRemaining = int(duration.Hours() / 24)
		usage.IsExpiring = usage.DaysRemaining <= 7
	} else {
		usage.DaysRemaining = 0
		usage.IsExpiring = true
	}

	// Check if can make requests
	// Allow if within limit OR if overage is allowed
	canMakeRequests := sub.Status == subscription.SubscriptionStatusActive &&
		sub.CurrentPeriodEnd.After(now)
	
	if sub.RequestsLimit.Valid {
		if usage.RequestsUsed < int(sub.RequestsLimit.Int32) {
			// Within limit
			canMakeRequests = canMakeRequests && true
		} else if plan.OverageCharge.Valid {
			// Over limit but overage allowed
			canMakeRequests = canMakeRequests && true
		} else {
			// Over limit and no overage allowed
			canMakeRequests = false
		}
	}
	
	usage.CanMakeRequests = canMakeRequests

	return usage, nil
}

// IncrementRequestUsage increments request usage counter
func (s *SubscriptionService) IncrementRequestUsage(ctx context.Context, agentID int64) error {
	sub, err := s.subscriptionRepo.FindActiveByAgent(ctx, agentID)
	if err != nil {
		return fmt.Errorf("no active subscription found: %w", err)
	}

	// Get plan to check overage
	plan, err := s.planRepo.FindByID(ctx, sub.SubscriptionPlanID)
	if err != nil {
		return fmt.Errorf("plan not found: %w", err)
	}

	// Check if limit reached and overage not allowed
	if sub.RequestsLimit.Valid && sub.RequestsUsed >= int(sub.RequestsLimit.Int32) {
		if !plan.OverageCharge.Valid {
			return fmt.Errorf("request limit reached and overage not allowed")
		}
		// Overage allowed - log warning
		s.logger.Warn("request usage over limit",
			zap.Int64("subscription_id", sub.ID),
			zap.Int("requests_used", sub.RequestsUsed),
			zap.Int("requests_limit", int(sub.RequestsLimit.Int32)),
		)
	}

	return s.subscriptionRepo.IncrementRequestUsage(ctx, sub.ID)
}

// CheckSubscriptionAccess checks if agent has active subscription access
func (s *SubscriptionService) CheckSubscriptionAccess(ctx context.Context, agentID int64) (bool, error) {
	sub, err := s.subscriptionRepo.FindActiveByAgent(ctx, agentID)
	if err != nil {
		return false, nil // No active subscription
	}

	// Check if active and not expired
	if sub.Status != subscription.SubscriptionStatusActive {
		return false, nil
	}

	if sub.CurrentPeriodEnd.Before(time.Now()) {
		return false, nil
	}

	// Get plan to check overage
	plan, err := s.planRepo.FindByID(ctx, sub.SubscriptionPlanID)
	if err != nil {
		return false, nil
	}

	// Check request limit
	if sub.RequestsLimit.Valid && sub.RequestsUsed >= int(sub.RequestsLimit.Int32) {
		// Check if overage is allowed
		if !plan.OverageCharge.Valid {
			return false, nil
		}
		// Overage allowed
	}

	return true, nil
}

// GetSubscriptionStats retrieves subscription statistics
func (s *SubscriptionService) GetSubscriptionStats(ctx context.Context, agentID int64, isAdmin bool) (*subscription.SubscriptionStats, error) {
	stats, err := s.subscriptionRepo.GetStats(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription stats: %w", err)
	}

	return stats, nil
}

// GetExpiringSubscriptions retrieves subscriptions expiring soon
func (s *SubscriptionService) GetExpiringSubscriptions(ctx context.Context, days int) ([]subscription.AgentSubscription, error) {
	if days < 1 {
		days = 7
	}
	if days > 30 {
		days = 30
	}

	subscriptions, err := s.subscriptionRepo.GetExpiringSubscriptions(ctx, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get expiring subscriptions: %w", err)
	}

	return subscriptions, nil
}

// ========== Admin Operations ==========

// DeactivateSubscription deactivates a subscription (admin only)
func (s *SubscriptionService) DeactivateSubscription(ctx context.Context, subscriptionID int64) error {
	tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.subscriptionRepo.UpdateStatusWithTx(ctx, tx, subscriptionID, subscription.SubscriptionStatusInactive); err != nil {
		return fmt.Errorf("failed to deactivate subscription: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Info("subscription deactivated by admin", zap.Int64("subscription_id", subscriptionID))
	return nil
}

// SuspendSubscription suspends a subscription (admin only)
func (s *SubscriptionService) SuspendSubscription(ctx context.Context, subscriptionID int64) error {
	tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.subscriptionRepo.UpdateStatusWithTx(ctx, tx, subscriptionID, subscription.SubscriptionStatusSuspended); err != nil {
		return fmt.Errorf("failed to suspend subscription: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Info("subscription suspended by admin", zap.Int64("subscription_id", subscriptionID))
	return nil
}

// ReactivateSubscription reactivates a subscription (admin only)
func (s *SubscriptionService) ReactivateSubscription(ctx context.Context, subscriptionID int64) error {
	tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.subscriptionRepo.UpdateStatusWithTx(ctx, tx, subscriptionID, subscription.SubscriptionStatusActive); err != nil {
		return fmt.Errorf("failed to reactivate subscription: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Info("subscription reactivated by admin", zap.Int64("subscription_id", subscriptionID))
	return nil
}

// ========== Helper Methods ==========

// generateSubscriptionReference generates unique subscription reference
func (s *SubscriptionService) generateSubscriptionReference() string {
	timestamp := time.Now().Format("20060102150405")
	random := generateRandomString(6)
	return fmt.Sprintf("SUB-%s-%s", timestamp, random)
}

// applyPromotionalCode applies promotional code and returns discount amount
func (s *SubscriptionService) applyPromotionalCode(ctx context.Context, code string, planID int64, basePrice float64) (float64, *int64, error) {
	campaign, err := s.campaignRepo.FindByPromotionalCode(ctx, code)
	if err != nil {
		return 0, nil, fmt.Errorf("promotional code not found: %w", err)
	}

	// Check if campaign is active
	now := time.Now()
	if campaign.Status != "active" || now.Before(campaign.StartDate) || now.After(campaign.EndDate) {
		return 0, nil, fmt.Errorf("promotional code is not active")
	}

	// Check if applicable to this plan
	if len(campaign.ApplicablePlans) > 0 {
		planApplicable := false
		for _, applicablePlanID := range campaign.ApplicablePlans {
			if applicablePlanID == planID {
				planApplicable = true
				break
			}
		}
		if !planApplicable {
			return 0, nil, fmt.Errorf("promotional code not applicable to this plan")
		}
	}

	// Check usage limit
	if campaign.MaxUses.Valid && campaign.CurrentUses >= int(campaign.MaxUses.Int32) {
		return 0, nil, fmt.Errorf("promotional code usage limit reached")
	}

	// Calculate discount
	var discount float64
	switch campaign.DiscountType {
	case "percentage":
		discount = basePrice * (campaign.DiscountValue / 100)
		if campaign.MaxDiscountAmount.Valid && discount > campaign.MaxDiscountAmount.Float64 {
			discount = campaign.MaxDiscountAmount.Float64
		}
	case "fixed_amount":
		discount = campaign.DiscountValue
		if discount > basePrice {
			discount = basePrice
		}
	case "free_trial":
		discount = basePrice // Full discount
	}

	return discount, &campaign.ID, nil
}

// calculatePeriodEnd calculates period end date based on billing cycle
func (s *SubscriptionService) calculatePeriodEnd(start time.Time, cycle subscription.RenewalPeriod) time.Time {
	switch cycle {
	case subscription.RenewalDaily:
		return start.AddDate(0, 0, 1)
	case subscription.RenewalWeekly:
		return start.AddDate(0, 0, 7)
	case subscription.RenewalMonthly:
		return start.AddDate(0, 1, 0)
	case subscription.RenewalQuarterly:
		return start.AddDate(0, 3, 0)
	case subscription.RenewalYearly:
		return start.AddDate(1, 0, 0)
	default:
		return start.AddDate(0, 1, 0) // Default to 1 month
	}
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