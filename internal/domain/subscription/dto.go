// internal/domain/subscription/dto.go
package subscription

type CreatePlanRequest struct {
	PlanCode    string                 `json:"plan_code" binding:"required,max=50"`
	Name        string                 `json:"name" binding:"required,max=255"`
	Description string                 `json:"description"`
	
	// Pricing
	Price       float64 `json:"price" binding:"required,min=0"`
	Currency    string  `json:"currency" binding:"required,len=3"`
	SetupFee    float64 `json:"setup_fee" binding:"min=0"`
	
	// Billing
	BillingUsage   int           `json:"billing_usage" binding:"required,min=1"`
	BillingCycle   RenewalPeriod `json:"billing_cycle" binding:"required"`
	OverageCharge  *float64      `json:"overage_charge" binding:"omitempty,min=0"`
	
	// Features
	MaxOffers    *int32                 `json:"max_offers" binding:"omitempty,min=1"`
	MaxCustomers *int32                 `json:"max_customers" binding:"omitempty,min=1"`
	Features     map[string]interface{} `json:"features"`
	
	// Status
	IsPublic bool `json:"is_public"`
	
	// Metadata
	Metadata map[string]interface{} `json:"metadata"`
}

type UpdatePlanRequest struct {
	Name        *string                `json:"name" binding:"omitempty,max=255"`
	Description *string                `json:"description"`
	
	// Pricing
	Price       *float64 `json:"price" binding:"omitempty,min=0"`
	SetupFee    *float64 `json:"setup_fee" binding:"omitempty,min=0"`
	
	// Billing
	BillingUsage   *int     `json:"billing_usage" binding:"omitempty,min=1"`
	BillingCycle   *RenewalPeriod `json:"billing_cycle"`
	OverageCharge  *float64 `json:"overage_charge" binding:"omitempty,min=0"`
	
	// Features
	MaxOffers    *int32                 `json:"max_offers" binding:"omitempty,min=1"`
	MaxCustomers *int32                 `json:"max_customers" binding:"omitempty,min=1"`
	Features     map[string]interface{} `json:"features"`
	
	// Status
	IsPublic *bool `json:"is_public"`
	
	// Metadata
	Metadata map[string]interface{} `json:"metadata"`
}

type PlanListFilters struct {
	Status     *SubscriptionStatus `form:"status"`
	IsPublic   *bool               `form:"is_public"`
	MinPrice   *float64            `form:"min_price"`
	MaxPrice   *float64            `form:"max_price"`
	BillingCycle *RenewalPeriod    `form:"billing_cycle"`
	Search     string              `form:"search"`
	Page       int                 `form:"page" binding:"min=1"`
	PageSize   int                 `form:"page_size" binding:"min=1,max=100"`
	SortBy     string              `form:"sort_by"` // price, name, created_at
	SortOrder  string              `form:"sort_order" binding:"omitempty,oneof=asc desc"`
}

type PlanListResponse struct {
	Plans      []SubscriptionPlan `json:"plans"`
	Total      int64              `json:"total"`
	Page       int                `json:"page"`
	PageSize   int                `json:"page_size"`
	TotalPages int                `json:"total_pages"`
}