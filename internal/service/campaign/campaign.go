// internal/usecase/campaign/campaign_service.go
package campaign

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"bingwa-service/internal/domain/campaign"
	xerrors "bingwa-service/internal/pkg/errors"
	"bingwa-service/internal/repository/postgres"

	"github.com/lib/pq"
	"go.uber.org/zap"
)

type CampaignService struct {
	campaignRepo *postgres.PromotionalCampaignRepository
	logger       *zap.Logger
}

func NewCampaignService(campaignRepo *postgres.PromotionalCampaignRepository, logger *zap.Logger) *CampaignService {
	return &CampaignService{
		campaignRepo: campaignRepo,
		logger:       logger,
	}
}

// ========== Admin Operations ==========

// CreateCampaign creates a new promotional campaign (admin only)
func (s *CampaignService) CreateCampaign(ctx context.Context, req *campaign.CreateCampaignRequest) (*campaign.PromotionalCampaign, error) {
	// Validate dates
	if req.EndDate.Before(req.StartDate) {
		return nil, fmt.Errorf("end date must be after start date")
	}

	// Validate discount type and value
	if err := s.validateDiscountTypeAndValue(req.DiscountType, req.DiscountValue); err != nil {
		return nil, err
	}

	// Validate promotional code format
	if err := s.validatePromotionalCode(req.PromotionalCode); err != nil {
		return nil, err
	}

	// Check if promotional code already exists
	exists, err := s.campaignRepo.ExistsByPromotionalCode(ctx, req.PromotionalCode)
	if err != nil {
		return nil, fmt.Errorf("failed to check promotional code: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("promotional code '%s' already exists", req.PromotionalCode)
	}

	// Generate unique campaign code
	campaignCode, err := s.generateCampaignCode(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate campaign code: %w", err)
	}

	// Create campaign entity
	c := &campaign.PromotionalCampaign{
		CampaignCode:    campaignCode,
		Name:            req.Name,
		Description:     sql.NullString{String: req.Description, Valid: req.Description != ""},
		PromotionalCode: strings.ToUpper(req.PromotionalCode),
		DiscountType:    req.DiscountType,
		DiscountValue:   req.DiscountValue,
		StartDate:       req.StartDate,
		EndDate:         req.EndDate,
		UsesPerUser:     req.UsesPerUser,
		CurrentUses:     0,
		ApplicablePlans: pq.Int64Array(req.ApplicablePlans),
		TargetUserTypes: pq.StringArray(req.TargetUserTypes),
		Status:          campaign.CampaignStatusActive,
		Metadata:        req.Metadata,
	}

	// Set optional fields
	if req.MaxDiscountAmount != nil {
		c.MaxDiscountAmount = sql.NullFloat64{Float64: *req.MaxDiscountAmount, Valid: true}
	}
	if req.MaxUses != nil {
		c.MaxUses = sql.NullInt32{Int32: *req.MaxUses, Valid: true}
	}

	// Create in database
	if err := s.campaignRepo.Create(ctx, c); err != nil {
		s.logger.Error("failed to create campaign", zap.Error(err))
		return nil, fmt.Errorf("failed to create campaign: %w", err)
	}

	s.logger.Info("campaign created",
		zap.Int64("campaign_id", c.ID),
		zap.String("campaign_code", c.CampaignCode),
		zap.String("promotional_code", c.PromotionalCode),
	)

	return c, nil
}

// UpdateCampaign updates a promotional campaign (admin only)
func (s *CampaignService) UpdateCampaign(ctx context.Context, id int64, req *campaign.UpdateCampaignRequest) (*campaign.PromotionalCampaign, error) {
	// Get existing campaign
	c, err := s.campaignRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.Name != nil {
		c.Name = *req.Name
	}
	if req.Description != nil {
		c.Description = sql.NullString{String: *req.Description, Valid: *req.Description != ""}
	}
	if req.DiscountValue != nil {
		// Validate discount value
		if err := s.validateDiscountTypeAndValue(c.DiscountType, *req.DiscountValue); err != nil {
			return nil, err
		}
		c.DiscountValue = *req.DiscountValue
	}
	if req.MaxDiscountAmount != nil {
		c.MaxDiscountAmount = sql.NullFloat64{Float64: *req.MaxDiscountAmount, Valid: true}
	}
	if req.StartDate != nil {
		c.StartDate = *req.StartDate
	}
	if req.EndDate != nil {
		c.EndDate = *req.EndDate
	}
	if req.MaxUses != nil {
		c.MaxUses = sql.NullInt32{Int32: *req.MaxUses, Valid: true}
	}
	if req.UsesPerUser != nil {
		c.UsesPerUser = *req.UsesPerUser
	}
	if req.ApplicablePlans != nil {
		c.ApplicablePlans = pq.Int64Array(req.ApplicablePlans)
	}
	if req.TargetUserTypes != nil {
		c.TargetUserTypes = pq.StringArray(req.TargetUserTypes)
	}
	if req.Metadata != nil {
		c.Metadata = req.Metadata
	}

	// Validate dates
	if c.EndDate.Before(c.StartDate) {
		return nil, fmt.Errorf("end date must be after start date")
	}

	// Update in database
	if err := s.campaignRepo.Update(ctx, id, c); err != nil {
		s.logger.Error("failed to update campaign", zap.Error(err))
		return nil, fmt.Errorf("failed to update campaign: %w", err)
	}

	s.logger.Info("campaign updated",
		zap.Int64("campaign_id", id),
	)

	// Return updated campaign
	return s.campaignRepo.FindByID(ctx, id)
}

// ActivateCampaign activates a campaign (admin only)
func (s *CampaignService) ActivateCampaign(ctx context.Context, id int64) error {
	if err := s.campaignRepo.UpdateStatus(ctx, id, campaign.CampaignStatusActive); err != nil {
		return fmt.Errorf("failed to activate campaign: %w", err)
	}

	s.logger.Info("campaign activated", zap.Int64("campaign_id", id))
	return nil
}

// DeactivateCampaign deactivates a campaign (admin only)
func (s *CampaignService) DeactivateCampaign(ctx context.Context, id int64) error {
	if err := s.campaignRepo.UpdateStatus(ctx, id, campaign.CampaignStatusInactive); err != nil {
		return fmt.Errorf("failed to deactivate campaign: %w", err)
	}

	s.logger.Info("campaign deactivated", zap.Int64("campaign_id", id))
	return nil
}

// DeleteCampaign deletes a campaign (admin only)
func (s *CampaignService) DeleteCampaign(ctx context.Context, id int64) error {
	// Check if campaign has been used
	c, err := s.campaignRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if c.CurrentUses > 0 {
		return fmt.Errorf("cannot delete campaign that has been used %d times", c.CurrentUses)
	}

	if err := s.campaignRepo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete campaign", zap.Error(err))
		return fmt.Errorf("failed to delete campaign: %w", err)
	}

	s.logger.Info("campaign deleted", zap.Int64("campaign_id", id))
	return nil
}

// ========== Public/User Operations ==========

// GetCampaign retrieves a campaign by ID
func (s *CampaignService) GetCampaign(ctx context.Context, id int64) (*campaign.PromotionalCampaign, error) {
	c, err := s.campaignRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// GetCampaignByCode retrieves a campaign by promotional code
func (s *CampaignService) GetCampaignByCode(ctx context.Context, promoCode string) (*campaign.PromotionalCampaign, error) {
	c, err := s.campaignRepo.FindByPromotionalCode(ctx, strings.ToUpper(promoCode))
	if err != nil {
		return nil, err
	}

	return c, nil
}

// ListCampaigns retrieves campaigns with filters
func (s *CampaignService) ListCampaigns(ctx context.Context, filters *campaign.CampaignListFilters) (*campaign.CampaignListResponse, error) {
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

	campaigns, total, err := s.campaignRepo.List(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list campaigns: %w", err)
	}

	totalPages := int(total) / filters.PageSize
	if int(total)%filters.PageSize > 0 {
		totalPages++
	}

	return &campaign.CampaignListResponse{
		Campaigns:  campaigns,
		Total:      total,
		Page:       filters.Page,
		PageSize:   filters.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetActiveCampaigns retrieves currently active campaigns
func (s *CampaignService) GetActiveCampaigns(ctx context.Context) ([]campaign.PromotionalCampaign, error) {
	campaigns, err := s.campaignRepo.GetActiveCampaigns(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active campaigns: %w", err)
	}

	return campaigns, nil
}

// ValidateCampaign validates a promotional code and calculates discount
func (s *CampaignService) ValidateCampaign(ctx context.Context, req *campaign.ValidateCampaignRequest) (*campaign.ValidateCampaignResponse, error) {
	// Get campaign by promotional code
	c, err := s.campaignRepo.FindByPromotionalCode(ctx, strings.ToUpper(req.PromotionalCode))
	if err != nil {
		if err == xerrors.ErrNotFound {
			return &campaign.ValidateCampaignResponse{
				Valid:   false,
				Message: "Invalid promotional code",
			}, nil
		}
		return nil, err
	}

	// Check if campaign is active
	if !s.IsCampaignActive(c) {
		return &campaign.ValidateCampaignResponse{
			Valid:    false,
			Campaign: c,
			Message:  "This promotional code is not currently active",
		}, nil
	}

	// Check if plan is applicable
	if len(c.ApplicablePlans) > 0 {
		planApplicable := false
		for _, planID := range c.ApplicablePlans {
			if planID == req.PlanID {
				planApplicable = true
				break
			}
		}
		if !planApplicable {
			return &campaign.ValidateCampaignResponse{
				Valid:    false,
				Campaign: c,
				Message:  "This promotional code is not applicable to the selected plan",
			}, nil
		}
	}

	// Check user type targeting
	if len(c.TargetUserTypes) > 0 && req.UserType != "" {
		userTypeApplicable := false
		for _, targetType := range c.TargetUserTypes {
			if targetType == req.UserType {
				userTypeApplicable = true
				break
			}
		}
		if !userTypeApplicable {
			return &campaign.ValidateCampaignResponse{
				Valid:    false,
				Campaign: c,
				Message:  "This promotional code is not available for your user type",
			}, nil
		}
	}

	// Check max uses
	if c.MaxUses.Valid && c.CurrentUses >= int(c.MaxUses.Int32) {
		return &campaign.ValidateCampaignResponse{
			Valid:    false,
			Campaign: c,
			Message:  "This promotional code has reached its usage limit",
		}, nil
	}

	// TODO: Check uses per user when agent_subscriptions table is implemented

	return &campaign.ValidateCampaignResponse{
		Valid:    true,
		Campaign: c,
		Message:  "Promotional code is valid",
	}, nil
}

// ApplyCampaign applies a promotional campaign to calculate discount
func (s *CampaignService) ApplyCampaign(ctx context.Context, campaignID int64, originalPrice float64) (float64, float64, error) {
	c, err := s.campaignRepo.FindByID(ctx, campaignID)
	if err != nil {
		return 0, 0, err
	}

	discountAmount := s.CalculateDiscount(c, originalPrice)
	finalPrice := originalPrice - discountAmount

	if finalPrice < 0 {
		finalPrice = 0
	}

	return discountAmount, finalPrice, nil
}

// GetCampaignStats retrieves campaign statistics (admin only)
func (s *CampaignService) GetCampaignStats(ctx context.Context) (*campaign.CampaignStats, error) {
	stats, err := s.campaignRepo.GetStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get campaign stats: %w", err)
	}

	return stats, nil
}

// ========== Helper Methods ==========

// validateDiscountTypeAndValue validates discount type and value
func (s *CampaignService) validateDiscountTypeAndValue(discountType campaign.DiscountType, value float64) error {
	switch discountType {
	case campaign.DiscountTypePercentage:
		if value < 0 || value > 100 {
			return fmt.Errorf("percentage discount must be between 0 and 100")
		}
	case campaign.DiscountTypeFixedAmount:
		if value < 0 {
			return fmt.Errorf("fixed amount discount cannot be negative")
		}
	case campaign.DiscountTypeFreeTrial:
		// Value represents days for free trial
		if value < 1 {
			return fmt.Errorf("free trial days must be at least 1")
		}
	default:
		return fmt.Errorf("invalid discount type: %s", discountType)
	}

	return nil
}

// validatePromotionalCode validates promotional code format
func (s *CampaignService) validatePromotionalCode(code string) error {
	if len(code) < 3 || len(code) > 50 {
		return fmt.Errorf("promotional code must be between 3 and 50 characters")
	}

	// Only allow alphanumeric and hyphens
	for _, char := range code {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || char == '-' || char == '_') {
			return fmt.Errorf("promotional code can only contain letters, numbers, hyphens, and underscores")
		}
	}

	return nil
}

// generateCampaignCode generates a unique campaign code
func (s *CampaignService) generateCampaignCode(ctx context.Context, req *campaign.CreateCampaignRequest) (string, error) {
	// Format: CAMP-{DISCOUNT_TYPE}-{YEAR}{MONTH}
	// Example: CAMP-PCT-202401, CAMP-FIXED-202401

	maxAttempts := 5
	for i := 0; i < maxAttempts; i++ {
		typePrefix := ""
		switch req.DiscountType {
		case campaign.DiscountTypePercentage:
			typePrefix = "PT"
		case campaign.DiscountTypeFixedAmount:
			typePrefix = "FX"
		case campaign.DiscountTypeFreeTrial:
			typePrefix = "TL"
		}

		yearMonth := time.Now().Format("200601")
		code := fmt.Sprintf("CMP-%s-%s", typePrefix, yearMonth)

		// Add suffix if duplicate
		if i > 0 {
			code = fmt.Sprintf("%s-%d", code, i)
		}

		// Check if exists
		exists, err := s.campaignRepo.ExistsByCampaignCode(ctx, code)
		if err != nil {
			return "", fmt.Errorf("failed to check campaign code: %w", err)
		}

		if !exists {
			return code, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique campaign code after %d attempts", maxAttempts)
}

// ========== Business Logic Methods ==========

// IsCampaignActive checks if a campaign is currently active
func (s *CampaignService) IsCampaignActive(c *campaign.PromotionalCampaign) bool {
	now := time.Now()

	// Check status
	if c.Status != campaign.CampaignStatusActive {
		return false
	}

	// Check date range
	if now.Before(c.StartDate) || now.After(c.EndDate) {
		return false
	}

	// Check max uses
	if c.MaxUses.Valid && c.CurrentUses >= int(c.MaxUses.Int32) {
		return false
	}

	return true
}

// CalculateDiscount calculates the discount amount based on campaign type
func (s *CampaignService) CalculateDiscount(c *campaign.PromotionalCampaign, originalPrice float64) float64 {
	var discount float64

	switch c.DiscountType {
	case campaign.DiscountTypePercentage:
		discount = originalPrice * (c.DiscountValue / 100)
		// Apply max discount cap if set
		if c.MaxDiscountAmount.Valid && discount > c.MaxDiscountAmount.Float64 {
			discount = c.MaxDiscountAmount.Float64
		}

	case campaign.DiscountTypeFixedAmount:
		discount = c.DiscountValue
		// Don't exceed original price
		if discount > originalPrice {
			discount = originalPrice
		}

	case campaign.DiscountTypeFreeTrial:
		// For free trial, discount is 100% of the price
		discount = originalPrice
	}

	return discount
}

// GetCampaignUsagePercentage calculates usage percentage
func (s *CampaignService) GetCampaignUsagePercentage(c *campaign.PromotionalCampaign) float64 {
	if !c.MaxUses.Valid || c.MaxUses.Int32 == 0 {
		return 0
	}

	return (float64(c.CurrentUses) / float64(c.MaxUses.Int32)) * 100
}

// GetRemainingUses returns remaining uses for a campaign
func (s *CampaignService) GetRemainingUses(c *campaign.PromotionalCampaign) int {
	if !c.MaxUses.Valid {
		return -1 // Unlimited
	}

	remaining := int(c.MaxUses.Int32) - c.CurrentUses
	if remaining < 0 {
		remaining = 0
	}

	return remaining
}

// GetCampaignDaysRemaining returns days remaining for campaign
func (s *CampaignService) GetCampaignDaysRemaining(c *campaign.PromotionalCampaign) int {
	now := time.Now()
	if now.After(c.EndDate) {
		return 0
	}

	duration := c.EndDate.Sub(now)
	days := int(duration.Hours() / 24)

	return days
}

// ExtendCampaign extends campaign end date (admin only)
func (s *CampaignService) ExtendCampaign(ctx context.Context, id int64, days int) error {
	if days < 1 {
		return fmt.Errorf("extension days must be positive")
	}

	c, err := s.campaignRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	c.EndDate = c.EndDate.AddDate(0, 0, days)

	if err := s.campaignRepo.Update(ctx, id, c); err != nil {
		return fmt.Errorf("failed to extend campaign: %w", err)
	}

	s.logger.Info("campaign extended",
		zap.Int64("campaign_id", id),
		zap.Int("days", days),
		zap.Time("new_end_date", c.EndDate),
	)

	return nil
}

// IncrementCampaignUses increments usage counter
func (s *CampaignService) IncrementCampaignUses(ctx context.Context, id int64) error {
	return s.campaignRepo.IncrementUses(ctx, id)
}