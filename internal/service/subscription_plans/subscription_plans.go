// internal/usecase/subscription/plan_service.go
package subscription

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"bingwa-service/internal/domain/subscription"
	//xerrors "bingwa-service/internal/pkg/errors"
	"bingwa-service/internal/repository/postgres"

	"go.uber.org/zap"
)

type PlanService struct {
	planRepo *postgres.SubscriptionPlanRepository
	logger   *zap.Logger
}

func NewPlanService(planRepo *postgres.SubscriptionPlanRepository, logger *zap.Logger) *PlanService {
	return &PlanService{
		planRepo: planRepo,
		logger:   logger,
	}
}

// CreatePlan creates a new subscription plan
func (s *PlanService) CreatePlan(ctx context.Context, req *subscription.CreatePlanRequest) (*subscription.SubscriptionPlan, error) {
	// Validate plan code format
	if err := s.validatePlanCode(req.PlanCode); err != nil {
		return nil, err
	}

	// Check if plan code already exists
	exists, err := s.planRepo.ExistsByPlanCode(ctx, req.PlanCode)
	if err != nil {
		return nil, fmt.Errorf("failed to check plan code: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("plan code already exists")
	}

	// Validate billing cycle
	if !s.isValidBillingCycle(req.BillingCycle) {
		return nil, fmt.Errorf("invalid billing cycle: %s", req.BillingCycle)
	}

	// Create plan entity
	plan := &subscription.SubscriptionPlan{
		PlanCode:     req.PlanCode,
		Name:         req.Name,
		Description:  sql.NullString{String: req.Description, Valid: req.Description != ""},
		Price:        req.Price,
		Currency:     strings.ToUpper(req.Currency),
		SetupFee:     req.SetupFee,
		BillingUsage: req.BillingUsage,
		BillingCycle: req.BillingCycle,
		Features:     req.Features,
		Status:       subscription.StatusActive,
		IsPublic:     req.IsPublic,
		Metadata:     req.Metadata,
	}

	// Set optional fields
	if req.OverageCharge != nil {
		plan.OverageCharge = sql.NullFloat64{Float64: *req.OverageCharge, Valid: true}
	}
	if req.MaxOffers != nil {
		plan.MaxOffers = sql.NullInt32{Int32: *req.MaxOffers, Valid: true}
	}
	if req.MaxCustomers != nil {
		plan.MaxCustomers = sql.NullInt32{Int32: *req.MaxCustomers, Valid: true}
	}

	// Create in database
	if err := s.planRepo.Create(ctx, plan); err != nil {
		s.logger.Error("failed to create plan", zap.Error(err))
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}

	s.logger.Info("subscription plan created",
		zap.Int64("plan_id", plan.ID),
		zap.String("plan_code", plan.PlanCode),
	)

	return plan, nil
}

// GetPlan retrieves a subscription plan by ID
func (s *PlanService) GetPlan(ctx context.Context, id int64) (*subscription.SubscriptionPlan, error) {
	plan, err := s.planRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return plan, nil
}

// GetPlanByCode retrieves a subscription plan by plan code
func (s *PlanService) GetPlanByCode(ctx context.Context, planCode string) (*subscription.SubscriptionPlan, error) {
	plan, err := s.planRepo.FindByPlanCode(ctx, planCode)
	if err != nil {
		return nil, err
	}

	return plan, nil
}

// ListPlans retrieves subscription plans with filters
func (s *PlanService) ListPlans(ctx context.Context, filters *subscription.PlanListFilters) (*subscription.PlanListResponse, error) {
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

	plans, total, err := s.planRepo.List(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list plans: %w", err)
	}

	totalPages := int(total) / filters.PageSize
	if int(total)%filters.PageSize > 0 {
		totalPages++
	}

	return &subscription.PlanListResponse{
		Plans:      plans,
		Total:      total,
		Page:       filters.Page,
		PageSize:   filters.PageSize,
		TotalPages: totalPages,
	}, nil
}

// ListPublicPlans retrieves only public (subscribable) plans
func (s *PlanService) ListPublicPlans(ctx context.Context, page, pageSize int) (*subscription.PlanListResponse, error) {
	isPublic := true
	status := subscription.StatusActive

	filters := &subscription.PlanListFilters{
		Status:   &status,
		IsPublic: &isPublic,
		Page:     page,
		PageSize: pageSize,
		SortBy:   "price",
		SortOrder: "asc",
	}

	return s.ListPlans(ctx, filters)
}

// UpdatePlan updates a subscription plan
func (s *PlanService) UpdatePlan(ctx context.Context, id int64, req *subscription.UpdatePlanRequest) (*subscription.SubscriptionPlan, error) {
	// Get existing plan
	plan, err := s.planRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.Name != nil {
		plan.Name = *req.Name
	}
	if req.Description != nil {
		plan.Description = sql.NullString{String: *req.Description, Valid: *req.Description != ""}
	}
	if req.Price != nil {
		plan.Price = *req.Price
	}
	if req.SetupFee != nil {
		plan.SetupFee = *req.SetupFee
	}
	if req.BillingUsage != nil {
		plan.BillingUsage = *req.BillingUsage
	}
	if req.BillingCycle != nil {
		if !s.isValidBillingCycle(*req.BillingCycle) {
			return nil, fmt.Errorf("invalid billing cycle: %s", *req.BillingCycle)
		}
		plan.BillingCycle = *req.BillingCycle
	}
	if req.OverageCharge != nil {
		plan.OverageCharge = sql.NullFloat64{Float64: *req.OverageCharge, Valid: true}
	}
	if req.MaxOffers != nil {
		plan.MaxOffers = sql.NullInt32{Int32: *req.MaxOffers, Valid: true}
	}
	if req.MaxCustomers != nil {
		plan.MaxCustomers = sql.NullInt32{Int32: *req.MaxCustomers, Valid: true}
	}
	if req.Features != nil {
		plan.Features = req.Features
	}
	if req.IsPublic != nil {
		plan.IsPublic = *req.IsPublic
	}
	if req.Metadata != nil {
		plan.Metadata = req.Metadata
	}

	// Update in database
	if err := s.planRepo.Update(ctx, id, plan); err != nil {
		s.logger.Error("failed to update plan", zap.Error(err))
		return nil, fmt.Errorf("failed to update plan: %w", err)
	}

	s.logger.Info("subscription plan updated",
		zap.Int64("plan_id", plan.ID),
		zap.String("plan_code", plan.PlanCode),
	)

	// Return updated plan
	return s.planRepo.FindByID(ctx, id)
}

// ActivatePlan activates a subscription plan
func (s *PlanService) ActivatePlan(ctx context.Context, id int64) error {
	if err := s.planRepo.UpdateStatus(ctx, id, subscription.StatusActive); err != nil {
		return fmt.Errorf("failed to activate plan: %w", err)
	}

	s.logger.Info("subscription plan activated", zap.Int64("plan_id", id))
	return nil
}

// DeactivatePlan deactivates a subscription plan
func (s *PlanService) DeactivatePlan(ctx context.Context, id int64) error {
	if err := s.planRepo.UpdateStatus(ctx, id, subscription.StatusInactive); err != nil {
		return fmt.Errorf("failed to deactivate plan: %w", err)
	}

	s.logger.Info("subscription plan deactivated", zap.Int64("plan_id", id))
	return nil
}

// DeletePlan deletes a subscription plan (only if no active subscriptions)
func (s *PlanService) DeletePlan(ctx context.Context, id int64) error {
	// Check if plan has active subscriptions
	hasSubscriptions, err := s.hasActiveSubscriptions(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to check subscriptions: %w", err)
	}

	if hasSubscriptions {
		return fmt.Errorf("cannot delete plan with active subscriptions")
	}

	// Delete plan
	if err := s.planRepo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete plan", zap.Error(err))
		return fmt.Errorf("failed to delete plan: %w", err)
	}

	s.logger.Info("subscription plan deleted", zap.Int64("plan_id", id))
	return nil
}

// GetStats retrieves subscription plan statistics
func (s *PlanService) GetStats(ctx context.Context) (*subscription.SubscriptionPlanStats, error) {
	stats, err := s.planRepo.GetStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	return stats, nil
}

// ========== Helper Methods ==========

// validatePlanCode validates plan code format
func (s *PlanService) validatePlanCode(code string) error {
	if len(code) < 3 || len(code) > 50 {
		return fmt.Errorf("plan code must be between 3 and 50 characters")
	}

	// Plan code should be alphanumeric with hyphens/underscores
	for _, char := range code {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || char == '-' || char == '_') {
			return fmt.Errorf("plan code can only contain letters, numbers, hyphens, and underscores")
		}
	}

	return nil
}

// isValidBillingCycle checks if billing cycle is valid
func (s *PlanService) isValidBillingCycle(cycle subscription.RenewalPeriod) bool {
	validCycles := []subscription.RenewalPeriod{
		subscription.RenewalDaily,
		subscription.RenewalWeekly,
		subscription.RenewalMonthly,
		subscription.RenewalQuarterly,
		subscription.RenewalYearly,
	}

	for _, valid := range validCycles {
		if cycle == valid {
			return true
		}
	}

	return false
}

// hasActiveSubscriptions checks if a plan has active subscriptions
func (s *PlanService) hasActiveSubscriptions(ctx context.Context, planID int64) (bool, error) {
	// This would query the agent_subscriptions table
	// For now, returning false to allow deletion
	// TODO: Implement actual check when agent_subscriptions is implemented
	return false, nil
}

// ========== Business Logic Methods ==========

// CalculatePlanCost calculates the total cost for a plan including setup fee
func (s *PlanService) CalculatePlanCost(plan *subscription.SubscriptionPlan, months int) float64 {
	setupCost := plan.SetupFee
	recurringCost := plan.Price * float64(months)
	return setupCost + recurringCost
}

// CalculateOverageCost calculates overage cost for exceeding billing usage
func (s *PlanService) CalculateOverageCost(plan *subscription.SubscriptionPlan, usedRequests, allowedRequests int) float64 {
	if usedRequests <= allowedRequests {
		return 0
	}

	if !plan.OverageCharge.Valid {
		return 0
	}

	overage := usedRequests - allowedRequests
	return float64(overage) * plan.OverageCharge.Float64
}

// GetNextBillingDate calculates the next billing date based on billing cycle
func (s *PlanService) GetNextBillingDate(startDate time.Time, cycle subscription.RenewalPeriod) time.Time {
	switch cycle {
	case subscription.RenewalDaily:
		return startDate.AddDate(0, 0, 1)
	case subscription.RenewalWeekly:
		return startDate.AddDate(0, 0, 7)
	case subscription.RenewalMonthly:
		return startDate.AddDate(0, 1, 0)
	case subscription.RenewalQuarterly:
		return startDate.AddDate(0, 3, 0)
	case subscription.RenewalYearly:
		return startDate.AddDate(1, 0, 0)
	default:
		return startDate.AddDate(0, 1, 0) // Default to monthly
	}
}

// ComparePlans compares two plans and returns differences
func (s *PlanService) ComparePlans(ctx context.Context, planID1, planID2 int64) (map[string]interface{}, error) {
	plan1, err := s.planRepo.FindByID(ctx, planID1)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan 1: %w", err)
	}

	plan2, err := s.planRepo.FindByID(ctx, planID2)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan 2: %w", err)
	}

	comparison := map[string]interface{}{
		"plan_1": map[string]interface{}{
			"id":            plan1.ID,
			"name":          plan1.Name,
			"price":         plan1.Price,
			"billing_cycle": plan1.BillingCycle,
			"max_offers":    plan1.MaxOffers,
			"max_customers": plan1.MaxCustomers,
		},
		"plan_2": map[string]interface{}{
			"id":            plan2.ID,
			"name":          plan2.Name,
			"price":         plan2.Price,
			"billing_cycle": plan2.BillingCycle,
			"max_offers":    plan2.MaxOffers,
			"max_customers": plan2.MaxCustomers,
		},
		"price_difference": plan2.Price - plan1.Price,
		"better_value":     s.determineBetterValue(plan1, plan2),
	}

	return comparison, nil
}

// determineBetterValue determines which plan offers better value
func (s *PlanService) determineBetterValue(plan1, plan2 *subscription.SubscriptionPlan) string {
	// Simple comparison based on price per billing usage
	value1 := plan1.Price / float64(plan1.BillingUsage)
	value2 := plan2.Price / float64(plan2.BillingUsage)

	if value1 < value2 {
		return plan1.Name
	}
	return plan2.Name
}