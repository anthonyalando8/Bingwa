// internal/domain/customer/entity.go
package customer

import (
	"database/sql"
	"time"

	"github.com/lib/pq"
)

type AgentCustomer struct {
	ID                int64                  `json:"id" db:"id"`
	AgentIdentityID   int64                  `json:"agent_identity_id" db:"agent_identity_id"`
	CustomerReference string                 `json:"customer_reference" db:"customer_reference"`
	
	// Customer details
	FullName        sql.NullString `json:"full_name,omitempty" db:"full_name"`
	PhoneNumber     string         `json:"phone_number" db:"phone_number"`
	AltPhoneNumber  sql.NullString `json:"alt_phone_number,omitempty" db:"alt_phone_number"`
	Email           sql.NullString `json:"email,omitempty" db:"email"`
	
	// Status and flags
	IsActive    bool           `json:"is_active" db:"is_active"`
	IsVerified  bool           `json:"is_verified" db:"is_verified"`
	VerifiedAt  sql.NullTime   `json:"verified_at,omitempty" db:"verified_at"`
	
	// Additional info
	Notes    sql.NullString         `json:"notes,omitempty" db:"notes"`
	Tags     pq.StringArray         `json:"tags,omitempty" db:"tags"`
	Metadata map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	
	// Timestamps
	CreatedAt time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt time.Time    `json:"updated_at" db:"updated_at"`
	DeletedAt sql.NullTime `json:"deleted_at,omitempty" db:"deleted_at"`
}

type CustomerStats struct {
	TotalCustomers    int64 `json:"total_customers"`
	ActiveCustomers   int64 `json:"active_customers"`
	VerifiedCustomers int64 `json:"verified_customers"`
	NewThisMonth      int64 `json:"new_this_month"`
}