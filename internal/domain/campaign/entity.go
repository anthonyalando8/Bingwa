// internal/domain/campaign/entity.go
package campaign

import (
	"database/sql"
	"time"

	"github.com/lib/pq"
)

type DiscountType string

const (
	DiscountTypePercentage  DiscountType = "percentage"
	DiscountTypeFixedAmount DiscountType = "fixed_amount"
	DiscountTypeFreeTrial   DiscountType = "free_trial"
)

type CampaignStatus string

const (
	CampaignStatusActive    CampaignStatus = "active"
	CampaignStatusInactive  CampaignStatus = "inactive"
	CampaignStatusExpired   CampaignStatus = "expired"
	CampaignStatusCancelled CampaignStatus = "cancelled"
	CampaignStatusSuspended CampaignStatus = "suspended"
)

type PromotionalCampaign struct {
	ID              int64          `json:"id" db:"id"`
	CampaignCode    string         `json:"campaign_code" db:"campaign_code"`
	Name            string         `json:"name" db:"name"`
	Description     sql.NullString `json:"description,omitempty" db:"description"`
	PromotionalCode string         `json:"promotional_code" db:"promotional_code"`

	// Discount
	DiscountType      DiscountType    `json:"discount_type" db:"discount_type"`
	DiscountValue     float64         `json:"discount_value" db:"discount_value"`
	MaxDiscountAmount sql.NullFloat64 `json:"max_discount_amount,omitempty" db:"max_discount_amount"`

	// Validity
	StartDate time.Time `json:"start_date" db:"start_date"`
	EndDate   time.Time `json:"end_date" db:"end_date"`

	// Usage limits
	MaxUses       sql.NullInt32 `json:"max_uses,omitempty" db:"max_uses"`
	UsesPerUser   int           `json:"uses_per_user" db:"uses_per_user"`
	CurrentUses   int           `json:"current_uses" db:"current_uses"`

	// Targeting
	ApplicablePlans  pq.Int64Array  `json:"applicable_plans,omitempty" db:"applicable_plans"`
	TargetUserTypes  pq.StringArray `json:"target_user_types,omitempty" db:"target_user_types"`

	// Status
	Status CampaignStatus `json:"status" db:"status"`

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty" db:"metadata"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type CampaignStats struct {
	TotalCampaigns   int64   `json:"total_campaigns"`
	ActiveCampaigns  int64   `json:"active_campaigns"`
	ExpiredCampaigns int64   `json:"expired_campaigns"`
	TotalUses        int64   `json:"total_uses"`
	TotalDiscount    float64 `json:"total_discount_given"`
}