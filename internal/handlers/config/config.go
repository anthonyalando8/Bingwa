// internal/handlers/config/config_handler.go
package config

import (
	"net/http"
	"strconv"

	"bingwa-service/internal/domain/config"
	"bingwa-service/internal/middleware"
	"bingwa-service/internal/pkg/response"
	service "bingwa-service/internal/service/config"

	"github.com/gin-gonic/gin"
)

type ConfigHandler struct {
	configService *service.ConfigService
}

func NewConfigHandler(configService *service.ConfigService) *ConfigHandler {
	return &ConfigHandler{
		configService: configService,
	}
}

// ========== General Config Endpoints ==========

// CreateConfig creates a new configuration
func (h *ConfigHandler) CreateConfig(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	var req config.CreateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	result, err := h.configService.CreateConfig(c.Request.Context(), agentID, &req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "failed to create config", err)
		return
	}

	response.Success(c, http.StatusCreated, "config created successfully", result)
}

// GetConfig retrieves a configuration by ID
func (h *ConfigHandler) GetConfig(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	configIDStr := c.Param("id")
	configID, err := strconv.ParseInt(configIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid config ID", err)
		return
	}

	result, err := h.configService.GetConfig(c.Request.Context(), agentID, configID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "config not found", err)
		return
	}

	response.Success(c, http.StatusOK, "config retrieved", result)
}

// GetConfigByKey retrieves a configuration by key
func (h *ConfigHandler) GetConfigByKey(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	configKey := c.Param("key")
	if configKey == "" {
		response.Error(c, http.StatusBadRequest, "config key is required", nil)
		return
	}

	deviceID := c.Query("device_id")
	var deviceIDPtr *string
	if deviceID != "" {
		deviceIDPtr = &deviceID
	}

	result, err := h.configService.GetConfigByKey(c.Request.Context(), agentID, configKey, deviceIDPtr)
	if err != nil {
		response.Error(c, http.StatusNotFound, "config not found", err)
		return
	}

	response.Success(c, http.StatusOK, "config retrieved", result)
}

// ListConfigs retrieves configurations with filters
func (h *ConfigHandler) ListConfigs(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	var filters config.ConfigListFilters
	if err := c.ShouldBindQuery(&filters); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid query parameters", err)
		return
	}

	result, err := h.configService.ListConfigs(c.Request.Context(), agentID, &filters)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list configs", err)
		return
	}

	response.Success(c, http.StatusOK, "configs retrieved", result)
}

// GetAllConfigs retrieves all configurations
func (h *ConfigHandler) GetAllConfigs(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	configs, err := h.configService.GetAllConfigs(c.Request.Context(), agentID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get configs", err)
		return
	}

	response.Success(c, http.StatusOK, "all configs retrieved", gin.H{
		"configs": configs,
		"count":   len(configs),
	})
}

// GetGlobalConfigs retrieves global configurations
func (h *ConfigHandler) GetGlobalConfigs(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	configs, err := h.configService.GetGlobalConfigs(c.Request.Context(), agentID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get global configs", err)
		return
	}

	response.Success(c, http.StatusOK, "global configs retrieved", gin.H{
		"configs": configs,
		"count":   len(configs),
	})
}

// GetDeviceConfigs retrieves device-specific configurations
func (h *ConfigHandler) GetDeviceConfigs(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	deviceID := c.Param("device_id")
	if deviceID == "" {
		response.Error(c, http.StatusBadRequest, "device_id is required", nil)
		return
	}

	configs, err := h.configService.GetDeviceConfigs(c.Request.Context(), agentID, deviceID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get device configs", err)
		return
	}

	response.Success(c, http.StatusOK, "device configs retrieved", gin.H{
		"device_id": deviceID,
		"configs":   configs,
		"count":     len(configs),
	})
}

// UpdateConfig updates a configuration
func (h *ConfigHandler) UpdateConfig(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	configIDStr := c.Param("id")
	configID, err := strconv.ParseInt(configIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid config ID", err)
		return
	}

	var req config.UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	result, err := h.configService.UpdateConfig(c.Request.Context(), agentID, configID, &req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "failed to update config", err)
		return
	}

	response.Success(c, http.StatusOK, "config updated successfully", result)
}

// DeleteConfig deletes a configuration
func (h *ConfigHandler) DeleteConfig(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	configIDStr := c.Param("id")
	configID, err := strconv.ParseInt(configIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid config ID", err)
		return
	}

	if err := h.configService.DeleteConfig(c.Request.Context(), agentID, configID); err != nil {
		response.Error(c, http.StatusBadRequest, "failed to delete config", err)
		return
	}

	response.Success(c, http.StatusOK, "config deleted successfully", nil)
}

// ========== Specific Config Type Endpoints ==========

// GetNotificationConfig retrieves notification configuration
func (h *ConfigHandler) GetNotificationConfig(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	result, err := h.configService.GetNotificationConfig(c.Request.Context(), agentID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get notification config", err)
		return
	}

	response.Success(c, http.StatusOK, "notification config retrieved", result)
}

// SetNotificationConfig sets notification configuration
func (h *ConfigHandler) SetNotificationConfig(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	var req config.NotificationConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	if err := h.configService.SetNotificationConfig(c.Request.Context(), agentID, &req); err != nil {
		response.Error(c, http.StatusBadRequest, "failed to set notification config", err)
		return
	}

	response.Success(c, http.StatusOK, "notification config saved successfully", req)
}

// GetUSSDConfig retrieves USSD configuration
func (h *ConfigHandler) GetUSSDConfig(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	result, err := h.configService.GetUSSDConfig(c.Request.Context(), agentID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get USSD config", err)
		return
	}

	response.Success(c, http.StatusOK, "USSD config retrieved", result)
}

// SetUSSDConfig sets USSD configuration
func (h *ConfigHandler) SetUSSDConfig(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	var req config.USSDConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	if err := h.configService.SetUSSDConfig(c.Request.Context(), agentID, &req); err != nil {
		response.Error(c, http.StatusBadRequest, "failed to set USSD config", err)
		return
	}

	response.Success(c, http.StatusOK, "USSD config saved successfully", req)
}

// GetAndroidDeviceConfig retrieves Android device configuration
func (h *ConfigHandler) GetAndroidDeviceConfig(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	deviceID := c.Param("device_id")
	if deviceID == "" {
		response.Error(c, http.StatusBadRequest, "device_id is required", nil)
		return
	}

	result, err := h.configService.GetAndroidDeviceConfig(c.Request.Context(), agentID, deviceID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get Android device config", err)
		return
	}

	response.Success(c, http.StatusOK, "Android device config retrieved", result)
}

// SetAndroidDeviceConfig sets Android device configuration
func (h *ConfigHandler) SetAndroidDeviceConfig(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	deviceID := c.Param("device_id")
	if deviceID == "" {
		response.Error(c, http.StatusBadRequest, "device_id is required", nil)
		return
	}

	var req config.AndroidDeviceConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	if err := h.configService.SetAndroidDeviceConfig(c.Request.Context(), agentID, deviceID, &req); err != nil {
		response.Error(c, http.StatusBadRequest, "failed to set Android device config", err)
		return
	}

	response.Success(c, http.StatusOK, "Android device config saved successfully", req)
}

// GetBusinessConfig retrieves business configuration
func (h *ConfigHandler) GetBusinessConfig(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	result, err := h.configService.GetBusinessConfig(c.Request.Context(), agentID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get business config", err)
		return
	}

	response.Success(c, http.StatusOK, "business config retrieved", result)
}

// SetBusinessConfig sets business configuration
func (h *ConfigHandler) SetBusinessConfig(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	var req config.BusinessConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	if err := h.configService.SetBusinessConfig(c.Request.Context(), agentID, &req); err != nil {
		response.Error(c, http.StatusBadRequest, "failed to set business config", err)
		return
	}

	response.Success(c, http.StatusOK, "business config saved successfully", req)
}

// GetDisplayConfig retrieves display configuration
func (h *ConfigHandler) GetDisplayConfig(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	result, err := h.configService.GetDisplayConfig(c.Request.Context(), agentID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get display config", err)
		return
	}

	response.Success(c, http.StatusOK, "display config retrieved", result)
}

// SetDisplayConfig sets display configuration
func (h *ConfigHandler) SetDisplayConfig(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	var req config.DisplayConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	if err := h.configService.SetDisplayConfig(c.Request.Context(), agentID, &req); err != nil {
		response.Error(c, http.StatusBadRequest, "failed to set display config", err)
		return
	}

	response.Success(c, http.StatusOK, "display config saved successfully", req)
}

// GetSecurityConfig retrieves security configuration
func (h *ConfigHandler) GetSecurityConfig(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	result, err := h.configService.GetSecurityConfig(c.Request.Context(), agentID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get security config", err)
		return
	}

	response.Success(c, http.StatusOK, "security config retrieved", result)
}

// SetSecurityConfig sets security configuration
func (h *ConfigHandler) SetSecurityConfig(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	var req config.SecurityConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	if err := h.configService.SetSecurityConfig(c.Request.Context(), agentID, &req); err != nil {
		response.Error(c, http.StatusBadRequest, "failed to set security config", err)
		return
	}

	response.Success(c, http.StatusOK, "security config saved successfully", req)
}