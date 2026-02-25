// internal/domain/transaction/dto.go
package transaction

import "time"

type CreateOfferRequestInput struct {
	OfferID       int64         `json:"offer_id" binding:"required"`
	CustomerPhone string        `json:"customer_phone" binding:"required"`
	CustomerName  string        `json:"customer_name"`
	PaymentMethod PaymentMethod `json:"payment_method" binding:"required"`
	AmountPaid    float64       `json:"amount_paid" binding:"required,min=0"`
	Currency      string        `json:"currency" binding:"required,len=3"`
	
	// M-Pesa specific
	MpesaTransactionID   string    `json:"mpesa_transaction_id"`
	MpesaReceiptNumber   string    `json:"mpesa_receipt_number"`
	MpesaTransactionDate time.Time `json:"mpesa_transaction_date"`
	MpesaPhoneNumber     string    `json:"mpesa_phone_number"`
	MpesaMessage         string    `json:"mpesa_message"`
	
	// Device info
	DeviceInfo map[string]interface{} `json:"device_info"`
	Metadata   map[string]interface{} `json:"metadata"`
}

type OfferRequestListFilters struct {
	Status        *TransactionStatus `form:"status"`
	OfferID       *int64             `form:"offer_id"`
	CustomerPhone string             `form:"customer_phone"`
	PaymentMethod *PaymentMethod     `form:"payment_method"`
	DateFrom      *time.Time         `form:"date_from"`
	DateTo        *time.Time         `form:"date_to"`
	Search        string             `form:"search"`
	Page          int                `form:"page" binding:"min=1"`
	PageSize      int                `form:"page_size" binding:"min=1,max=100"`
	SortBy        string             `form:"sort_by"`
	SortOrder     string             `form:"sort_order" binding:"omitempty,oneof=asc desc"`
}

type OfferRequestListResponse struct {
	Requests   []OfferRequest `json:"requests"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
}

type RedemptionListFilters struct {
	Status         *TransactionStatus `form:"status"`
	OfferID        *int64             `form:"offer_id"`
	OfferRequestID *int64             `form:"offer_request_id"`
	CustomerPhone  string             `form:"customer_phone"`
	DateFrom       *time.Time         `form:"date_from"`
	DateTo         *time.Time         `form:"date_to"`
	Page           int                `form:"page" binding:"min=1"`
	PageSize       int                `form:"page_size" binding:"min=1,max=100"`
	SortBy         string             `form:"sort_by"`
	SortOrder      string             `form:"sort_order" binding:"omitempty,oneof=asc desc"`
}

type RedemptionListResponse struct {
	Redemptions []OfferRedemption `json:"redemptions"`
	Total       int64             `json:"total"`
	Page        int               `json:"page"`
	PageSize    int               `json:"page_size"`
	TotalPages  int               `json:"total_pages"`
}

type UpdateUSSDResponseInput struct {
	USSDResponse       string `json:"ussd_response"`
	USSDSessionID      string `json:"ussd_session_id"`
	USSDProcessingTime int32  `json:"ussd_processing_time"`
	Status             TransactionStatus `json:"status"`
	FailureReason      string `json:"failure_reason"`
}