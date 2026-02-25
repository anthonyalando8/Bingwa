// internal/domain/campaign/dto.go
package campaign

import "time"

type CreateCampaignRequest struct {
	Name            string       `json:"name" binding:"required,max=255"`
	Description     string       `json:"description"`
	PromotionalCode string       `json:"promotional_code" binding:"required,max=50"`
	
	// Discount
	DiscountType      DiscountType `json:"discount_type" binding:"required"`
	DiscountValue     float64      `json:"discount_value" binding:"required,min=0"`
	MaxDiscountAmount *float64     `json:"max_discount_amount" binding:"omitempty,min=0"`
	
	// Validity
	StartDate time.Time `json:"start_date" binding:"required"`
	EndDate   time.Time `json:"end_date" binding:"required"`
	
	// Usage limits
	MaxUses     *int32 `json:"max_uses" binding:"omitempty,min=1"`
	UsesPerUser int    `json:"uses_per_user" binding:"min=1"`
	
	// Targeting
	ApplicablePlans []int64  `json:"applicable_plans"`
	TargetUserTypes []string `json:"target_user_types"`
	
	// Metadata
	Metadata map[string]interface{} `json:"metadata"`
}

type UpdateCampaignRequest struct {
	Name            *string       `json:"name" binding:"omitempty,max=255"`
	Description     *string       `json:"description"`
	
	// Discount
	DiscountValue     *float64 `json:"discount_value" binding:"omitempty,min=0"`
	MaxDiscountAmount *float64 `json:"max_discount_amount" binding:"omitempty,min=0"`
	
	// Validity
	StartDate *time.Time `json:"start_date"`
	EndDate   *time.Time `json:"end_date"`
	
	// Usage limits
	MaxUses     *int32 `json:"max_uses" binding:"omitempty,min=1"`
	UsesPerUser *int   `json:"uses_per_user" binding:"omitempty,min=1"`
	
	// Targeting
	ApplicablePlans []int64  `json:"applicable_plans"`
	TargetUserTypes []string `json:"target_user_types"`
	
	// Metadata
	Metadata map[string]interface{} `json:"metadata"`
}

type CampaignListFilters struct {
	Status          *CampaignStatus `form:"status"`
	DiscountType    *DiscountType   `form:"discount_type"`
	IsActive        *bool           `form:"is_active"` // Currently active based on dates
	Search          string          `form:"search"`
	Page            int             `form:"page" binding:"min=1"`
	PageSize        int             `form:"page_size" binding:"min=1,max=100"`
	SortBy          string          `form:"sort_by"` // created_at, start_date, end_date
	SortOrder       string          `form:"sort_order" binding:"omitempty,oneof=asc desc"`
}

type CampaignListResponse struct {
	Campaigns  []PromotionalCampaign `json:"campaigns"`
	Total      int64                 `json:"total"`
	Page       int                   `json:"page"`
	PageSize   int                   `json:"page_size"`
	TotalPages int                   `json:"total_pages"`
}

type ValidateCampaignRequest struct {
	PromotionalCode string `json:"promotional_code" binding:"required"`
	PlanID          int64  `json:"plan_id" binding:"required"`
	UserType        string `json:"user_type"` // new_user, existing_user
}

type ValidateCampaignResponse struct {
	Valid             bool              `json:"valid"`
	Campaign          *PromotionalCampaign `json:"campaign,omitempty"`
	DiscountAmount    float64           `json:"discount_amount"`
	FinalPrice        float64           `json:"final_price"`
	Message           string            `json:"message"`
}