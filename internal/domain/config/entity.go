// internal/domain/config/entity.go
package config

import (
	"database/sql"
	"time"
)

type AgentConfig struct {
	ID              int64                  `json:"id" db:"id"`
	AgentIdentityID int64                  `json:"agent_identity_id" db:"agent_identity_id"`
	ConfigKey       string                 `json:"config_key" db:"config_key"`
	ConfigValue     map[string]interface{} `json:"config_value" db:"config_value"`
	Description     sql.NullString         `json:"description,omitempty" db:"description"`
	DeviceID        sql.NullString         `json:"device_id,omitempty" db:"device_id"`
	IsGlobal        bool                   `json:"is_global" db:"is_global"`
	Metadata        map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at" db:"updated_at"`
}

// Predefined config keys
const (
	// Notification settings
	ConfigKeyNotifications           = "notifications"
	ConfigKeyNotificationSound       = "notification_sound"
	ConfigKeyNotificationVibration   = "notification_vibration"
	
	// USSD settings
	ConfigKeyUSSDAutoRetry           = "ussd_auto_retry"
	ConfigKeyUSSDRetryAttempts       = "ussd_retry_attempts"
	ConfigKeyUSSDTimeout             = "ussd_timeout"
	
	// Android device settings
	ConfigKeyAndroidDeviceEnabled    = "android_device_enabled"
	ConfigKeyAndroidDeviceAPIURL     = "android_device_api_url"
	ConfigKeyAndroidDeviceAPIKey     = "android_device_api_key"
	
	// Business settings
	ConfigKeyAutoRenewalEnabled      = "auto_renewal_enabled"
	ConfigKeyDefaultOfferValidity    = "default_offer_validity"
	ConfigKeyMaxOffersPerCustomer    = "max_offers_per_customer"
	
	// Display settings
	ConfigKeyTheme                   = "theme"
	ConfigKeyLanguage                = "language"
	ConfigKeyTimezone                = "timezone"
	ConfigKeyDateFormat              = "date_format"
	
	// Security settings
	ConfigKey2FAEnabled              = "2fa_enabled"
	ConfigKeySessionTimeout          = "session_timeout"
	ConfigKeyIPWhitelist             = "ip_whitelist"
)