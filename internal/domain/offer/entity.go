// internal/domain/offer/entity.go
package offer

import (
	"database/sql"
	"time"
)

type OfferType string

const (
	OfferTypeData  OfferType = "data"
	OfferTypeSMS   OfferType = "sms"
	OfferTypeVoice OfferType = "voice"
	OfferTypeCombo OfferType = "combo"
)

type OfferUnits string

const (
	UnitsGB      OfferUnits = "GB"
	UnitsMB      OfferUnits = "MB"
	UnitsKB      OfferUnits = "KB"
	UnitsMinutes OfferUnits = "minutes"
	UnitsSMS     OfferUnits = "sms"
	UnitsUnits   OfferUnits = "units"
)

type OfferStatus string

const (
	OfferStatusActive    OfferStatus = "active"
	OfferStatusInactive  OfferStatus = "inactive"
	OfferStatusPaused    OfferStatus = "paused"
	OfferStatusSuspended OfferStatus = "suspended"
	OfferStatusArchived  OfferStatus = "archived"
)

type USSDProcessingType string

const (
	USSDProcessingExpress   USSDProcessingType = "express"
	USSDProcessingMultistep USSDProcessingType = "multistep"
	USSDProcessingCallback  USSDProcessingType = "callback"
)

type AgentOffer struct {
	ID              int64  `json:"id" db:"id"`
	AgentIdentityID int64  `json:"agent_identity_id" db:"agent_identity_id"`
	OfferCode       string `json:"offer_code" db:"offer_code"`

	// Offer details
	Name        string         `json:"name" db:"name"`
	Description sql.NullString `json:"description,omitempty" db:"description"`
	Type        OfferType      `json:"type" db:"type"`
	Amount      float64        `json:"amount" db:"amount"`
	Units       OfferUnits     `json:"units" db:"units"`

	// Pricing
	Price              float64 `json:"price" db:"price"`
	Currency           string  `json:"currency" db:"currency"`
	DiscountPercentage float64 `json:"discount_percentage" db:"discount_percentage"`

	// Validity
	ValidityDays  int            `json:"validity_days" db:"validity_days"`
	ValidityLabel sql.NullString `json:"validity_label,omitempty" db:"validity_label"`

	// USSD Configuration
	USSDCodeTemplate     string             `json:"ussd_code_template" db:"ussd_code_template"`
	USSDProcessingType   USSDProcessingType `json:"ussd_processing_type" db:"ussd_processing_type"`
	USSDExpectedResponse sql.NullString     `json:"ussd_expected_response,omitempty" db:"ussd_expected_response"`
	USSDErrorPattern     sql.NullString     `json:"ussd_error_pattern,omitempty" db:"ussd_error_pattern"`

	// Features
	IsFeatured              bool          `json:"is_featured" db:"is_featured"`
	IsRecurring             bool          `json:"is_recurring" db:"is_recurring"`
	MaxPurchasesPerCustomer sql.NullInt32 `json:"max_purchases_per_customer,omitempty" db:"max_purchases_per_customer"`

	// Status
	Status         OfferStatus  `json:"status" db:"status"`
	AvailableFrom  sql.NullTime `json:"available_from,omitempty" db:"available_from"`
	AvailableUntil sql.NullTime `json:"available_until,omitempty" db:"available_until"`

	// Metadata
	Tags     []string        `json:"tags,omitempty" db:"tags"`
	Metadata map[string]interface{} `json:"metadata,omitempty" db:"metadata"`

	// Timestamps
	CreatedAt time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt time.Time    `json:"updated_at" db:"updated_at"`
	DeletedAt sql.NullTime `json:"deleted_at,omitempty" db:"deleted_at"`

	// Primary USSD Code (loaded separately)
	PrimaryUSSDCode *OfferUSSDCode `json:"primary_ussd_code,omitempty" db:"-"`
}

type OfferStats struct {
	TotalOffers       int64   `json:"total_offers"`
	ActiveOffers      int64   `json:"active_offers"`
	FeaturedOffers    int64   `json:"featured_offers"`
	TotalRevenue      float64 `json:"total_revenue"`
	AveragePrice      float64 `json:"average_price"`
	MostPopularOfferID int64  `json:"most_popular_offer_id,omitempty"`
	MostPopularOfferName string `json:"most_popular_offer_name,omitempty"`
}

type OfferUSSDCode struct {
	ID               int64                  `json:"id" db:"id"`
	OfferID          int64                  `json:"offer_id" db:"offer_id"`
	
	// USSD code details
	USSDCode         string                 `json:"ussd_code" db:"ussd_code"`
	SignaturePattern sql.NullString         `json:"signature_pattern,omitempty" db:"signature_pattern"`
	Priority         int                    `json:"priority" db:"priority"`
	IsActive         bool                   `json:"is_active" db:"is_active"`
	
	// Processing details
	ExpectedResponse sql.NullString         `json:"expected_response,omitempty" db:"expected_response"`
	ErrorPattern     sql.NullString         `json:"error_pattern,omitempty" db:"error_pattern"`
	ProcessingType   USSDProcessingType     `json:"processing_type" db:"processing_type"`
	
	// Usage statistics
	SuccessCount     int                    `json:"success_count" db:"success_count"`
	FailureCount     int                    `json:"failure_count" db:"failure_count"`
	LastUsedAt       sql.NullTime           `json:"last_used_at,omitempty" db:"last_used_at"`
	LastSuccessAt    sql.NullTime           `json:"last_success_at,omitempty" db:"last_success_at"`
	LastFailureAt    sql.NullTime           `json:"last_failure_at,omitempty" db:"last_failure_at"`
	
	// Metadata
	Metadata         map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	
	// Timestamps
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at" db:"updated_at"`
}

type USSDCodeStats struct {
	TotalCodes    int     `json:"total_codes"`
	ActiveCodes   int     `json:"active_codes"`
	SuccessRate   float64 `json:"success_rate"`
	TotalAttempts int     `json:"total_attempts"`
}