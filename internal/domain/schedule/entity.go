// internal/domain/schedule/entity.go
package schedule

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

type ScheduleStatus string

const (
	ScheduleStatusActive    ScheduleStatus = "active"
	ScheduleStatusInactive  ScheduleStatus = "inactive"
	ScheduleStatusPaused    ScheduleStatus = "paused"
	ScheduleStatusCancelled ScheduleStatus = "cancelled"
	ScheduleStatusExpired   ScheduleStatus = "expired"
	ScheduleStatusCompleted ScheduleStatus = "completed"
)

type ScheduledOffer struct {
	ID                int64          `json:"id" db:"id"`
	ScheduleReference string         `json:"schedule_reference" db:"schedule_reference"`
	
	// Related entities
	OfferID         int64         `json:"offer_id" db:"offer_id"`
	AgentIdentityID int64         `json:"agent_identity_id" db:"agent_identity_id"`
	CustomerID      sql.NullInt64 `json:"customer_id,omitempty" db:"customer_id"`
	CustomerPhone   string        `json:"customer_phone" db:"customer_phone"`
	
	// Schedule details
	ScheduledTime    time.Time    `json:"scheduled_time" db:"scheduled_time"`
	NextRenewalDate  sql.NullTime `json:"next_renewal_date,omitempty" db:"next_renewal_date"`
	LastRenewalDate  sql.NullTime `json:"last_renewal_date,omitempty" db:"last_renewal_date"`
	
	// Auto-renewal configuration
	AutoRenew      bool           `json:"auto_renew" db:"auto_renew"`
	RenewalPeriod  sql.NullString `json:"renewal_period,omitempty" db:"renewal_period"`
	RenewalCount   int            `json:"renewal_count" db:"renewal_count"`
	RenewalLimit   sql.NullInt32  `json:"renewal_limit,omitempty" db:"renewal_limit"`
	RenewUntil     sql.NullTime   `json:"renew_until,omitempty" db:"renew_until"`
	
	// Status
	Status              ScheduleStatus `json:"status" db:"status"`
	PausedAt            sql.NullTime   `json:"paused_at,omitempty" db:"paused_at"`
	CancelledAt         sql.NullTime   `json:"cancelled_at,omitempty" db:"cancelled_at"`
	CancellationReason  sql.NullString `json:"cancellation_reason,omitempty" db:"cancellation_reason"`
	
	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	
	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type ScheduledOfferHistory struct {
	ID                 int64          `json:"id" db:"id"`
	ScheduledOfferID   int64          `json:"scheduled_offer_id" db:"scheduled_offer_id"`
	OfferRedemptionID  sql.NullInt64  `json:"offer_redemption_id,omitempty" db:"offer_redemption_id"`
	CustomerID         sql.NullInt64  `json:"customer_id,omitempty" db:"customer_id"`
	CustomerPhone      string         `json:"customer_phone" db:"customer_phone"`
	
	// Renewal details
	RenewalTime   time.Time `json:"renewal_time" db:"renewal_time"`
	RenewalNumber int       `json:"renewal_number" db:"renewal_number"`
	Status        string    `json:"status" db:"status"` // transaction_status
	FailureReason sql.NullString `json:"failure_reason,omitempty" db:"failure_reason"`
	
	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	
	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type ScheduleStats struct {
	TotalSchedules    int64 `json:"total_schedules"`
	ActiveSchedules   int64 `json:"active_schedules"`
	PausedSchedules   int64 `json:"paused_schedules"`
	TotalRenewals     int64 `json:"total_renewals"`
	SuccessfulRenewals int64 `json:"successful_renewals"`
	FailedRenewals    int64 `json:"failed_renewals"`
}