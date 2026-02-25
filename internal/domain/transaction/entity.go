// internal/domain/transaction/entity.go
package transaction

import (
	"database/sql"
	"time"
)

type PaymentMethod string

const (
	PaymentMethodMpesa       PaymentMethod = "mpesa"
	PaymentMethodAirtelMoney PaymentMethod = "airtel_money"
	PaymentMethodTigopesa    PaymentMethod = "tigopesa"
	PaymentMethodCard        PaymentMethod = "card"
	PaymentMethodBank        PaymentMethod = "bank"
	PaymentMethodAgentBalance PaymentMethod = "agent_balance"
)

type TransactionStatus string

const (
	TransactionStatusPending    TransactionStatus = "pending"
	TransactionStatusProcessing TransactionStatus = "processing"
	TransactionStatusSuccess    TransactionStatus = "success"
	TransactionStatusFailed     TransactionStatus = "failed"
	TransactionStatusCancelled  TransactionStatus = "cancelled"
	TransactionStatusReversed   TransactionStatus = "reversed"
)

type OfferRequest struct {
	ID                 int64             `json:"id" db:"id"`
	RequestReference   string            `json:"request_reference" db:"request_reference"`
	
	// Offer and customer info
	OfferID           int64             `json:"offer_id" db:"offer_id"`
	AgentIdentityID   int64             `json:"agent_identity_id" db:"agent_identity_id"`
	CustomerID        sql.NullInt64     `json:"customer_id,omitempty" db:"customer_id"`
	CustomerPhone     string            `json:"customer_phone" db:"customer_phone"`
	CustomerName      sql.NullString    `json:"customer_name,omitempty" db:"customer_name"`
	
	// Payment details
	PaymentMethod     PaymentMethod     `json:"payment_method" db:"payment_method"`
	AmountPaid        float64           `json:"amount_paid" db:"amount_paid"`
	Currency          string            `json:"currency" db:"currency"`
	
	// M-Pesa specific
	MpesaTransactionID   sql.NullString `json:"mpesa_transaction_id,omitempty" db:"mpesa_transaction_id"`
	MpesaReceiptNumber   sql.NullString `json:"mpesa_receipt_number,omitempty" db:"mpesa_receipt_number"`
	MpesaTransactionDate sql.NullTime   `json:"mpesa_transaction_date,omitempty" db:"mpesa_transaction_date"`
	MpesaPhoneNumber     sql.NullString `json:"mpesa_phone_number,omitempty" db:"mpesa_phone_number"`
	MpesaMessage         sql.NullString `json:"mpesa_message,omitempty" db:"mpesa_message"`
	
	// Request details
	RequestTime   time.Time         `json:"request_time" db:"request_time"`
	ProcessedAt   sql.NullTime      `json:"processed_at,omitempty" db:"processed_at"`
	Status        TransactionStatus `json:"status" db:"status"`
	FailureReason sql.NullString    `json:"failure_reason,omitempty" db:"failure_reason"`
	RetryCount    int               `json:"retry_count" db:"retry_count"`
	
	// Metadata
	DeviceInfo map[string]interface{} `json:"device_info,omitempty" db:"device_info"`
	Metadata   map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	
	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type OfferRedemption struct {
	ID                  int64             `json:"id" db:"id"`
	RedemptionReference string            `json:"redemption_reference" db:"redemption_reference"`
	
	// Related entities
	OfferID           int64         `json:"offer_id" db:"offer_id"`
	OfferRequestID    int64         `json:"offer_request_id" db:"offer_request_id"`
	AgentIdentityID   int64         `json:"agent_identity_id" db:"agent_identity_id"`
	CustomerID        sql.NullInt64 `json:"customer_id,omitempty" db:"customer_id"`
	CustomerPhone     string        `json:"customer_phone" db:"customer_phone"`
	
	// Redemption details
	Amount       float64 `json:"amount" db:"amount"`
	Currency     string  `json:"currency" db:"currency"`
	USSDCodeUsed string  `json:"ussd_code_used" db:"ussd_code_used"`
	
	// USSD Response
	USSDResponse       sql.NullString `json:"ussd_response,omitempty" db:"ussd_response"`
	USSDSessionID      sql.NullString `json:"ussd_session_id,omitempty" db:"ussd_session_id"`
	USSDProcessingTime sql.NullInt32  `json:"ussd_processing_time,omitempty" db:"ussd_processing_time"`
	
	// Status
	RedemptionTime time.Time         `json:"redemption_time" db:"redemption_time"`
	CompletedAt    sql.NullTime      `json:"completed_at,omitempty" db:"completed_at"`
	Status         TransactionStatus `json:"status" db:"status"`
	FailureReason  sql.NullString    `json:"failure_reason,omitempty" db:"failure_reason"`
	RetryCount     int               `json:"retry_count" db:"retry_count"`
	MaxRetries     int               `json:"max_retries" db:"max_retries"`
	
	// Validity
	ValidFrom  sql.NullTime `json:"valid_from,omitempty" db:"valid_from"`
	ValidUntil sql.NullTime `json:"valid_until,omitempty" db:"valid_until"`
	
	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	
	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type TransactionStats struct {
	TotalRequests       int64   `json:"total_requests"`
	SuccessfulRequests  int64   `json:"successful_requests"`
	PendingRequests     int64   `json:"pending_requests"`
	FailedRequests      int64   `json:"failed_requests"`
	TotalRevenue        float64 `json:"total_revenue"`
	SuccessRate         float64 `json:"success_rate"`
}