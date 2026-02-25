// internal/domain/offer/dto.go
package offer

import "time"

type CreateOfferRequest struct {
	Name        string     `json:"name" binding:"required,max=255"`
	Description string     `json:"description"`
	Type        OfferType  `json:"type" binding:"required"`
	Amount      float64    `json:"amount" binding:"required,min=0"`
	Units       OfferUnits `json:"units" binding:"required"`

	// Pricing
	Price              float64 `json:"price" binding:"required,min=0"`
	Currency           string  `json:"currency" binding:"required,len=3"`
	DiscountPercentage float64 `json:"discount_percentage" binding:"min=0,max=100"`

	// Validity
	ValidityDays  int    `json:"validity_days" binding:"required,min=1"`
	ValidityLabel string `json:"validity_label"`

	// USSD Configuration
	USSDCodeTemplate     string             `json:"ussd_code_template" binding:"required"`
	USSDProcessingType   USSDProcessingType `json:"ussd_processing_type" binding:"required"`
	USSDExpectedResponse string             `json:"ussd_expected_response"`
	USSDErrorPattern     string             `json:"ussd_error_pattern"`

	// Features
	IsFeatured              bool  `json:"is_featured"`
	IsRecurring             bool  `json:"is_recurring"`
	MaxPurchasesPerCustomer *int32 `json:"max_purchases_per_customer"`

	// Availability
	AvailableFrom  *time.Time `json:"available_from"`
	AvailableUntil *time.Time `json:"available_until"`

	// Metadata
	Tags     []string               `json:"tags"`
	Metadata map[string]interface{} `json:"metadata"`
}

type UpdateOfferRequest struct {
	Name        *string    `json:"name" binding:"omitempty,max=255"`
	Description *string    `json:"description"`
	Type        *OfferType `json:"type"`
	Amount      *float64   `json:"amount" binding:"omitempty,min=0"`
	Units       *OfferUnits `json:"units"`

	// Pricing
	Price              *float64 `json:"price" binding:"omitempty,min=0"`
	DiscountPercentage *float64 `json:"discount_percentage" binding:"omitempty,min=0,max=100"`

	// Validity
	ValidityDays  *int    `json:"validity_days" binding:"omitempty,min=1"`
	ValidityLabel *string `json:"validity_label"`

	// USSD Configuration
	USSDCodeTemplate     *string             `json:"ussd_code_template"`
	USSDProcessingType   *USSDProcessingType `json:"ussd_processing_type"`
	USSDExpectedResponse *string             `json:"ussd_expected_response"`
	USSDErrorPattern     *string             `json:"ussd_error_pattern"`

	// Features
	IsFeatured              *bool  `json:"is_featured"`
	IsRecurring             *bool  `json:"is_recurring"`
	MaxPurchasesPerCustomer *int32 `json:"max_purchases_per_customer"`

	// Availability
	AvailableFrom  *time.Time `json:"available_from"`
	AvailableUntil *time.Time `json:"available_until"`

	// Metadata
	Tags     []string               `json:"tags"`
	Metadata map[string]interface{} `json:"metadata"`
}

type OfferListFilters struct {
	Type       *OfferType  `form:"type"`
	Status     *OfferStatus `form:"status"`
	IsFeatured *bool       `form:"is_featured"`
	MinPrice   *float64    `form:"min_price"`
	MaxPrice   *float64    `form:"max_price"`
	Search     string      `form:"search"`
	Tags       []string    `form:"tags"`
	Page       int         `form:"page" binding:"min=1"`
	PageSize   int         `form:"page_size" binding:"min=1,max=100"`
	SortBy     string      `form:"sort_by"` // price, created_at, name
	SortOrder  string      `form:"sort_order" binding:"omitempty,oneof=asc desc"`
}

type OfferListResponse struct {
	Offers     []AgentOffer `json:"offers"`
	Total      int64        `json:"total"`
	Page       int          `json:"page"`
	PageSize   int          `json:"page_size"`
	TotalPages int          `json:"total_pages"`
}

type AddUSSDCodeRequest struct {
	USSDCode         string             `json:"ussd_code" binding:"required"`
	SignaturePattern string             `json:"signature_pattern"`
	Priority         int                `json:"priority" binding:"min=1"`
	ExpectedResponse string             `json:"expected_response"`
	ErrorPattern     string             `json:"error_pattern"`
	ProcessingType   USSDProcessingType `json:"processing_type" binding:"required"`
	Metadata         map[string]interface{} `json:"metadata"`
}

type UpdateUSSDCodeRequest struct {
	USSDCode         *string             `json:"ussd_code"`
	SignaturePattern *string             `json:"signature_pattern"`
	Priority         *int                `json:"priority" binding:"omitempty,min=1"`
	IsActive         *bool               `json:"is_active"`
	ExpectedResponse *string             `json:"expected_response"`
	ErrorPattern     *string             `json:"error_pattern"`
	ProcessingType   *USSDProcessingType `json:"processing_type"`
	Metadata         map[string]interface{} `json:"metadata"`
}

type ReorderUSSDCodesRequest struct {
	Codes []struct {
		ID       int64 `json:"id" binding:"required"`
		Priority int   `json:"priority" binding:"required,min=1"`
	} `json:"codes" binding:"required,min=1"`
}

type RecordUSSDResultRequest struct {
	USSDCodeID int64  `json:"ussd_code_id" binding:"required"`
	Success    bool   `json:"success"`
	Response   string `json:"response"`
}