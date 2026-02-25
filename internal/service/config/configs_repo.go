// internal/usecase/config/config_service.go
package config

import (
	"context"
	"database/sql"
	"fmt"

	"bingwa-service/internal/domain/config"
	xerrors "bingwa-service/internal/pkg/errors"
	"bingwa-service/internal/repository/postgres"

	"go.uber.org/zap"
)

type ConfigService struct {
	configRepo *postgres.AgentConfigRepository
	logger     *zap.Logger
}

func NewConfigService(configRepo *postgres.AgentConfigRepository, logger *zap.Logger) *ConfigService {
	return &ConfigService{
		configRepo: configRepo,
		logger:     logger,
	}
}

// CreateConfig creates a new configuration
func (s *ConfigService) CreateConfig(ctx context.Context, agentID int64, req *config.CreateConfigRequest) (*config.AgentConfig, error) {
	// Validate config key
	if err := s.validateConfigKey(req.ConfigKey); err != nil {
		return nil, err
	}

	// Validate config value based on key
	if err := s.validateConfigValue(req.ConfigKey, req.ConfigValue); err != nil {
		return nil, err
	}

	// Check if config already exists
	var deviceIDPtr *string
	if req.DeviceID != "" {
		deviceIDPtr = &req.DeviceID
	}

	exists, err := s.configRepo.ExistsByKey(ctx, agentID, req.ConfigKey, deviceIDPtr)
	if err != nil {
		return nil, fmt.Errorf("failed to check config existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("config with key '%s' already exists", req.ConfigKey)
	}

	// Create config entity
	cfg := &config.AgentConfig{
		AgentIdentityID: agentID,
		ConfigKey:       req.ConfigKey,
		ConfigValue:     req.ConfigValue,
		Description:     sql.NullString{String: req.Description, Valid: req.Description != ""},
		DeviceID:        sql.NullString{String: req.DeviceID, Valid: req.DeviceID != ""},
		IsGlobal:        req.IsGlobal,
		Metadata:        req.Metadata,
	}

	// If device_id is specified, it's not global
	if req.DeviceID != "" {
		cfg.IsGlobal = false
	}

	// Create in database
	if err := s.configRepo.Create(ctx, cfg); err != nil {
		s.logger.Error("failed to create config", zap.Error(err))
		return nil, fmt.Errorf("failed to create config: %w", err)
	}

	s.logger.Info("config created",
		zap.Int64("config_id", cfg.ID),
		zap.String("config_key", cfg.ConfigKey),
		zap.Int64("agent_id", agentID),
	)

	return cfg, nil
}

// GetConfig retrieves a configuration by ID
func (s *ConfigService) GetConfig(ctx context.Context, agentID, configID int64) (*config.AgentConfig, error) {
	cfg, err := s.configRepo.FindByID(ctx, configID)
	if err != nil {
		return nil, err
	}

	// Verify config belongs to agent
	if cfg.AgentIdentityID != agentID {
		return nil, xerrors.ErrUnauthorized
	}

	return cfg, nil
}

// GetConfigByKey retrieves a configuration by key
func (s *ConfigService) GetConfigByKey(ctx context.Context, agentID int64, configKey string, deviceID *string) (*config.AgentConfig, error) {
	cfg, err := s.configRepo.FindByKey(ctx, agentID, configKey, deviceID)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// ListConfigs retrieves configurations with filters
func (s *ConfigService) ListConfigs(ctx context.Context, agentID int64, filters *config.ConfigListFilters) (*config.ConfigListResponse, error) {
	// Set defaults
	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.PageSize < 1 {
		filters.PageSize = 20
	}
	if filters.PageSize > 100 {
		filters.PageSize = 100
	}

	configs, total, err := s.configRepo.List(ctx, agentID, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list configs: %w", err)
	}

	totalPages := int(total) / filters.PageSize
	if int(total)%filters.PageSize > 0 {
		totalPages++
	}

	return &config.ConfigListResponse{
		Configs:    configs,
		Total:      total,
		Page:       filters.Page,
		PageSize:   filters.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetAllConfigs retrieves all configurations for an agent
func (s *ConfigService) GetAllConfigs(ctx context.Context, agentID int64) ([]config.AgentConfig, error) {
	configs, err := s.configRepo.GetAllByAgent(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get all configs: %w", err)
	}

	return configs, nil
}

// GetGlobalConfigs retrieves all global configurations
func (s *ConfigService) GetGlobalConfigs(ctx context.Context, agentID int64) ([]config.AgentConfig, error) {
	configs, err := s.configRepo.GetGlobalConfigs(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get global configs: %w", err)
	}

	return configs, nil
}

// GetDeviceConfigs retrieves configurations for a specific device
func (s *ConfigService) GetDeviceConfigs(ctx context.Context, agentID int64, deviceID string) ([]config.AgentConfig, error) {
	configs, err := s.configRepo.GetDeviceConfigs(ctx, agentID, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device configs: %w", err)
	}

	return configs, nil
}

// UpdateConfig updates a configuration
func (s *ConfigService) UpdateConfig(ctx context.Context, agentID, configID int64, req *config.UpdateConfigRequest) (*config.AgentConfig, error) {
	// Get existing config
	cfg, err := s.configRepo.FindByID(ctx, configID)
	if err != nil {
		return nil, err
	}

	// Verify config belongs to agent
	if cfg.AgentIdentityID != agentID {
		return nil, xerrors.ErrUnauthorized
	}

	// Validate config value
	if err := s.validateConfigValue(cfg.ConfigKey, req.ConfigValue); err != nil {
		return nil, err
	}

	// Update fields
	cfg.ConfigValue = req.ConfigValue
	if req.Description != nil {
		cfg.Description = sql.NullString{String: *req.Description, Valid: *req.Description != ""}
	}
	if req.Metadata != nil {
		cfg.Metadata = req.Metadata
	}

	// Update in database
	if err := s.configRepo.Update(ctx, configID, cfg); err != nil {
		s.logger.Error("failed to update config", zap.Error(err))
		return nil, fmt.Errorf("failed to update config: %w", err)
	}

	s.logger.Info("config updated",
		zap.Int64("config_id", configID),
		zap.Int64("agent_id", agentID),
	)

	// Return updated config
	return s.configRepo.FindByID(ctx, configID)
}

// DeleteConfig deletes a configuration
func (s *ConfigService) DeleteConfig(ctx context.Context, agentID, configID int64) error {
	// Verify ownership
	cfg, err := s.configRepo.FindByID(ctx, configID)
	if err != nil {
		return err
	}
	if cfg.AgentIdentityID != agentID {
		return xerrors.ErrUnauthorized
	}

	if err := s.configRepo.Delete(ctx, configID); err != nil {
		s.logger.Error("failed to delete config", zap.Error(err))
		return fmt.Errorf("failed to delete config: %w", err)
	}

	s.logger.Info("config deleted",
		zap.Int64("config_id", configID),
		zap.Int64("agent_id", agentID),
	)

	return nil
}

// ========== Specific Config Getters/Setters ==========

// GetNotificationConfig retrieves notification configuration
func (s *ConfigService) GetNotificationConfig(ctx context.Context, agentID int64) (*config.NotificationConfig, error) {
	cfg, err := s.configRepo.FindByKey(ctx, agentID, config.ConfigKeyNotifications, nil)
	if err != nil {
		if err == xerrors.ErrNotFound {
			// Return default config
			return s.getDefaultNotificationConfig(), nil
		}
		return nil, err
	}

	var notifConfig config.NotificationConfig
	if err := s.mapConfigValue(cfg.ConfigValue, &notifConfig); err != nil {
		return nil, err
	}

	return &notifConfig, nil
}

// SetNotificationConfig sets notification configuration
func (s *ConfigService) SetNotificationConfig(ctx context.Context, agentID int64, notifConfig *config.NotificationConfig) error {
	configValue := map[string]interface{}{
		"enabled":      notifConfig.Enabled,
		"sound":        notifConfig.Sound,
		"vibration":    notifConfig.Vibration,
		"email_alerts": notifConfig.EmailAlerts,
		"push_enabled": notifConfig.PushEnabled,
	}

	return s.setOrUpdateConfig(ctx, agentID, config.ConfigKeyNotifications, configValue, "Notification preferences")
}

// GetUSSDConfig retrieves USSD configuration
func (s *ConfigService) GetUSSDConfig(ctx context.Context, agentID int64) (*config.USSDConfig, error) {
	cfg, err := s.configRepo.FindByKey(ctx, agentID, config.ConfigKeyUSSDAutoRetry, nil)
	if err != nil {
		if err == xerrors.ErrNotFound {
			return s.getDefaultUSSDConfig(), nil
		}
		return nil, err
	}

	var ussdConfig config.USSDConfig
	if err := s.mapConfigValue(cfg.ConfigValue, &ussdConfig); err != nil {
		return nil, err
	}

	return &ussdConfig, nil
}

// SetUSSDConfig sets USSD configuration
func (s *ConfigService) SetUSSDConfig(ctx context.Context, agentID int64, ussdConfig *config.USSDConfig) error {
	configValue := map[string]interface{}{
		"auto_retry":           ussdConfig.AutoRetry,
		"retry_attempts":       ussdConfig.RetryAttempts,
		"timeout_seconds":      ussdConfig.TimeoutSeconds,
		"auto_dismiss_dialog":  ussdConfig.AutoDismissDialog,
	}

	return s.setOrUpdateConfig(ctx, agentID, config.ConfigKeyUSSDAutoRetry, configValue, "USSD processing settings")
}

// GetAndroidDeviceConfig retrieves Android device configuration
func (s *ConfigService) GetAndroidDeviceConfig(ctx context.Context, agentID int64, deviceID string) (*config.AndroidDeviceConfig, error) {
	cfg, err := s.configRepo.FindByKey(ctx, agentID, config.ConfigKeyAndroidDeviceEnabled, &deviceID)
	if err != nil {
		if err == xerrors.ErrNotFound {
			return s.getDefaultAndroidDeviceConfig(), nil
		}
		return nil, err
	}

	var deviceConfig config.AndroidDeviceConfig
	if err := s.mapConfigValue(cfg.ConfigValue, &deviceConfig); err != nil {
		return nil, err
	}

	return &deviceConfig, nil
}

// SetAndroidDeviceConfig sets Android device configuration
func (s *ConfigService) SetAndroidDeviceConfig(ctx context.Context, agentID int64, deviceID string, deviceConfig *config.AndroidDeviceConfig) error {
	configValue := map[string]interface{}{
		"enabled":                   deviceConfig.Enabled,
		"api_url":                   deviceConfig.APIURL,
		"api_key":                   deviceConfig.APIKey,
		"max_concurrent_requests":   deviceConfig.MaxConcurrent,
		"health_check_interval_seconds": deviceConfig.HealthCheckInterval,
	}

	// Create request with device_id
	req := &config.CreateConfigRequest{
		ConfigKey:   config.ConfigKeyAndroidDeviceEnabled,
		ConfigValue: configValue,
		Description: fmt.Sprintf("Android device configuration for device %s", deviceID),
		DeviceID:    deviceID,
		IsGlobal:    false,
	}

	// Check if exists
	exists, _ := s.configRepo.ExistsByKey(ctx, agentID, config.ConfigKeyAndroidDeviceEnabled, &deviceID)
	if exists {
		// Update
		cfg, _ := s.configRepo.FindByKey(ctx, agentID, config.ConfigKeyAndroidDeviceEnabled, &deviceID)
		cfg.ConfigValue = configValue
		return s.configRepo.Update(ctx, cfg.ID, cfg)
	}

	// Create
	cfg := &config.AgentConfig{
		AgentIdentityID: agentID,
		ConfigKey:       req.ConfigKey,
		ConfigValue:     req.ConfigValue,
		Description:     sql.NullString{String: req.Description, Valid: true},
		DeviceID:        sql.NullString{String: req.DeviceID, Valid: true},
		IsGlobal:        false,
	}

	return s.configRepo.Create(ctx, cfg)
}

// GetBusinessConfig retrieves business configuration
func (s *ConfigService) GetBusinessConfig(ctx context.Context, agentID int64) (*config.BusinessConfig, error) {
	cfg, err := s.configRepo.FindByKey(ctx, agentID, config.ConfigKeyAutoRenewalEnabled, nil)
	if err != nil {
		if err == xerrors.ErrNotFound {
			return s.getDefaultBusinessConfig(), nil
		}
		return nil, err
	}

	var businessConfig config.BusinessConfig
	if err := s.mapConfigValue(cfg.ConfigValue, &businessConfig); err != nil {
		return nil, err
	}

	return &businessConfig, nil
}

// SetBusinessConfig sets business configuration
func (s *ConfigService) SetBusinessConfig(ctx context.Context, agentID int64, businessConfig *config.BusinessConfig) error {
	configValue := map[string]interface{}{
		"auto_renewal_enabled":           businessConfig.AutoRenewalEnabled,
		"default_offer_validity_days":    businessConfig.DefaultOfferValidityDays,
		"max_offers_per_customer":        businessConfig.MaxOffersPerCustomer,
		"require_customer_verification":  businessConfig.RequireCustomerVerification,
	}

	return s.setOrUpdateConfig(ctx, agentID, config.ConfigKeyAutoRenewalEnabled, configValue, "Business settings")
}

// GetDisplayConfig retrieves display configuration
func (s *ConfigService) GetDisplayConfig(ctx context.Context, agentID int64) (*config.DisplayConfig, error) {
	cfg, err := s.configRepo.FindByKey(ctx, agentID, config.ConfigKeyTheme, nil)
	if err != nil {
		if err == xerrors.ErrNotFound {
			return s.getDefaultDisplayConfig(), nil
		}
		return nil, err
	}

	var displayConfig config.DisplayConfig
	if err := s.mapConfigValue(cfg.ConfigValue, &displayConfig); err != nil {
		return nil, err
	}

	return &displayConfig, nil
}

// SetDisplayConfig sets display configuration
func (s *ConfigService) SetDisplayConfig(ctx context.Context, agentID int64, displayConfig *config.DisplayConfig) error {
	configValue := map[string]interface{}{
		"theme":       displayConfig.Theme,
		"language":    displayConfig.Language,
		"timezone":    displayConfig.Timezone,
		"date_format": displayConfig.DateFormat,
		"currency":    displayConfig.Currency,
	}

	return s.setOrUpdateConfig(ctx, agentID, config.ConfigKeyTheme, configValue, "Display preferences")
}

// GetSecurityConfig retrieves security configuration
func (s *ConfigService) GetSecurityConfig(ctx context.Context, agentID int64) (*config.SecurityConfig, error) {
	cfg, err := s.configRepo.FindByKey(ctx, agentID, config.ConfigKey2FAEnabled, nil)
	if err != nil {
		if err == xerrors.ErrNotFound {
			return s.getDefaultSecurityConfig(), nil
		}
		return nil, err
	}

	var securityConfig config.SecurityConfig
	if err := s.mapConfigValue(cfg.ConfigValue, &securityConfig); err != nil {
		return nil, err
	}

	return &securityConfig, nil
}

// SetSecurityConfig sets security configuration
func (s *ConfigService) SetSecurityConfig(ctx context.Context, agentID int64, securityConfig *config.SecurityConfig) error {
	configValue := map[string]interface{}{
		"2fa_enabled":            securityConfig.TwoFactorEnabled,
		"session_timeout_minutes": securityConfig.SessionTimeoutMinutes,
		"ip_whitelist":           securityConfig.IPWhitelist,
		"allowed_devices":        securityConfig.AllowedDevices,
	}

	return s.setOrUpdateConfig(ctx, agentID, config.ConfigKey2FAEnabled, configValue, "Security settings")
}

// ========== Helper Methods ==========

// validateConfigKey validates config key format
func (s *ConfigService) validateConfigKey(key string) error {
	if key == "" {
		return fmt.Errorf("config key cannot be empty")
	}
	if len(key) > 255 {
		return fmt.Errorf("config key is too long")
	}
	return nil
}

// validateConfigValue validates config value based on key
func (s *ConfigService) validateConfigValue(key string, value map[string]interface{}) error {
	if value == nil {
		return fmt.Errorf("config value cannot be nil")
	}

	// Add specific validations based on config key
	switch key {
	case config.ConfigKeyUSSDTimeout:
		if timeout, ok := value["timeout_seconds"].(float64); ok {
			if timeout < 5 || timeout > 300 {
				return fmt.Errorf("USSD timeout must be between 5 and 300 seconds")
			}
		}
	case config.ConfigKeyUSSDRetryAttempts:
		if attempts, ok := value["retry_attempts"].(float64); ok {
			if attempts < 0 || attempts > 10 {
				return fmt.Errorf("retry attempts must be between 0 and 10")
			}
		}
	}

	return nil
}

// setOrUpdateConfig creates or updates a config
func (s *ConfigService) setOrUpdateConfig(ctx context.Context, agentID int64, key string, value map[string]interface{}, description string) error {
	exists, _ := s.configRepo.ExistsByKey(ctx, agentID, key, nil)

	if exists {
		// Update
		cfg, _ := s.configRepo.FindByKey(ctx, agentID, key, nil)
		cfg.ConfigValue = value
		return s.configRepo.Update(ctx, cfg.ID, cfg)
	}

	// Create
	req := &config.CreateConfigRequest{
		ConfigKey:   key,
		ConfigValue: value,
		Description: description,
		IsGlobal:    true,
	}

	_, err := s.CreateConfig(ctx, agentID, req)
	return err
}

// mapConfigValue maps config value to struct
func (s *ConfigService) mapConfigValue(value map[string]interface{}, target interface{}) error {
	// Simple type assertion mapping
	// In production, use a proper mapper like mapstructure
	return nil
}

// Default config getters
func (s *ConfigService) getDefaultNotificationConfig() *config.NotificationConfig {
	return &config.NotificationConfig{
		Enabled:     true,
		Sound:       true,
		Vibration:   true,
		EmailAlerts: false,
		PushEnabled: true,
	}
}

func (s *ConfigService) getDefaultUSSDConfig() *config.USSDConfig {
	return &config.USSDConfig{
		AutoRetry:        true,
		RetryAttempts:    3,
		TimeoutSeconds:   30,
		AutoDismissDialog: true,
	}
}

func (s *ConfigService) getDefaultAndroidDeviceConfig() *config.AndroidDeviceConfig {
	return &config.AndroidDeviceConfig{
		Enabled:             false,
		MaxConcurrent:       5,
		HealthCheckInterval: 60,
	}
}

func (s *ConfigService) getDefaultBusinessConfig() *config.BusinessConfig {
	return &config.BusinessConfig{
		AutoRenewalEnabled:           false,
		DefaultOfferValidityDays:     30,
		MaxOffersPerCustomer:         10,
		RequireCustomerVerification:  false,
	}
}

func (s *ConfigService) getDefaultDisplayConfig() *config.DisplayConfig {
	return &config.DisplayConfig{
		Theme:      "light",
		Language:   "en",
		Timezone:   "Africa/Nairobi",
		DateFormat: "DD/MM/YYYY",
		Currency:   "KES",
	}
}

func (s *ConfigService) getDefaultSecurityConfig() *config.SecurityConfig {
	return &config.SecurityConfig{
		TwoFactorEnabled:      false,
		SessionTimeoutMinutes: 30,
		IPWhitelist:           []string{},
		AllowedDevices:        5,
	}
}