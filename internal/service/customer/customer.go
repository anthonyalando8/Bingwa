// internal/usecase/customer/customer_service.go
package customer

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"bingwa-service/internal/domain/customer"
	xerrors "bingwa-service/internal/pkg/errors"
	"bingwa-service/internal/repository/postgres"

	"github.com/lib/pq"
	"go.uber.org/zap"
)

type CustomerService struct {
	customerRepo *postgres.AgentCustomerRepository
	logger       *zap.Logger
}

func NewCustomerService(customerRepo *postgres.AgentCustomerRepository, logger *zap.Logger) *CustomerService {
	return &CustomerService{
		customerRepo: customerRepo,
		logger:       logger,
	}
}

// CreateCustomer creates a new customer for an agent
func (s *CustomerService) CreateCustomer(ctx context.Context, agentID int64, req *customer.CreateCustomerRequest) (*customer.AgentCustomer, error) {
	// Validate phone number format
	if err := s.validatePhoneNumber(req.PhoneNumber); err != nil {
		return nil, err
	}

	// Check if customer already exists for this agent with same phone
	exists, err := s.customerRepo.ExistsByAgentAndPhone(ctx, agentID, req.PhoneNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to check customer existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("customer with phone number %s already exists", req.PhoneNumber)
	}

	// Generate unique customer reference
	customerRef, err := s.generateCustomerReference(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate customer reference: %w", err)
	}

	// Create customer entity
	c := &customer.AgentCustomer{
		AgentIdentityID:   agentID,
		CustomerReference: customerRef,
		FullName:          sql.NullString{String: req.FullName, Valid: req.FullName != ""},
		PhoneNumber:       req.PhoneNumber,
		AltPhoneNumber:    sql.NullString{String: req.AltPhoneNumber, Valid: req.AltPhoneNumber != ""},
		Email:             sql.NullString{String: req.Email, Valid: req.Email != ""},
		Notes:             sql.NullString{String: req.Notes, Valid: req.Notes != ""},
		Tags:              pq.StringArray(req.Tags),
		Metadata:          req.Metadata,
		IsActive:          true,
		IsVerified:        false,
	}

	// Create in database
	if err := s.customerRepo.Create(ctx, c); err != nil {
		s.logger.Error("failed to create customer", zap.Error(err))
		return nil, fmt.Errorf("failed to create customer: %w", err)
	}

	s.logger.Info("customer created",
		zap.Int64("customer_id", c.ID),
		zap.String("customer_reference", c.CustomerReference),
		zap.Int64("agent_id", agentID),
	)

	return c, nil
}

// GetCustomer retrieves a customer by ID
func (s *CustomerService) GetCustomer(ctx context.Context, agentID, customerID int64) (*customer.AgentCustomer, error) {
	c, err := s.customerRepo.FindByID(ctx, customerID)
	if err != nil {
		return nil, err
	}

	// Verify customer belongs to agent
	if c.AgentIdentityID != agentID {
		return nil, xerrors.ErrUnauthorized
	}

	return c, nil
}

// GetCustomerByReference retrieves a customer by reference
func (s *CustomerService) GetCustomerByReference(ctx context.Context, agentID int64, reference string) (*customer.AgentCustomer, error) {
	c, err := s.customerRepo.FindByReference(ctx, reference)
	if err != nil {
		return nil, err
	}

	// Verify customer belongs to agent
	if c.AgentIdentityID != agentID {
		return nil, xerrors.ErrUnauthorized
	}

	return c, nil
}

// GetCustomerByPhone retrieves a customer by phone number
func (s *CustomerService) GetCustomerByPhone(ctx context.Context, agentID int64, phone string) (*customer.AgentCustomer, error) {
	c, err := s.customerRepo.FindByAgentAndPhone(ctx, agentID, phone)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// ListCustomers retrieves customers for an agent with filters
func (s *CustomerService) ListCustomers(ctx context.Context, agentID int64, filters *customer.CustomerListFilters) (*customer.CustomerListResponse, error) {
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

	customers, total, err := s.customerRepo.List(ctx, agentID, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list customers: %w", err)
	}

	totalPages := int(total) / filters.PageSize
	if int(total)%filters.PageSize > 0 {
		totalPages++
	}

	return &customer.CustomerListResponse{
		Customers:  customers,
		Total:      total,
		Page:       filters.Page,
		PageSize:   filters.PageSize,
		TotalPages: totalPages,
	}, nil
}

// UpdateCustomer updates a customer
func (s *CustomerService) UpdateCustomer(ctx context.Context, agentID, customerID int64, req *customer.UpdateCustomerRequest) (*customer.AgentCustomer, error) {
	// Get existing customer
	c, err := s.customerRepo.FindByID(ctx, customerID)
	if err != nil {
		return nil, err
	}

	// Verify customer belongs to agent
	if c.AgentIdentityID != agentID {
		return nil, xerrors.ErrUnauthorized
	}

	// Update fields if provided
	if req.FullName != nil {
		c.FullName = sql.NullString{String: *req.FullName, Valid: *req.FullName != ""}
	}
	if req.PhoneNumber != nil {
		// Validate phone number
		if err := s.validatePhoneNumber(*req.PhoneNumber); err != nil {
			return nil, err
		}

		// Check if new phone number conflicts with another customer
		if *req.PhoneNumber != c.PhoneNumber {
			exists, err := s.customerRepo.ExistsByAgentAndPhone(ctx, agentID, *req.PhoneNumber)
			if err != nil {
				return nil, fmt.Errorf("failed to check phone existence: %w", err)
			}
			if exists {
				return nil, fmt.Errorf("phone number already in use by another customer")
			}
		}

		c.PhoneNumber = *req.PhoneNumber
	}
	if req.AltPhoneNumber != nil {
		c.AltPhoneNumber = sql.NullString{String: *req.AltPhoneNumber, Valid: *req.AltPhoneNumber != ""}
	}
	if req.Email != nil {
		c.Email = sql.NullString{String: *req.Email, Valid: *req.Email != ""}
	}
	if req.Notes != nil {
		c.Notes = sql.NullString{String: *req.Notes, Valid: *req.Notes != ""}
	}
	if req.Tags != nil {
		c.Tags = pq.StringArray(req.Tags)
	}
	if req.Metadata != nil {
		c.Metadata = req.Metadata
	}

	// Update in database
	if err := s.customerRepo.Update(ctx, customerID, c); err != nil {
		s.logger.Error("failed to update customer", zap.Error(err))
		return nil, fmt.Errorf("failed to update customer: %w", err)
	}

	s.logger.Info("customer updated",
		zap.Int64("customer_id", customerID),
		zap.Int64("agent_id", agentID),
	)

	// Return updated customer
	return s.customerRepo.FindByID(ctx, customerID)
}

// ActivateCustomer activates a customer
func (s *CustomerService) ActivateCustomer(ctx context.Context, agentID, customerID int64) error {
	// Verify ownership
	c, err := s.customerRepo.FindByID(ctx, customerID)
	if err != nil {
		return err
	}
	if c.AgentIdentityID != agentID {
		return xerrors.ErrUnauthorized
	}

	if err := s.customerRepo.UpdateStatus(ctx, customerID, true); err != nil {
		return fmt.Errorf("failed to activate customer: %w", err)
	}

	s.logger.Info("customer activated",
		zap.Int64("customer_id", customerID),
		zap.Int64("agent_id", agentID),
	)

	return nil
}

// DeactivateCustomer deactivates a customer
func (s *CustomerService) DeactivateCustomer(ctx context.Context, agentID, customerID int64) error {
	// Verify ownership
	c, err := s.customerRepo.FindByID(ctx, customerID)
	if err != nil {
		return err
	}
	if c.AgentIdentityID != agentID {
		return xerrors.ErrUnauthorized
	}

	if err := s.customerRepo.UpdateStatus(ctx, customerID, false); err != nil {
		return fmt.Errorf("failed to deactivate customer: %w", err)
	}

	s.logger.Info("customer deactivated",
		zap.Int64("customer_id", customerID),
		zap.Int64("agent_id", agentID),
	)

	return nil
}

// VerifyCustomer marks a customer as verified
func (s *CustomerService) VerifyCustomer(ctx context.Context, agentID, customerID int64) error {
	// Verify ownership
	c, err := s.customerRepo.FindByID(ctx, customerID)
	if err != nil {
		return err
	}
	if c.AgentIdentityID != agentID {
		return xerrors.ErrUnauthorized
	}

	if c.IsVerified {
		return fmt.Errorf("customer already verified")
	}

	if err := s.customerRepo.MarkAsVerified(ctx, customerID); err != nil {
		return fmt.Errorf("failed to verify customer: %w", err)
	}

	s.logger.Info("customer verified",
		zap.Int64("customer_id", customerID),
		zap.Int64("agent_id", agentID),
	)

	return nil
}

// DeleteCustomer soft deletes a customer
func (s *CustomerService) DeleteCustomer(ctx context.Context, agentID, customerID int64) error {
	// Verify ownership
	c, err := s.customerRepo.FindByID(ctx, customerID)
	if err != nil {
		return err
	}
	if c.AgentIdentityID != agentID {
		return xerrors.ErrUnauthorized
	}

	// TODO: Check if customer has active subscriptions/pending orders
	// For now, just delete

	if err := s.customerRepo.SoftDelete(ctx, customerID); err != nil {
		s.logger.Error("failed to delete customer", zap.Error(err))
		return fmt.Errorf("failed to delete customer: %w", err)
	}

	s.logger.Info("customer deleted",
		zap.Int64("customer_id", customerID),
		zap.Int64("agent_id", agentID),
	)

	return nil
}

// GetCustomerStats retrieves statistics for an agent's customers
func (s *CustomerService) GetCustomerStats(ctx context.Context, agentID int64) (*customer.CustomerStats, error) {
	stats, err := s.customerRepo.GetStats(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer stats: %w", err)
	}

	return stats, nil
}

// ========== Helper Methods ==========

// validatePhoneNumber validates phone number format
func (s *CustomerService) validatePhoneNumber(phone string) error {
	// Remove spaces and special characters
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.ReplaceAll(phone, "(", "")
	phone = strings.ReplaceAll(phone, ")", "")

	// Check length (assuming Kenyan format: 10 digits or +254...)
	if len(phone) < 9 || len(phone) > 13 {
		return fmt.Errorf("invalid phone number length")
	}

	// Check if all characters are digits (except + at start)
	for i, char := range phone {
		if i == 0 && char == '+' {
			continue
		}
		if char < '0' || char > '9' {
			return fmt.Errorf("phone number must contain only digits")
		}
	}

	return nil
}

// generateCustomerReference generates a unique customer reference
func (s *CustomerService) generateCustomerReference(ctx context.Context, agentID int64) (string, error) {
	// Format: CUST-{AGENT_ID}-{TIMESTAMP}-{RANDOM}
	// Example: CUST-1-20240115-A3B2

	maxAttempts := 5
	for i := 0; i < maxAttempts; i++ {
		// Generate reference
		timestamp := time.Now().Format("20060102")
		random := generateRandomString(4)
		reference := fmt.Sprintf("CUST-%d-%s-%s", agentID, timestamp, random)

		// Check if exists
		exists, err := s.customerRepo.ExistsByReference(ctx, reference)
		if err != nil {
			return "", fmt.Errorf("failed to check reference: %w", err)
		}

		if !exists {
			return reference, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique customer reference after %d attempts", maxAttempts)
}

// generateRandomString generates a random alphanumeric string
func generateRandomString(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(time.Nanosecond) // Ensure uniqueness
	}
	return string(result)
}

// ========== Business Logic Methods ==========

// AddTag adds a tag to a customer
func (s *CustomerService) AddTag(ctx context.Context, agentID, customerID int64, tag string) error {
	c, err := s.customerRepo.FindByID(ctx, customerID)
	if err != nil {
		return err
	}

	if c.AgentIdentityID != agentID {
		return xerrors.ErrUnauthorized
	}

	// Check if tag already exists
	for _, t := range c.Tags {
		if t == tag {
			return fmt.Errorf("tag already exists")
		}
	}

	// Add tag
	c.Tags = append(c.Tags, tag)

	return s.customerRepo.Update(ctx, customerID, c)
}

// RemoveTag removes a tag from a customer
func (s *CustomerService) RemoveTag(ctx context.Context, agentID, customerID int64, tag string) error {
	c, err := s.customerRepo.FindByID(ctx, customerID)
	if err != nil {
		return err
	}

	if c.AgentIdentityID != agentID {
		return xerrors.ErrUnauthorized
	}

	// Remove tag
	newTags := []string{}
	for _, t := range c.Tags {
		if t != tag {
			newTags = append(newTags, t)
		}
	}

	c.Tags = pq.StringArray(newTags)

	return s.customerRepo.Update(ctx, customerID, c)
}

// BulkImportCustomers imports multiple customers at once
func (s *CustomerService) BulkImportCustomers(ctx context.Context, agentID int64, customers []customer.CreateCustomerRequest) ([]int64, []error) {
	createdIDs := []int64{}
	errors := []error{}

	for _, req := range customers {
		c, err := s.CreateCustomer(ctx, agentID, &req)
		if err != nil {
			errors = append(errors, err)
			createdIDs = append(createdIDs, 0)
		} else {
			errors = append(errors, nil)
			createdIDs = append(createdIDs, c.ID)
		}
	}

	return createdIDs, errors
}

// SearchCustomers searches customers by various criteria
func (s *CustomerService) SearchCustomers(ctx context.Context, agentID int64, query string) ([]customer.AgentCustomer, error) {
	filters := &customer.CustomerListFilters{
		Search:   query,
		Page:     1,
		PageSize: 50,
	}

	result,_ , err := s.customerRepo.List(ctx, agentID, filters)
	if err != nil {
		return nil, err
	}

	return result, nil
}