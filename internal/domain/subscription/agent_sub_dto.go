// internal/domain/subscription/dto.go
package subscription

type CreateSubscriptionRequest struct {
	SubscriptionPlanID    int64                  `json:"subscription_plan_id" binding:"required"`
	PromotionalCode       string                 `json:"promotional_code"`
	AutoRenew             bool                   `json:"auto_renew"`
	
	// Payment details (from mobile)
	AmountPaid            float64                `json:"amount_paid" binding:"required,min=0"`
	Currency              string                 `json:"currency" binding:"required,len=3"`
	PaymentReference      string                 `json:"payment_reference"` // M-Pesa transaction ID
	PaymentMethod         string                 `json:"payment_method"`
	
	// Metadata
	Metadata              map[string]interface{} `json:"metadata"`
}

type UpdateSubscriptionRequest struct {
	AutoRenew             *bool                  `json:"auto_renew"`
	Metadata              map[string]interface{} `json:"metadata"`
}

type SubscriptionListFilters struct {
	Status                *SubscriptionStatus `form:"status"`
	PlanID                *int64              `form:"plan_id"`
	IsExpiring            bool                `form:"is_expiring"` // Expiring in next 7 days
	HasCampaign           *bool               `form:"has_campaign"`
	Page                  int                 `form:"page" binding:"min=1"`
	PageSize              int                 `form:"page_size" binding:"min=1,max=100"`
	SortBy                string              `form:"sort_by"`
	SortOrder             string              `form:"sort_order" binding:"omitempty,oneof=asc desc"`
}

type SubscriptionListResponse struct {
	Subscriptions []AgentSubscription `json:"subscriptions"`
	Total         int64               `json:"total"`
	Page          int                 `json:"page"`
	PageSize      int                 `json:"page_size"`
	TotalPages    int                 `json:"total_pages"`
}

type RenewSubscriptionRequest struct {
	AmountPaid            float64                `json:"amount_paid" binding:"required,min=0"`
	Currency              string                 `json:"currency" binding:"required,len=3"`
	PaymentReference      string                 `json:"payment_reference"`
	PaymentMethod         string                 `json:"payment_method"`
	PromotionalCode       string                 `json:"promotional_code"`
}

type CancelSubscriptionRequest struct {
	Reason                string `json:"reason"`
	CancelImmediately     bool   `json:"cancel_immediately"` // If false, cancel at period end
}

type SubscriptionUsageInfo struct {
	SubscriptionID        int64   `json:"subscription_id"`
	RequestsUsed          int     `json:"requests_used"`
	RequestsLimit         int     `json:"requests_limit"`
	RequestsRemaining     int     `json:"requests_remaining"`
	UsagePercentage       float64 `json:"usage_percentage"`
	DaysRemaining         int     `json:"days_remaining"`
	IsExpiring            bool    `json:"is_expiring"`
	CanMakeRequests       bool    `json:"can_make_requests"`
	Metadata              map[string]interface{} `json:"metadata"`
}