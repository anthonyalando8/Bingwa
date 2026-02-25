// internal/domain/customer/dto.go
package customer

type CreateCustomerRequest struct {
	FullName       string                 `json:"full_name" binding:"max=255"`
	PhoneNumber    string                 `json:"phone_number" binding:"required,max=20"`
	AltPhoneNumber string                 `json:"alt_phone_number" binding:"max=20"`
	Email          string                 `json:"email" binding:"omitempty,email,max=255"`
	Notes          string                 `json:"notes"`
	Tags           []string               `json:"tags"`
	Metadata       map[string]interface{} `json:"metadata"`
}

type UpdateCustomerRequest struct {
	FullName       *string                `json:"full_name" binding:"omitempty,max=255"`
	PhoneNumber    *string                `json:"phone_number" binding:"omitempty,max=20"`
	AltPhoneNumber *string                `json:"alt_phone_number" binding:"omitempty,max=20"`
	Email          *string                `json:"email" binding:"omitempty,email,max=255"`
	Notes          *string                `json:"notes"`
	Tags           []string               `json:"tags"`
	Metadata       map[string]interface{} `json:"metadata"`
}

type CustomerListFilters struct {
	IsActive   *bool  `form:"is_active"`
	IsVerified *bool  `form:"is_verified"`
	Search     string `form:"search"` // Search by name, phone, email
	Tags       []string `form:"tags"`
	Page       int    `form:"page" binding:"min=1"`
	PageSize   int    `form:"page_size" binding:"min=1,max=100"`
	SortBy     string `form:"sort_by"` // created_at, full_name, phone_number
	SortOrder  string `form:"sort_order" binding:"omitempty,oneof=asc desc"`
}

type CustomerListResponse struct {
	Customers  []AgentCustomer `json:"customers"`
	Total      int64           `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	TotalPages int             `json:"total_pages"`
}