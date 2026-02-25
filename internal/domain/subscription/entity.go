// internal/domain/subscription/entity.go
package subscription

import (
	"database/sql"
	"time"
)

type RenewalPeriod string

const (
	RenewalDaily     RenewalPeriod = "daily"
	RenewalWeekly    RenewalPeriod = "weekly"
	RenewalMonthly   RenewalPeriod = "monthly"
	RenewalQuarterly RenewalPeriod = "quarterly"
	RenewalYearly    RenewalPeriod = "yearly"
)



const (
	StatusActive    SubscriptionStatus = "active"
	StatusInactive  SubscriptionStatus = "inactive"
	StatusExpired   SubscriptionStatus = "expired"
	StatusCancelled SubscriptionStatus = "cancelled"
	StatusSuspended SubscriptionStatus = "suspended"
)

type SubscriptionPlan struct {
	ID          int64                  `json:"id" db:"id"`
	PlanCode    string                 `json:"plan_code" db:"plan_code"`
	Name        string                 `json:"name" db:"name"`
	Description sql.NullString         `json:"description,omitempty" db:"description"`
	
	// Pricing
	Price       float64 `json:"price" db:"price"`
	Currency    string  `json:"currency" db:"currency"`
	SetupFee    float64 `json:"setup_fee" db:"setup_fee"`
	
	// Billing
	BillingUsage   int           `json:"billing_usage" db:"billing_usage"`
	BillingCycle   RenewalPeriod `json:"billing_cycle" db:"billing_cycle"`
	OverageCharge  sql.NullFloat64 `json:"overage_charge,omitempty" db:"overage_charge"`
	
	// Features
	MaxOffers    sql.NullInt32          `json:"max_offers,omitempty" db:"max_offers"`
	MaxCustomers sql.NullInt32          `json:"max_customers,omitempty" db:"max_customers"`
	Features     map[string]interface{} `json:"features,omitempty" db:"features"`
	
	// Status
	Status   SubscriptionStatus `json:"status" db:"status"`
	IsPublic bool               `json:"is_public" db:"is_public"`
	
	// Metadata
	Metadata  map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	
	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type SubscriptionPlanStats struct {
	TotalPlans          int64   `json:"total_plans"`
	ActivePlans         int64   `json:"active_plans"`
	InactivePlans       int64   `json:"inactive_plans"`
	TotalSubscribers    int64   `json:"total_subscribers"`
	AveragePrice        float64 `json:"average_price"`
	MostPopularPlanID   int64   `json:"most_popular_plan_id,omitempty"`
	MostPopularPlanName string  `json:"most_popular_plan_name,omitempty"`
}