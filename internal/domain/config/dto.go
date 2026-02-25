// internal/domain/config/dto.go
package config

type CreateConfigRequest struct {
	ConfigKey   string                 `json:"config_key" binding:"required,max=255"`
	ConfigValue map[string]interface{} `json:"config_value" binding:"required"`
	Description string                 `json:"description"`
	DeviceID    string                 `json:"device_id"`
	IsGlobal    bool                   `json:"is_global"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type UpdateConfigRequest struct {
	ConfigValue map[string]interface{} `json:"config_value" binding:"required"`
	Description *string                `json:"description"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type ConfigListFilters struct {
	IsGlobal *bool  `form:"is_global"`
	DeviceID string `form:"device_id"`
	Search   string `form:"search"` // Search by key or description
	Page     int    `form:"page" binding:"min=1"`
	PageSize int    `form:"page_size" binding:"min=1,max=100"`
}

type ConfigListResponse struct {
	Configs    []AgentConfig `json:"configs"`
	Total      int64         `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
	TotalPages int           `json:"total_pages"`
}

// Specific config value structures
type NotificationConfig struct {
	Enabled         bool   `json:"enabled"`
	Sound           bool   `json:"sound"`
	Vibration       bool   `json:"vibration"`
	EmailAlerts     bool   `json:"email_alerts"`
	PushEnabled     bool   `json:"push_enabled"`
}

type USSDConfig struct {
	AutoRetry       bool `json:"auto_retry"`
	RetryAttempts   int  `json:"retry_attempts"`
	TimeoutSeconds  int  `json:"timeout_seconds"`
	AutoDismissDialog bool `json:"auto_dismiss_dialog"`
}

type AndroidDeviceConfig struct {
	Enabled         bool   `json:"enabled"`
	APIURL          string `json:"api_url"`
	APIKey          string `json:"api_key"`
	MaxConcurrent   int    `json:"max_concurrent_requests"`
	HealthCheckInterval int `json:"health_check_interval_seconds"`
}

type BusinessConfig struct {
	AutoRenewalEnabled       bool `json:"auto_renewal_enabled"`
	DefaultOfferValidityDays int  `json:"default_offer_validity_days"`
	MaxOffersPerCustomer     int  `json:"max_offers_per_customer"`
	RequireCustomerVerification bool `json:"require_customer_verification"`
}

type DisplayConfig struct {
	Theme      string `json:"theme"` // light, dark
	Language   string `json:"language"` // en, sw
	Timezone   string `json:"timezone"`
	DateFormat string `json:"date_format"`
	Currency   string `json:"currency"`
}

type SecurityConfig struct {
	TwoFactorEnabled    bool     `json:"2fa_enabled"`
	SessionTimeoutMinutes int    `json:"session_timeout_minutes"`
	IPWhitelist         []string `json:"ip_whitelist"`
	AllowedDevices      int      `json:"allowed_devices"`
}