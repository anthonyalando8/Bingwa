// internal/repository/postgres/agent_customer_repository.go
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"bingwa-service/internal/domain/customer"
	xerrors "bingwa-service/internal/pkg/errors"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq"
)

type AgentCustomerRepository struct {
	db *pgxpool.Pool
}

func NewAgentCustomerRepository(db *pgxpool.Pool) *AgentCustomerRepository {
	return &AgentCustomerRepository{db: db}
}

// Create creates a new agent customer
func (r *AgentCustomerRepository) Create(ctx context.Context, c *customer.AgentCustomer) error {
	query := `
		INSERT INTO agent_customers (
			agent_identity_id, customer_reference, full_name, phone_number,
			alt_phone_number, email, notes, tags, metadata, is_active
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at
	`

	var metadataJSON []byte
	var err error

	if c.Metadata != nil {
		metadataJSON, err = json.Marshal(c.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	err = r.db.QueryRow(
		ctx, query,
		c.AgentIdentityID, c.CustomerReference, c.FullName, c.PhoneNumber,
		c.AltPhoneNumber, c.Email, c.Notes, c.Tags, metadataJSON, c.IsActive,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create customer: %w", err)
	}

	return nil
}

// FindByID retrieves a customer by ID
func (r *AgentCustomerRepository) FindByID(ctx context.Context, id int64) (*customer.AgentCustomer, error) {
	query := `
		SELECT id, agent_identity_id, customer_reference, full_name, phone_number,
		       alt_phone_number, email, is_active, is_verified, verified_at,
		       notes, tags, metadata, created_at, updated_at, deleted_at
		FROM agent_customers
		WHERE id = $1 AND deleted_at IS NULL
	`

	var c customer.AgentCustomer
	var metadataJSON []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.AgentIdentityID, &c.CustomerReference, &c.FullName, &c.PhoneNumber,
		&c.AltPhoneNumber, &c.Email, &c.IsActive, &c.IsVerified, &c.VerifiedAt,
		&c.Notes, &c.Tags, &metadataJSON, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, xerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find customer: %w", err)
	}

	// Unmarshal metadata
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &c.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &c, nil
}

// FindByReference retrieves a customer by reference
func (r *AgentCustomerRepository) FindByReference(ctx context.Context, reference string) (*customer.AgentCustomer, error) {
	query := `
		SELECT id, agent_identity_id, customer_reference, full_name, phone_number,
		       alt_phone_number, email, is_active, is_verified, verified_at,
		       notes, tags, metadata, created_at, updated_at, deleted_at
		FROM agent_customers
		WHERE customer_reference = $1 AND deleted_at IS NULL
	`

	var c customer.AgentCustomer
	var metadataJSON []byte

	err := r.db.QueryRow(ctx, query, reference).Scan(
		&c.ID, &c.AgentIdentityID, &c.CustomerReference, &c.FullName, &c.PhoneNumber,
		&c.AltPhoneNumber, &c.Email, &c.IsActive, &c.IsVerified, &c.VerifiedAt,
		&c.Notes, &c.Tags, &metadataJSON, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, xerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find customer: %w", err)
	}

	// Unmarshal metadata
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &c.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &c, nil
}

// FindByAgentAndPhone retrieves a customer by agent ID and phone number
func (r *AgentCustomerRepository) FindByAgentAndPhone(ctx context.Context, agentID int64, phone string) (*customer.AgentCustomer, error) {
	query := `
		SELECT id, agent_identity_id, customer_reference, full_name, phone_number,
		       alt_phone_number, email, is_active, is_verified, verified_at,
		       notes, tags, metadata, created_at, updated_at, deleted_at
		FROM agent_customers
		WHERE agent_identity_id = $1 AND phone_number = $2 AND deleted_at IS NULL
	`

	var c customer.AgentCustomer
	var metadataJSON []byte

	err := r.db.QueryRow(ctx, query, agentID, phone).Scan(
		&c.ID, &c.AgentIdentityID, &c.CustomerReference, &c.FullName, &c.PhoneNumber,
		&c.AltPhoneNumber, &c.Email, &c.IsActive, &c.IsVerified, &c.VerifiedAt,
		&c.Notes, &c.Tags, &metadataJSON, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, xerrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find customer: %w", err)
	}

	// Unmarshal metadata
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &c.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &c, nil
}

// Update updates a customer
func (r *AgentCustomerRepository) Update(ctx context.Context, id int64, c *customer.AgentCustomer) error {
	query := `
		UPDATE agent_customers
		SET full_name = $1, phone_number = $2, alt_phone_number = $3, email = $4,
		    notes = $5, tags = $6, metadata = $7, updated_at = $8
		WHERE id = $9 AND deleted_at IS NULL
	`

	var metadataJSON []byte
	var err error

	if c.Metadata != nil {
		metadataJSON, err = json.Marshal(c.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	result, err := r.db.Exec(
		ctx, query,
		c.FullName, c.PhoneNumber, c.AltPhoneNumber, c.Email,
		c.Notes, c.Tags, metadataJSON, time.Now(), id,
	)

	if err != nil {
		return fmt.Errorf("failed to update customer: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// UpdateStatus updates customer status (active/inactive)
func (r *AgentCustomerRepository) UpdateStatus(ctx context.Context, id int64, isActive bool) error {
	query := `UPDATE agent_customers SET is_active = $1, updated_at = $2 WHERE id = $3 AND deleted_at IS NULL`

	result, err := r.db.Exec(ctx, query, isActive, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// MarkAsVerified marks a customer as verified
func (r *AgentCustomerRepository) MarkAsVerified(ctx context.Context, id int64) error {
	query := `
		UPDATE agent_customers 
		SET is_verified = TRUE, verified_at = $1, updated_at = $2 
		WHERE id = $3 AND deleted_at IS NULL
	`

	result, err := r.db.Exec(ctx, query, time.Now(), time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to verify customer: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// SoftDelete soft deletes a customer
func (r *AgentCustomerRepository) SoftDelete(ctx context.Context, id int64) error {
	query := `UPDATE agent_customers SET deleted_at = $1, updated_at = $2 WHERE id = $3 AND deleted_at IS NULL`

	result, err := r.db.Exec(ctx, query, time.Now(), time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to delete customer: %w", err)
	}

	if result.RowsAffected() == 0 {
		return xerrors.ErrNotFound
	}

	return nil
}

// List retrieves customers with filters
func (r *AgentCustomerRepository) List(ctx context.Context, agentID int64, filters *customer.CustomerListFilters) ([]customer.AgentCustomer, int64, error) {
	// Build WHERE clause
	conditions := []string{"agent_identity_id = $1", "deleted_at IS NULL"}
	args := []interface{}{agentID}
	argPos := 2

	if filters.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argPos))
		args = append(args, *filters.IsActive)
		argPos++
	}

	if filters.IsVerified != nil {
		conditions = append(conditions, fmt.Sprintf("is_verified = $%d", argPos))
		args = append(args, *filters.IsVerified)
		argPos++
	}

	if filters.Search != "" {
		conditions = append(conditions, fmt.Sprintf(
			"(full_name ILIKE $%d OR phone_number ILIKE $%d OR email ILIKE $%d)",
			argPos, argPos, argPos,
		))
		args = append(args, "%"+filters.Search+"%")
		argPos++
	}

	if len(filters.Tags) > 0 {
		conditions = append(conditions, fmt.Sprintf("tags && $%d", argPos))
		args = append(args, pq.Array(filters.Tags))
		argPos++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM agent_customers WHERE %s", whereClause)
	var total int64
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count customers: %w", err)
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

	// Query customers
	query := fmt.Sprintf(`
		SELECT id, agent_identity_id, customer_reference, full_name, phone_number,
		       alt_phone_number, email, is_active, is_verified, verified_at,
		       notes, tags, metadata, created_at, updated_at, deleted_at
		FROM agent_customers
		WHERE %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortBy, sortOrder, argPos, argPos+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list customers: %w", err)
	}
	defer rows.Close()

	customers := []customer.AgentCustomer{}
	for rows.Next() {
		var c customer.AgentCustomer
		var metadataJSON []byte

		err := rows.Scan(
			&c.ID, &c.AgentIdentityID, &c.CustomerReference, &c.FullName, &c.PhoneNumber,
			&c.AltPhoneNumber, &c.Email, &c.IsActive, &c.IsVerified, &c.VerifiedAt,
			&c.Notes, &c.Tags, &metadataJSON, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan customer: %w", err)
		}

		// Unmarshal metadata
		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &c.Metadata)
		}

		customers = append(customers, c)
	}

	return customers, total, nil
}

// GetStats retrieves customer statistics for an agent
func (r *AgentCustomerRepository) GetStats(ctx context.Context, agentID int64) (*customer.CustomerStats, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN is_active = TRUE THEN 1 END) as active,
			COUNT(CASE WHEN is_verified = TRUE THEN 1 END) as verified,
			COUNT(CASE WHEN created_at >= date_trunc('month', NOW()) THEN 1 END) as new_this_month
		FROM agent_customers
		WHERE agent_identity_id = $1 AND deleted_at IS NULL
	`

	var stats customer.CustomerStats
	err := r.db.QueryRow(ctx, query, agentID).Scan(
		&stats.TotalCustomers,
		&stats.ActiveCustomers,
		&stats.VerifiedCustomers,
		&stats.NewThisMonth,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	return &stats, nil
}

// ExistsByAgentAndPhone checks if customer exists for agent with phone number
func (r *AgentCustomerRepository) ExistsByAgentAndPhone(ctx context.Context, agentID int64, phone string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM agent_customers 
			WHERE agent_identity_id = $1 AND phone_number = $2 AND deleted_at IS NULL
		)
	`
	var exists bool
	err := r.db.QueryRow(ctx, query, agentID, phone).Scan(&exists)
	return exists, err
}

// ExistsByReference checks if customer reference exists
func (r *AgentCustomerRepository) ExistsByReference(ctx context.Context, reference string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM agent_customers WHERE customer_reference = $1)`
	var exists bool
	err := r.db.QueryRow(ctx, query, reference).Scan(&exists)
	return exists, err
}