// internal/usecase/offer/offer_service.go
package offer

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"bingwa-service/internal/domain/offer"
	xerrors "bingwa-service/internal/pkg/errors"
	"bingwa-service/internal/repository/postgres"

	"go.uber.org/zap"
)

type OfferService struct {
	offerRepo    *postgres.AgentOfferRepository
	ussdCodeRepo *postgres.OfferUSSDCodeRepository
	logger       *zap.Logger
}

func NewOfferService(offerRepo *postgres.AgentOfferRepository, ussdCodeRepo *postgres.OfferUSSDCodeRepository, logger *zap.Logger) *OfferService {
	return &OfferService{
		offerRepo:    offerRepo,
		ussdCodeRepo: ussdCodeRepo,
		logger:       logger,
	}
}

// ========== Offer CRUD Operations ==========

// CreateOffer creates a new offer for an agent (with initial USSD code in transaction)
func (s *OfferService) CreateOffer(ctx context.Context, agentID int64, req *offer.CreateOfferRequest) (*offer.AgentOffer, error) {
	// Validate offer type and units
	if err := s.validateOfferTypeAndUnits(req.Type, req.Units); err != nil {
		return nil, err
	}

	// Validate USSD code template
	if err := s.validateUSSDCodeTemplate(req.USSDCodeTemplate); err != nil {
		return nil, err
	}

	// Generate unique offer code
	offerCode, err := s.generateOfferCode(ctx, agentID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate offer code: %w", err)
	}

	// Generate validity label if not provided
	validityLabel := req.ValidityLabel
	if validityLabel == "" {
		validityLabel = s.generateValidityLabel(req.ValidityDays)
	}

	// Create offer entity
	o := &offer.AgentOffer{
		AgentIdentityID:      agentID,
		OfferCode:            offerCode,
		Name:                 req.Name,
		Description:          sql.NullString{String: req.Description, Valid: req.Description != ""},
		Type:                 req.Type,
		Amount:               req.Amount,
		Units:                req.Units,
		Price:                req.Price,
		Currency:             strings.ToUpper(req.Currency),
		DiscountPercentage:   req.DiscountPercentage,
		ValidityDays:         req.ValidityDays,
		ValidityLabel:        sql.NullString{String: validityLabel, Valid: true},
		USSDCodeTemplate:     req.USSDCodeTemplate,
		USSDProcessingType:   req.USSDProcessingType,
		USSDExpectedResponse: sql.NullString{String: req.USSDExpectedResponse, Valid: req.USSDExpectedResponse != ""},
		USSDErrorPattern:     sql.NullString{String: req.USSDErrorPattern, Valid: req.USSDErrorPattern != ""},
		IsFeatured:           req.IsFeatured,
		IsRecurring:          req.IsRecurring,
		Status:               offer.OfferStatusActive,
		Tags:                 req.Tags,
		Metadata:             req.Metadata,
	}

	// Set optional fields
	if req.MaxPurchasesPerCustomer != nil {
		o.MaxPurchasesPerCustomer = sql.NullInt32{Int32: *req.MaxPurchasesPerCustomer, Valid: true}
	}
	if req.AvailableFrom != nil {
		o.AvailableFrom = sql.NullTime{Time: *req.AvailableFrom, Valid: true}
	}
	if req.AvailableUntil != nil {
		o.AvailableUntil = sql.NullTime{Time: *req.AvailableUntil, Valid: true}
	}

	// Create in database (repo handles USSD code creation in transaction)
	if err := s.offerRepo.Create(ctx, o); err != nil {
		s.logger.Error("failed to create offer", zap.Error(err))
		return nil, fmt.Errorf("failed to create offer: %w", err)
	}

	s.logger.Info("offer created with initial USSD code",
		zap.Int64("offer_id", o.ID),
		zap.String("offer_code", o.OfferCode),
		zap.Int64("agent_id", agentID),
	)

	// Return with primary USSD code loaded
	return s.offerRepo.FindByID(ctx, o.ID)
}

// GetOffer retrieves an offer by ID (with primary USSD code)
func (s *OfferService) GetOffer(ctx context.Context, agentID, offerID int64) (*offer.AgentOffer, error) {
	o, err := s.offerRepo.FindByID(ctx, offerID)
	if err != nil {
		return nil, err
	}

	// Verify offer belongs to agent
	if o.AgentIdentityID != agentID {
		return nil, xerrors.ErrUnauthorized
	}

	return o, nil
}

// GetOfferByCode retrieves an offer by offer code (with primary USSD code)
func (s *OfferService) GetOfferByCode(ctx context.Context, agentID int64, offerCode string) (*offer.AgentOffer, error) {
	o, err := s.offerRepo.FindByOfferCode(ctx, offerCode)
	if err != nil {
		return nil, err
	}

	// Verify offer belongs to agent
	if o.AgentIdentityID != agentID {
		return nil, xerrors.ErrUnauthorized
	}

	return o, nil
}

// ListOffers retrieves offers for an agent with filters (with primary USSD codes)
func (s *OfferService) ListOffers(ctx context.Context, agentID int64, filters *offer.OfferListFilters) (*offer.OfferListResponse, error) {
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

	offers, total, err := s.offerRepo.List(ctx, agentID, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list offers: %w", err)
	}

	totalPages := int(total) / filters.PageSize
	if int(total)%filters.PageSize > 0 {
		totalPages++
	}

	return &offer.OfferListResponse{
		Offers:     offers,
		Total:      total,
		Page:       filters.Page,
		PageSize:   filters.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetFeaturedOffers retrieves featured offers for an agent (with primary USSD codes)
func (s *OfferService) GetFeaturedOffers(ctx context.Context, agentID int64, limit int) ([]offer.AgentOffer, error) {
	if limit < 1 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	offers, err := s.offerRepo.GetFeaturedOffers(ctx, agentID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get featured offers: %w", err)
	}

	return offers, nil
}

// UpdateOffer updates an offer
func (s *OfferService) UpdateOffer(ctx context.Context, agentID, offerID int64, req *offer.UpdateOfferRequest) (*offer.AgentOffer, error) {
	// Get existing offer
	o, err := s.offerRepo.FindByID(ctx, offerID)
	if err != nil {
		return nil, err
	}

	// Verify offer belongs to agent
	if o.AgentIdentityID != agentID {
		return nil, xerrors.ErrUnauthorized
	}

	// Update fields if provided
	if req.Name != nil {
		o.Name = *req.Name
	}
	if req.Description != nil {
		o.Description = sql.NullString{String: *req.Description, Valid: *req.Description != ""}
	}
	if req.Type != nil {
		o.Type = *req.Type
	}
	if req.Amount != nil {
		o.Amount = *req.Amount
	}
	if req.Units != nil {
		o.Units = *req.Units
	}
	if req.Price != nil {
		o.Price = *req.Price
	}
	if req.DiscountPercentage != nil {
		o.DiscountPercentage = *req.DiscountPercentage
	}
	if req.ValidityDays != nil {
		o.ValidityDays = *req.ValidityDays
		// Regenerate validity label
		o.ValidityLabel = sql.NullString{String: s.generateValidityLabel(*req.ValidityDays), Valid: true}
	}
	if req.ValidityLabel != nil {
		o.ValidityLabel = sql.NullString{String: *req.ValidityLabel, Valid: *req.ValidityLabel != ""}
	}
	if req.USSDCodeTemplate != nil {
		if err := s.validateUSSDCodeTemplate(*req.USSDCodeTemplate); err != nil {
			return nil, err
		}
		o.USSDCodeTemplate = *req.USSDCodeTemplate
	}
	if req.USSDProcessingType != nil {
		o.USSDProcessingType = *req.USSDProcessingType
	}
	if req.USSDExpectedResponse != nil {
		o.USSDExpectedResponse = sql.NullString{String: *req.USSDExpectedResponse, Valid: *req.USSDExpectedResponse != ""}
	}
	if req.USSDErrorPattern != nil {
		o.USSDErrorPattern = sql.NullString{String: *req.USSDErrorPattern, Valid: *req.USSDErrorPattern != ""}
	}
	if req.IsFeatured != nil {
		o.IsFeatured = *req.IsFeatured
	}
	if req.IsRecurring != nil {
		o.IsRecurring = *req.IsRecurring
	}
	if req.MaxPurchasesPerCustomer != nil {
		o.MaxPurchasesPerCustomer = sql.NullInt32{Int32: *req.MaxPurchasesPerCustomer, Valid: true}
	}
	if req.AvailableFrom != nil {
		o.AvailableFrom = sql.NullTime{Time: *req.AvailableFrom, Valid: true}
	}
	if req.AvailableUntil != nil {
		o.AvailableUntil = sql.NullTime{Time: *req.AvailableUntil, Valid: true}
	}
	if req.Tags != nil {
		o.Tags = req.Tags
	}
	if req.Metadata != nil {
		o.Metadata = req.Metadata
	}

	// Validate type and units combination
	if err := s.validateOfferTypeAndUnits(o.Type, o.Units); err != nil {
		return nil, err
	}

	// Update in database
	if err := s.offerRepo.Update(ctx, offerID, o); err != nil {
		s.logger.Error("failed to update offer", zap.Error(err))
		return nil, fmt.Errorf("failed to update offer: %w", err)
	}

	s.logger.Info("offer updated",
		zap.Int64("offer_id", offerID),
		zap.Int64("agent_id", agentID),
	)

	// Return updated offer
	return s.offerRepo.FindByID(ctx, offerID)
}

// ActivateOffer activates an offer
func (s *OfferService) ActivateOffer(ctx context.Context, agentID, offerID int64) error {
	// Verify ownership
	o, err := s.offerRepo.FindByID(ctx, offerID)
	if err != nil {
		return err
	}
	if o.AgentIdentityID != agentID {
		return xerrors.ErrUnauthorized
	}

	if err := s.offerRepo.UpdateStatus(ctx, offerID, offer.OfferStatusActive); err != nil {
		return fmt.Errorf("failed to activate offer: %w", err)
	}

	s.logger.Info("offer activated",
		zap.Int64("offer_id", offerID),
		zap.Int64("agent_id", agentID),
	)

	return nil
}

// DeactivateOffer deactivates an offer
func (s *OfferService) DeactivateOffer(ctx context.Context, agentID, offerID int64) error {
	// Verify ownership
	o, err := s.offerRepo.FindByID(ctx, offerID)
	if err != nil {
		return err
	}
	if o.AgentIdentityID != agentID {
		return xerrors.ErrUnauthorized
	}

	if err := s.offerRepo.UpdateStatus(ctx, offerID, offer.OfferStatusInactive); err != nil {
		return fmt.Errorf("failed to deactivate offer: %w", err)
	}

	s.logger.Info("offer deactivated",
		zap.Int64("offer_id", offerID),
		zap.Int64("agent_id", agentID),
	)

	return nil
}

// PauseOffer pauses an offer
func (s *OfferService) PauseOffer(ctx context.Context, agentID, offerID int64) error {
	// Verify ownership
	o, err := s.offerRepo.FindByID(ctx, offerID)
	if err != nil {
		return err
	}
	if o.AgentIdentityID != agentID {
		return xerrors.ErrUnauthorized
	}

	if err := s.offerRepo.UpdateStatus(ctx, offerID, offer.OfferStatusPaused); err != nil {
		return fmt.Errorf("failed to pause offer: %w", err)
	}

	s.logger.Info("offer paused",
		zap.Int64("offer_id", offerID),
		zap.Int64("agent_id", agentID),
	)

	return nil
}

// DeleteOffer soft deletes an offer
func (s *OfferService) DeleteOffer(ctx context.Context, agentID, offerID int64) error {
	// Verify ownership
	o, err := s.offerRepo.FindByID(ctx, offerID)
	if err != nil {
		return err
	}
	if o.AgentIdentityID != agentID {
		return xerrors.ErrUnauthorized
	}

	if err := s.offerRepo.SoftDelete(ctx, offerID); err != nil {
		s.logger.Error("failed to delete offer", zap.Error(err))
		return fmt.Errorf("failed to delete offer: %w", err)
	}

	s.logger.Info("offer deleted",
		zap.Int64("offer_id", offerID),
		zap.Int64("agent_id", agentID),
	)

	return nil
}

// GetOfferStats retrieves statistics for an agent's offers
func (s *OfferService) GetOfferStats(ctx context.Context, agentID int64) (*offer.OfferStats, error) {
	stats, err := s.offerRepo.GetStats(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get offer stats: %w", err)
	}

	return stats, nil
}

// CloneOffer creates a copy of an existing offer
func (s *OfferService) CloneOffer(ctx context.Context, agentID, offerID int64, newName string) (*offer.AgentOffer, error) {
	// Get original offer
	original, err := s.offerRepo.FindByID(ctx, offerID)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if original.AgentIdentityID != agentID {
		return nil, xerrors.ErrUnauthorized
	}

	// Create request from original
	req := &offer.CreateOfferRequest{
		Name:                    newName,
		Description:             original.Description.String,
		Type:                    original.Type,
		Amount:                  original.Amount,
		Units:                   original.Units,
		Price:                   original.Price,
		Currency:                original.Currency,
		DiscountPercentage:      original.DiscountPercentage,
		ValidityDays:            original.ValidityDays,
		ValidityLabel:           original.ValidityLabel.String,
		USSDCodeTemplate:        original.USSDCodeTemplate,
		USSDProcessingType:      original.USSDProcessingType,
		USSDExpectedResponse:    original.USSDExpectedResponse.String,
		USSDErrorPattern:        original.USSDErrorPattern.String,
		IsFeatured:              false, // Clones are not featured by default
		IsRecurring:             original.IsRecurring,
		Tags:                    original.Tags,
		Metadata:                original.Metadata,
	}

	if original.MaxPurchasesPerCustomer.Valid {
		maxPurchases := original.MaxPurchasesPerCustomer.Int32
		req.MaxPurchasesPerCustomer = &maxPurchases
	}

	return s.CreateOffer(ctx, agentID, req)
}

// SearchOffers searches offers by various criteria
func (s *OfferService) SearchOffers(ctx context.Context, agentID int64, query string) ([]offer.AgentOffer, error) {
	filters := &offer.OfferListFilters{
		Search:   query,
		Page:     1,
		PageSize: 50,
	}

	result, _, err := s.offerRepo.List(ctx, agentID, filters)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// ========== USSD Code Management ==========

// AddUSSDCode adds a new USSD code to an offer with automatic priority management
func (s *OfferService) AddUSSDCode(ctx context.Context, agentID, offerID int64, req *offer.AddUSSDCodeRequest) (*offer.OfferUSSDCode, error) {
	// Verify offer ownership
	existingOffer, err := s.offerRepo.FindByID(ctx, offerID)
	if err != nil {
		return nil, err
	}

	if existingOffer.AgentIdentityID != agentID {
		return nil, xerrors.ErrUnauthorized
	}

	// Check if USSD code already exists for this offer
	exists, err := s.ussdCodeRepo.ExistsByOfferAndCode(ctx, offerID, req.USSDCode)
	if err != nil {
		return nil, fmt.Errorf("failed to check USSD code existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("USSD code already exists for this offer")
	}

	// If no priority provided or priority is 1, set as new priority 1 and shift others
	priority := req.Priority
	if priority == 0 || priority == 1 {
		priority = 1
		// Shift existing priorities down by 1
		if err := s.shiftUSSDPriorities(ctx, offerID, 1); err != nil {
			return nil, fmt.Errorf("failed to shift priorities: %w", err)
		}
	} else {
		// Validate that no other code has this priority
		codes, err := s.ussdCodeRepo.ListByOfferID(ctx, offerID)
		if err != nil {
			return nil, fmt.Errorf("failed to list USSD codes: %w", err)
		}

		for _, code := range codes {
			if code.Priority == priority {
				return nil, fmt.Errorf("USSD code with priority %d already exists", priority)
			}
		}
	}

	// Create new USSD code
	ussdCode := &offer.OfferUSSDCode{
		OfferID:        offerID,
		USSDCode:       req.USSDCode,
		Priority:       priority,
		IsActive:       true,
		ProcessingType: req.ProcessingType,
	}

	if req.SignaturePattern != "" {
		ussdCode.SignaturePattern = sql.NullString{String: req.SignaturePattern, Valid: true}
	}
	if req.ExpectedResponse != "" {
		ussdCode.ExpectedResponse = sql.NullString{String: req.ExpectedResponse, Valid: true}
	}
	if req.ErrorPattern != "" {
		ussdCode.ErrorPattern = sql.NullString{String: req.ErrorPattern, Valid: true}
	}
	if req.Metadata != nil {
		ussdCode.Metadata = req.Metadata
	}

	if err := s.ussdCodeRepo.Create(ctx, ussdCode); err != nil {
		return nil, fmt.Errorf("failed to add USSD code: %w", err)
	}

	s.logger.Info("USSD code added to offer",
		zap.Int64("offer_id", offerID),
		zap.Int64("ussd_code_id", ussdCode.ID),
		zap.Int("priority", priority),
	)

	return ussdCode, nil
}

// shiftUSSDPriorities shifts all priorities >= startPriority down by 1
func (s *OfferService) shiftUSSDPriorities(ctx context.Context, offerID int64, startPriority int) error {
	codes, err := s.ussdCodeRepo.ListByOfferID(ctx, offerID)
	if err != nil {
		return fmt.Errorf("failed to list USSD codes: %w", err)
	}

	// Sort by priority to shift in correct order (highest first)
	sort.Slice(codes, func(i, j int) bool {
		return codes[i].Priority > codes[j].Priority
	})

	for _, code := range codes {
		if code.Priority >= startPriority {
			newPriority := code.Priority + 1
			if err := s.ussdCodeRepo.UpdatePriority(ctx, code.ID, newPriority); err != nil {
				return fmt.Errorf("failed to update priority for code %d: %w", code.ID, err)
			}
		}
	}

	return nil
}

// UpdateUSSDCode updates a USSD code
func (s *OfferService) UpdateUSSDCode(ctx context.Context, agentID, offerID, ussdCodeID int64, req *offer.UpdateUSSDCodeRequest) (*offer.OfferUSSDCode, error) {
	// Verify offer ownership
	existingOffer, err := s.offerRepo.FindByID(ctx, offerID)
	if err != nil {
		return nil, err
	}

	if existingOffer.AgentIdentityID != agentID {
		return nil, xerrors.ErrUnauthorized
	}

	// Get existing USSD code
	ussdCode, err := s.ussdCodeRepo.FindByID(ctx, ussdCodeID)
	if err != nil {
		return nil, err
	}

	// Verify USSD code belongs to this offer
	if ussdCode.OfferID != offerID {
		return nil, xerrors.ErrUnauthorized
	}

	oldPriority := ussdCode.Priority

	// Update fields
	if req.USSDCode != nil {
		// Check if new USSD code doesn't exist
		exists, err := s.ussdCodeRepo.ExistsByOfferAndCode(ctx, offerID, *req.USSDCode)
		if err != nil {
			return nil, fmt.Errorf("failed to check USSD code: %w", err)
		}
		if exists && *req.USSDCode != ussdCode.USSDCode {
			return nil, fmt.Errorf("USSD code already exists for this offer")
		}
		ussdCode.USSDCode = *req.USSDCode
	}

	if req.SignaturePattern != nil {
		ussdCode.SignaturePattern = sql.NullString{String: *req.SignaturePattern, Valid: *req.SignaturePattern != ""}
	}

	if req.Priority != nil {
		newPriority := *req.Priority

		// Check if priority is changing
		if newPriority != oldPriority {
			// Check if another code has this priority
			codes, err := s.ussdCodeRepo.ListByOfferID(ctx, offerID)
			if err != nil {
				return nil, fmt.Errorf("failed to list USSD codes: %w", err)
			}

			for _, code := range codes {
				if code.ID != ussdCodeID && code.Priority == newPriority {
					return nil, fmt.Errorf("USSD code with priority %d already exists", newPriority)
				}
			}

			ussdCode.Priority = newPriority
		}
	}

	if req.IsActive != nil {
		ussdCode.IsActive = *req.IsActive
	}

	if req.ExpectedResponse != nil {
		ussdCode.ExpectedResponse = sql.NullString{String: *req.ExpectedResponse, Valid: *req.ExpectedResponse != ""}
	}

	if req.ErrorPattern != nil {
		ussdCode.ErrorPattern = sql.NullString{String: *req.ErrorPattern, Valid: *req.ErrorPattern != ""}
	}

	if req.ProcessingType != nil {
		ussdCode.ProcessingType = *req.ProcessingType
	}

	if req.Metadata != nil {
		ussdCode.Metadata = req.Metadata
	}

	if err := s.ussdCodeRepo.Update(ctx, ussdCodeID, ussdCode); err != nil {
		return nil, fmt.Errorf("failed to update USSD code: %w", err)
	}

	s.logger.Info("USSD code updated",
		zap.Int64("ussd_code_id", ussdCodeID),
		zap.Int64("offer_id", offerID),
	)

	return s.ussdCodeRepo.FindByID(ctx, ussdCodeID)
}

// SetUSSDCodeAsPrimary sets a USSD code as priority 1 (primary)
func (s *OfferService) SetUSSDCodeAsPrimary(ctx context.Context, agentID, offerID, ussdCodeID int64) error {
	// Verify offer ownership
	existingOffer, err := s.offerRepo.FindByID(ctx, offerID)
	if err != nil {
		return err
	}

	if existingOffer.AgentIdentityID != agentID {
		return xerrors.ErrUnauthorized
	}

	// Get existing USSD code
	ussdCode, err := s.ussdCodeRepo.FindByID(ctx, ussdCodeID)
	if err != nil {
		return err
	}

	// Verify USSD code belongs to this offer
	if ussdCode.OfferID != offerID {
		return xerrors.ErrUnauthorized
	}

	// If already priority 1, nothing to do
	if ussdCode.Priority == 1 {
		return nil
	}

	oldPriority := ussdCode.Priority

	// Shift priorities between 1 and oldPriority up by 1
	codes, err := s.ussdCodeRepo.ListByOfferID(ctx, offerID)
	if err != nil {
		return fmt.Errorf("failed to list USSD codes: %w", err)
	}

	// Sort by priority
	sort.Slice(codes, func(i, j int) bool {
		return codes[i].Priority < codes[j].Priority
	})

	for _, code := range codes {
		if code.ID != ussdCodeID && code.Priority >= 1 && code.Priority < oldPriority {
			newPriority := code.Priority + 1
			if err := s.ussdCodeRepo.UpdatePriority(ctx, code.ID, newPriority); err != nil {
				return fmt.Errorf("failed to update priority: %w", err)
			}
		}
	}

	// Set this code as priority 1
	if err := s.ussdCodeRepo.UpdatePriority(ctx, ussdCodeID, 1); err != nil {
		return fmt.Errorf("failed to set as primary: %w", err)
	}

	s.logger.Info("USSD code set as primary",
		zap.Int64("ussd_code_id", ussdCodeID),
		zap.Int64("offer_id", offerID),
	)

	return nil
}

// ReorderUSSDCodes reorders USSD codes priorities
func (s *OfferService) ReorderUSSDCodes(ctx context.Context, agentID, offerID int64, req *offer.ReorderUSSDCodesRequest) error {
	// Verify offer ownership
	existingOffer, err := s.offerRepo.FindByID(ctx, offerID)
	if err != nil {
		return err
	}

	if existingOffer.AgentIdentityID != agentID {
		return xerrors.ErrUnauthorized
	}

	// Validate no duplicate priorities
	priorityMap := make(map[int]bool)
	for _, code := range req.Codes {
		if priorityMap[code.Priority] {
			return fmt.Errorf("duplicate priority %d in request", code.Priority)
		}
		priorityMap[code.Priority] = true
	}

	// Verify all codes belong to this offer
	for _, code := range req.Codes {
		ussdCode, err := s.ussdCodeRepo.FindByID(ctx, code.ID)
		if err != nil {
			return fmt.Errorf("USSD code %d not found", code.ID)
		}
		if ussdCode.OfferID != offerID {
			return fmt.Errorf("USSD code %d does not belong to offer %d", code.ID, offerID)
		}
	}

	// Update priorities
	for _, code := range req.Codes {
		if err := s.ussdCodeRepo.UpdatePriority(ctx, code.ID, code.Priority); err != nil {
			return fmt.Errorf("failed to update priority for code %d: %w", code.ID, err)
		}
	}

	s.logger.Info("USSD codes reordered",
		zap.Int64("offer_id", offerID),
		zap.Int("count", len(req.Codes)),
	)

	return nil
}

// DeleteUSSDCode deletes a USSD code
func (s *OfferService) DeleteUSSDCode(ctx context.Context, agentID, offerID, ussdCodeID int64) error {
	// Verify offer ownership
	existingOffer, err := s.offerRepo.FindByID(ctx, offerID)
	if err != nil {
		return err
	}

	if existingOffer.AgentIdentityID != agentID {
		return xerrors.ErrUnauthorized
	}

	// Get existing USSD code
	ussdCode, err := s.ussdCodeRepo.FindByID(ctx, ussdCodeID)
	if err != nil {
		return err
	}

	// Verify USSD code belongs to this offer
	if ussdCode.OfferID != offerID {
		return xerrors.ErrUnauthorized
	}

	// Check if this is the last USSD code
	codes, err := s.ussdCodeRepo.ListByOfferID(ctx, offerID)
	if err != nil {
		return fmt.Errorf("failed to list USSD codes: %w", err)
	}

	if len(codes) == 1 {
		return fmt.Errorf("cannot delete the last USSD code. Offer must have at least one USSD code")
	}

	deletedPriority := ussdCode.Priority

	// Delete the code
	if err := s.ussdCodeRepo.Delete(ctx, ussdCodeID); err != nil {
		return fmt.Errorf("failed to delete USSD code: %w", err)
	}

	// Shift priorities up for codes with higher priority
	for _, code := range codes {
		if code.ID != ussdCodeID && code.Priority > deletedPriority {
			newPriority := code.Priority - 1
			if err := s.ussdCodeRepo.UpdatePriority(ctx, code.ID, newPriority); err != nil {
				s.logger.Error("failed to adjust priority after deletion",
					zap.Int64("code_id", code.ID),
					zap.Error(err),
				)
			}
		}
	}

	s.logger.Info("USSD code deleted",
		zap.Int64("ussd_code_id", ussdCodeID),
		zap.Int64("offer_id", offerID),
	)

	return nil
}

// ListUSSDCodes lists all USSD codes for an offer
func (s *OfferService) ListUSSDCodes(ctx context.Context, agentID, offerID int64) ([]offer.OfferUSSDCode, error) {
	// Verify offer ownership
	existingOffer, err := s.offerRepo.FindByID(ctx, offerID)
	if err != nil {
		return nil, err
	}

	if existingOffer.AgentIdentityID != agentID {
		return nil, xerrors.ErrUnauthorized
	}

	codes, err := s.ussdCodeRepo.ListByOfferID(ctx, offerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list USSD codes: %w", err)
	}

	return codes, nil
}

// GetActiveUSSDCodes gets active USSD codes sorted by priority
func (s *OfferService) GetActiveUSSDCodes(ctx context.Context, agentID, offerID int64) ([]offer.OfferUSSDCode, error) {
	// Verify offer ownership
	existingOffer, err := s.offerRepo.FindByID(ctx, offerID)
	if err != nil {
		return nil, err
	}

	if existingOffer.AgentIdentityID != agentID {
		return nil, xerrors.ErrUnauthorized
	}

	codes, err := s.ussdCodeRepo.GetActiveCodesByPriority(ctx, offerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active codes: %w", err)
	}

	return codes, nil
}

// ToggleUSSDCodeStatus toggles USSD code active status
func (s *OfferService) ToggleUSSDCodeStatus(ctx context.Context, agentID, offerID, ussdCodeID int64, isActive bool) error {
	// Verify offer ownership
	existingOffer, err := s.offerRepo.FindByID(ctx, offerID)
	if err != nil {
		return err
	}

	if existingOffer.AgentIdentityID != agentID {
		return xerrors.ErrUnauthorized
	}

	// Get existing USSD code
	ussdCode, err := s.ussdCodeRepo.FindByID(ctx, ussdCodeID)
	if err != nil {
		return err
	}

	// Verify USSD code belongs to this offer
	if ussdCode.OfferID != offerID {
		return xerrors.ErrUnauthorized
	}

	// If deactivating, ensure at least one active code remains
	if !isActive {
		activeCodes, err := s.ussdCodeRepo.GetActiveCodesByPriority(ctx, offerID)
		if err != nil {
			return fmt.Errorf("failed to get active codes: %w", err)
		}

		if len(activeCodes) == 1 && activeCodes[0].ID == ussdCodeID {
			return fmt.Errorf("cannot deactivate the last active USSD code")
		}
	}

	if err := s.ussdCodeRepo.ToggleActive(ctx, ussdCodeID, isActive); err != nil {
		return fmt.Errorf("failed to toggle status: %w", err)
	}

	s.logger.Info("USSD code status toggled",
		zap.Int64("ussd_code_id", ussdCodeID),
		zap.Bool("is_active", isActive),
	)

	return nil
}

// RecordUSSDResult records the result of a USSD execution
func (s *OfferService) RecordUSSDResult(ctx context.Context, agentID, offerID int64, req *offer.RecordUSSDResultRequest) error {
	// Verify offer ownership
	existingOffer, err := s.offerRepo.FindByID(ctx, offerID)
	if err != nil {
		return err
	}

	if existingOffer.AgentIdentityID != agentID {
		return xerrors.ErrUnauthorized
	}

	// Get USSD code
	ussdCode, err := s.ussdCodeRepo.FindByID(ctx, req.USSDCodeID)
	if err != nil {
		return err
	}

	// Verify USSD code belongs to this offer
	if ussdCode.OfferID != offerID {
		return xerrors.ErrUnauthorized
	}

	// Record result
	if req.Success {
		if err := s.ussdCodeRepo.RecordSuccess(ctx, req.USSDCodeID); err != nil {
			return fmt.Errorf("failed to record success: %w", err)
		}
	} else {
		if err := s.ussdCodeRepo.RecordFailure(ctx, req.USSDCodeID); err != nil {
			return fmt.Errorf("failed to record failure: %w", err)
		}
	}

	s.logger.Info("USSD result recorded",
		zap.Int64("ussd_code_id", req.USSDCodeID),
		zap.Bool("success", req.Success),
	)

	return nil
}

// GetUSSDCodeStats gets statistics for USSD codes
func (s *OfferService) GetUSSDCodeStats(ctx context.Context, agentID, offerID int64) (*offer.USSDCodeStats, error) {
	// Verify offer ownership
	existingOffer, err := s.offerRepo.FindByID(ctx, offerID)
	if err != nil {
		return nil, err
	}

	if existingOffer.AgentIdentityID != agentID {
		return nil, xerrors.ErrUnauthorized
	}

	stats, err := s.ussdCodeRepo.GetStats(ctx, offerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	return stats, nil
}

// GetPrimaryUSSDCode gets the primary (highest priority) USSD code
func (s *OfferService) GetPrimaryUSSDCode(ctx context.Context, agentID, offerID int64) (*offer.OfferUSSDCode, error) {
	// Verify offer ownership
	existingOffer, err := s.offerRepo.FindByID(ctx, offerID)
	if err != nil {
		return nil, err
	}

	if existingOffer.AgentIdentityID != agentID {
		return nil, xerrors.ErrUnauthorized
	}

	code, err := s.ussdCodeRepo.GetPrimaryUSSDCode(ctx, offerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get primary USSD code: %w", err)
	}

	return code, nil
}

// ========== Business Logic & Helper Methods ==========

// validateOfferTypeAndUnits validates that units match the offer type
func (s *OfferService) validateOfferTypeAndUnits(offerType offer.OfferType, units offer.OfferUnits) error {
	validCombinations := map[offer.OfferType][]offer.OfferUnits{
		offer.OfferTypeData: {
			offer.UnitsGB,
			offer.UnitsMB,
			offer.UnitsKB,
		},
		offer.OfferTypeSMS: {
			offer.UnitsSMS,
		},
		offer.OfferTypeVoice: {
			offer.UnitsMinutes,
		},
		offer.OfferTypeCombo: {
			offer.UnitsUnits, // Combo can use generic "units"
		},
	}

	validUnits, exists := validCombinations[offerType]
	if !exists {
		return fmt.Errorf("invalid offer type: %s", offerType)
	}

	for _, validUnit := range validUnits {
		if units == validUnit {
			return nil
		}
	}

	return fmt.Errorf("invalid units '%s' for offer type '%s'", units, offerType)
}

// validateUSSDCodeTemplate validates USSD code template format
func (s *OfferService) validateUSSDCodeTemplate(template string) error {
	if !strings.HasPrefix(template, "*") {
		return fmt.Errorf("USSD code must start with *")
	}
	if !strings.HasSuffix(template, "#") {
		return fmt.Errorf("USSD code must end with #")
	}
	if len(template) < 3 {
		return fmt.Errorf("USSD code is too short")
	}

	// Check for valid placeholders
	validPlaceholders := []string{"{phone}", "{amount}", "{customer_phone}"}
	for _, placeholder := range validPlaceholders {
		if strings.Contains(template, placeholder) {
			return nil // At least one valid placeholder found
		}
	}

	// If no placeholders, it's still valid (static USSD)
	return nil
}

// generateOfferCode generates a unique offer code
func (s *OfferService) generateOfferCode(ctx context.Context, agentID int64, req *offer.CreateOfferRequest) (string, error) {
	// Format: {TYPE}-{AMOUNT}{UNITS}-{VALIDITY}D-{AGENT_ID}
	// Example: DATA-5GB-30D-1, SMS-100SMS-7D-1

	maxAttempts := 5
	for i := 0; i < maxAttempts; i++ {
		// Generate code
		typePrefix := strings.ToUpper(string(req.Type))
		amountStr := fmt.Sprintf("%.0f", req.Amount)
		unitsStr := strings.ToUpper(string(req.Units))
		validityStr := fmt.Sprintf("%dD", req.ValidityDays)

		code := fmt.Sprintf("%s-%s%s-%s-%d",
			typePrefix,
			amountStr,
			unitsStr,
			validityStr,
			agentID,
		)

		// Add random suffix if duplicate
		if i > 0 {
			code = fmt.Sprintf("%s-%d", code, i)
		}

		// Check if exists
		exists, err := s.offerRepo.ExistsByOfferCode(ctx, code)
		if err != nil {
			return "", fmt.Errorf("failed to check offer code: %w", err)
		}

		if !exists {
			return code, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique offer code after %d attempts", maxAttempts)
}

// generateValidityLabel generates a human-readable validity label
func (s *OfferService) generateValidityLabel(days int) string {
	if days == 1 {
		return "1 day"
	}
	if days == 7 {
		return "1 week"
	}
	if days == 30 {
		return "1 month"
	}
	if days == 90 {
		return "3 months"
	}
	if days == 365 {
		return "1 year"
	}
	if days%30 == 0 {
		return fmt.Sprintf("%d months", days/30)
	}
	if days%7 == 0 {
		return fmt.Sprintf("%d weeks", days/7)
	}
	return fmt.Sprintf("%d days", days)
}


// CalculateDiscountedPrice calculates price after discount
func (s *OfferService) CalculateDiscountedPrice(o *offer.AgentOffer) float64 {
	if o.DiscountPercentage <= 0 {
		return o.Price
	}

	discount := o.Price * (o.DiscountPercentage / 100)
	return o.Price - discount
}

// IsOfferAvailable checks if offer is currently available
func (s *OfferService) IsOfferAvailable(o *offer.AgentOffer) bool {
	now := time.Now()

	// Check status
	if o.Status != offer.OfferStatusActive {
		return false
	}

	// Check availability window
	if o.AvailableFrom.Valid && now.Before(o.AvailableFrom.Time) {
		return false
	}
	if o.AvailableUntil.Valid && now.After(o.AvailableUntil.Time) {
		return false
	}

	return true
}

// ValidateOfferPurchase validates if a customer can purchase an offer
func (s *OfferService) ValidateOfferPurchase(ctx context.Context, o *offer.AgentOffer, customerID int64) error {
	// Check if offer is available
	if !s.IsOfferAvailable(o) {
		return fmt.Errorf("offer is not currently available")
	}

	// TODO: Check max purchases per customer when offer_redemptions table is implemented
	// if o.MaxPurchasesPerCustomer.Valid {
	//     count := s.getPurchaseCount(ctx, o.ID, customerID)
	//     if count >= int(o.MaxPurchasesPerCustomer.Int32) {
	//         return fmt.Errorf("maximum purchase limit reached")
	//     }
	// }

	return nil
}



// GetOffersByAmount retrieves offers by exact amount
func (s *OfferService) GetOffersByAmount(ctx context.Context, agentID int64, amount float64) ([]offer.AgentOffer, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than 0")
	}

	offers, err := s.offerRepo.FindByAmount(ctx, agentID, amount)
	if err != nil {
		return nil, fmt.Errorf("failed to get offers by amount: %w", err)
	}

	return offers, nil
}

// GetOffersByAmountRange retrieves offers within amount range
func (s *OfferService) GetOffersByAmountRange(ctx context.Context, agentID int64, minAmount, maxAmount float64) ([]offer.AgentOffer, error) {
	if minAmount < 0 {
		return nil, fmt.Errorf("minimum amount cannot be negative")
	}
	if maxAmount < minAmount {
		return nil, fmt.Errorf("maximum amount must be greater than or equal to minimum amount")
	}

	offers, err := s.offerRepo.FindByAmountRange(ctx, agentID, minAmount, maxAmount)
	if err != nil {
		return nil, fmt.Errorf("failed to get offers by amount range: %w", err)
	}

	return offers, nil
}

// GetOffersByTypeAndAmount retrieves offers by type and amount
func (s *OfferService) GetOffersByTypeAndAmount(ctx context.Context, agentID int64, offerType offer.OfferType, amount float64) ([]offer.AgentOffer, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than 0")
	}

	offers, err := s.offerRepo.FindByTypeAndAmount(ctx, agentID, offerType, amount)
	if err != nil {
		return nil, fmt.Errorf("failed to get offers by type and amount: %w", err)
	}

	return offers, nil
}


// GetOfferByPrice retrieves a single offer by price
func (s *OfferService) GetOfferByPrice(ctx context.Context, agentID int64, price float64) (*offer.AgentOffer, error) {
	if price <= 0 {
		return nil, fmt.Errorf("price must be greater than 0")
	}

	o, err := s.offerRepo.FindByPrice(ctx, agentID, price)
	if err != nil {
		return nil, fmt.Errorf("failed to get offer by price: %w", err)
	}

	return o, nil
}

// GetOfferByPriceAndType retrieves a single offer by price and type
func (s *OfferService) GetOfferByPriceAndType(ctx context.Context, agentID int64, price float64, offerType offer.OfferType) (*offer.AgentOffer, error) {
	if price <= 0 {
		return nil, fmt.Errorf("price must be greater than 0")
	}

	o, err := s.offerRepo.FindByPriceAndType(ctx, agentID, price, offerType)
	if err != nil {
		return nil, fmt.Errorf("failed to get offer by price and type: %w", err)
	}

	return o, nil
}