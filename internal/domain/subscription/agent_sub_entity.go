// internal/domain/subscription/entity.go
package subscription

import (
	"database/sql"
	"time"
)

const (
	SubscriptionStatusActive    SubscriptionStatus = "active"
	SubscriptionStatusInactive  SubscriptionStatus = "inactive"
	SubscriptionStatusExpired   SubscriptionStatus = "expired"
	SubscriptionStatusCancelled SubscriptionStatus = "cancelled"
	SubscriptionStatusSuspended SubscriptionStatus = "suspended"
	SubscriptionStatusPending   SubscriptionStatus = "pending"
	SubscriptionStatusTrialing  SubscriptionStatus = "trialing"
)

type AgentSubscription struct {
	ID                     int64              `json:"id" db:"id"`
	SubscriptionReference  string             `json:"subscription_reference" db:"subscription_reference"`
	
	// Related entities
	AgentIdentityID        int64              `json:"agent_identity_id" db:"agent_identity_id"`
	SubscriptionPlanID     int64              `json:"subscription_plan_id" db:"subscription_plan_id"`
	PromotionalCampaignID  sql.NullInt64      `json:"promotional_campaign_id,omitempty" db:"promotional_campaign_id"`
	
	// Subscription period
	StartDate              time.Time          `json:"start_date" db:"start_date"`
	EndDate                sql.NullTime       `json:"end_date,omitempty" db:"end_date"`
	CurrentPeriodStart     time.Time          `json:"current_period_start" db:"current_period_start"`
	CurrentPeriodEnd       time.Time          `json:"current_period_end" db:"current_period_end"`
	
	// Renewal
	AutoRenew              bool               `json:"auto_renew" db:"auto_renew"`
	RenewalCount           int                `json:"renewal_count" db:"renewal_count"`
	NextBillingDate        sql.NullTime       `json:"next_billing_date,omitempty" db:"next_billing_date"`
	
	// Usage tracking
	RequestsUsed           int                `json:"requests_used" db:"requests_used"`
	RequestsLimit          sql.NullInt32      `json:"requests_limit,omitempty" db:"requests_limit"`
	
	// Pricing
	PlanPrice              float64            `json:"plan_price" db:"plan_price"`
	DiscountApplied        float64            `json:"discount_applied" db:"discount_applied"`
	AmountPaid             float64            `json:"amount_paid" db:"amount_paid"`
	Currency               string             `json:"currency" db:"currency"`
	
	// Status
	Status                 SubscriptionStatus `json:"status" db:"status"`
	CancelledAt            sql.NullTime       `json:"cancelled_at,omitempty" db:"cancelled_at"`
	CancellationReason     sql.NullString     `json:"cancellation_reason,omitempty" db:"cancellation_reason"`
	
	// Metadata
	Metadata               map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	
	// Timestamps
	CreatedAt              time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time          `json:"updated_at" db:"updated_at"`
}

type SubscriptionStats struct {
	TotalSubscriptions     int64   `json:"total_subscriptions"`
	ActiveSubscriptions    int64   `json:"active_subscriptions"`
	ExpiredSubscriptions   int64   `json:"expired_subscriptions"`
	CancelledSubscriptions int64   `json:"cancelled_subscriptions"`
	TotalRevenue           float64 `json:"total_revenue"`
	AverageSubscriptionValue float64 `json:"average_subscription_value"`
}