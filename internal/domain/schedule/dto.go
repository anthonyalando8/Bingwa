// internal/domain/schedule/dto.go
package schedule

import "time"

type CreateScheduledOfferRequest struct {
	OfferID       int64  `json:"offer_id" binding:"required"`
	CustomerPhone string `json:"customer_phone" binding:"required"`
	
	// Schedule details
	ScheduledTime time.Time `json:"scheduled_time" binding:"required"`
	
	// Auto-renewal
	AutoRenew     bool          `json:"auto_renew"`
	RenewalPeriod RenewalPeriod `json:"renewal_period"`
	RenewalLimit  *int32        `json:"renewal_limit"`
	RenewUntil    *time.Time    `json:"renew_until"`
	
	// Metadata
	Metadata map[string]interface{} `json:"metadata"`
}

type UpdateScheduledOfferRequest struct {
	ScheduledTime *time.Time `json:"scheduled_time"`
	AutoRenew     *bool      `json:"auto_renew"`
	RenewalPeriod *RenewalPeriod `json:"renewal_period"`
	RenewalLimit  *int32     `json:"renewal_limit"`
	RenewUntil    *time.Time `json:"renew_until"`
	Metadata      map[string]interface{} `json:"metadata"`
}

type ScheduledOfferListFilters struct {
	Status        *ScheduleStatus `form:"status"`
	OfferID       *int64          `form:"offer_id"`
	CustomerPhone string          `form:"customer_phone"`
	AutoRenew     *bool           `form:"auto_renew"`
	DueToday      bool            `form:"due_today"`
	Page          int             `form:"page" binding:"min=1"`
	PageSize      int             `form:"page_size" binding:"min=1,max=100"`
	SortBy        string          `form:"sort_by"`
	SortOrder     string          `form:"sort_order" binding:"omitempty,oneof=asc desc"`
}

type ScheduledOfferListResponse struct {
	Schedules  []ScheduledOffer `json:"schedules"`
	Total      int64            `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
}

type ExecuteScheduledOfferInput struct {
	USSDResponse       string `json:"ussd_response"`
	USSDSessionID      string `json:"ussd_session_id"`
	USSDProcessingTime int32  `json:"ussd_processing_time"`
	Status             string `json:"status"` // success, failed
	FailureReason      string `json:"failure_reason"`
}

type ScheduleHistoryListFilters struct {
	ScheduledOfferID *int64  `form:"scheduled_offer_id"`
	Status           *string `form:"status"`
	Page             int     `form:"page" binding:"min=1"`
	PageSize         int     `form:"page_size" binding:"min=1,max=100"`
}

type ScheduleHistoryListResponse struct {
	History    []ScheduledOfferHistory `json:"history"`
	Total      int64                   `json:"total"`
	Page       int                     `json:"page"`
	PageSize   int                     `json:"page_size"`
	TotalPages int                     `json:"total_pages"`
}