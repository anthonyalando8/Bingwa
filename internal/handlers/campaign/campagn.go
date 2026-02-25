// internal/handlers/campaign/campaign_handler.go
package campaign

import (
	"net/http"
	"strconv"

	"bingwa-service/internal/domain/campaign"
	//"bingwa-service/internal/middleware"
	"bingwa-service/internal/pkg/response"
	service "bingwa-service/internal/service/campaign"

	"github.com/gin-gonic/gin"
)

type CampaignHandler struct {
	campaignService *service.CampaignService
}

func NewCampaignHandler(campaignService *service.CampaignService) *CampaignHandler {
	return &CampaignHandler{
		campaignService: campaignService,
	}
}

// ========== Admin Only Endpoints ==========

// CreateCampaign creates a new promotional campaign (admin only)
func (h *CampaignHandler) CreateCampaign(c *gin.Context) {
	var req campaign.CreateCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	result, err := h.campaignService.CreateCampaign(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "failed to create campaign", err)
		return
	}

	response.Success(c, http.StatusCreated, "campaign created successfully", result)
}

// UpdateCampaign updates a promotional campaign (admin only)
func (h *CampaignHandler) UpdateCampaign(c *gin.Context) {
	campaignIDStr := c.Param("id")
	campaignID, err := strconv.ParseInt(campaignIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid campaign ID", err)
		return
	}

	var req campaign.UpdateCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	result, err := h.campaignService.UpdateCampaign(c.Request.Context(), campaignID, &req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "failed to update campaign", err)
		return
	}

	response.Success(c, http.StatusOK, "campaign updated successfully", result)
}

// ActivateCampaign activates a campaign (admin only)
func (h *CampaignHandler) ActivateCampaign(c *gin.Context) {
	campaignIDStr := c.Param("id")
	campaignID, err := strconv.ParseInt(campaignIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid campaign ID", err)
		return
	}

	if err := h.campaignService.ActivateCampaign(c.Request.Context(), campaignID); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to activate campaign", err)
		return
	}

	response.Success(c, http.StatusOK, "campaign activated successfully", nil)
}

// DeactivateCampaign deactivates a campaign (admin only)
func (h *CampaignHandler) DeactivateCampaign(c *gin.Context) {
	campaignIDStr := c.Param("id")
	campaignID, err := strconv.ParseInt(campaignIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid campaign ID", err)
		return
	}

	if err := h.campaignService.DeactivateCampaign(c.Request.Context(), campaignID); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to deactivate campaign", err)
		return
	}

	response.Success(c, http.StatusOK, "campaign deactivated successfully", nil)
}

// DeleteCampaign deletes a campaign (admin only)
func (h *CampaignHandler) DeleteCampaign(c *gin.Context) {
	campaignIDStr := c.Param("id")
	campaignID, err := strconv.ParseInt(campaignIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid campaign ID", err)
		return
	}

	if err := h.campaignService.DeleteCampaign(c.Request.Context(), campaignID); err != nil {
		response.Error(c, http.StatusBadRequest, "failed to delete campaign", err)
		return
	}

	response.Success(c, http.StatusOK, "campaign deleted successfully", nil)
}

// ExtendCampaign extends campaign duration (admin only)
func (h *CampaignHandler) ExtendCampaign(c *gin.Context) {
	campaignIDStr := c.Param("id")
	campaignID, err := strconv.ParseInt(campaignIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid campaign ID", err)
		return
	}

	var req struct {
		Days int `json:"days" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	if err := h.campaignService.ExtendCampaign(c.Request.Context(), campaignID, req.Days); err != nil {
		response.Error(c, http.StatusBadRequest, "failed to extend campaign", err)
		return
	}

	response.Success(c, http.StatusOK, "campaign extended successfully", gin.H{
		"extended_by_days": req.Days,
	})
}

// GetCampaignStats retrieves campaign statistics (admin only)
func (h *CampaignHandler) GetCampaignStats(c *gin.Context) {
	stats, err := h.campaignService.GetCampaignStats(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get campaign stats", err)
		return
	}

	response.Success(c, http.StatusOK, "campaign stats retrieved", stats)
}

// ========== Public/User Endpoints ==========

// GetCampaign retrieves a campaign by ID
func (h *CampaignHandler) GetCampaign(c *gin.Context) {
	campaignIDStr := c.Param("id")
	campaignID, err := strconv.ParseInt(campaignIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid campaign ID", err)
		return
	}

	result, err := h.campaignService.GetCampaign(c.Request.Context(), campaignID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "campaign not found", err)
		return
	}

	response.Success(c, http.StatusOK, "campaign retrieved", result)
}

// GetCampaignByCode retrieves a campaign by promotional code
func (h *CampaignHandler) GetCampaignByCode(c *gin.Context) {
	promoCode := c.Param("code")
	if promoCode == "" {
		response.Error(c, http.StatusBadRequest, "promotional code is required", nil)
		return
	}

	result, err := h.campaignService.GetCampaignByCode(c.Request.Context(), promoCode)
	if err != nil {
		response.Error(c, http.StatusNotFound, "campaign not found", err)
		return
	}

	response.Success(c, http.StatusOK, "campaign retrieved", result)
}

// ListCampaigns retrieves campaigns with filters
func (h *CampaignHandler) ListCampaigns(c *gin.Context) {
	var filters campaign.CampaignListFilters
	if err := c.ShouldBindQuery(&filters); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid query parameters", err)
		return
	}

	result, err := h.campaignService.ListCampaigns(c.Request.Context(), &filters)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list campaigns", err)
		return
	}

	response.Success(c, http.StatusOK, "campaigns retrieved", result)
}

// GetActiveCampaigns retrieves currently active campaigns
func (h *CampaignHandler) GetActiveCampaigns(c *gin.Context) {
	campaigns, err := h.campaignService.GetActiveCampaigns(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get active campaigns", err)
		return
	}

	response.Success(c, http.StatusOK, "active campaigns retrieved", gin.H{
		"campaigns": campaigns,
		"count":     len(campaigns),
	})
}

// ValidateCampaign validates a promotional code
func (h *CampaignHandler) ValidateCampaign(c *gin.Context) {
	var req campaign.ValidateCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	result, err := h.campaignService.ValidateCampaign(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to validate campaign", err)
		return
	}

	if result.Valid {
		response.Success(c, http.StatusOK, result.Message, result)
	} else {
		response.Success(c, http.StatusOK, result.Message, result)
	}
}

// ApplyCampaign applies a promotional code and calculates discount
func (h *CampaignHandler) ApplyCampaign(c *gin.Context) {
	campaignIDStr := c.Param("id")
	campaignID, err := strconv.ParseInt(campaignIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid campaign ID", err)
		return
	}

	var req struct {
		OriginalPrice float64 `json:"original_price" binding:"required,min=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	discountAmount, finalPrice, err := h.campaignService.ApplyCampaign(c.Request.Context(), campaignID, req.OriginalPrice)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "failed to apply campaign", err)
		return
	}

	response.Success(c, http.StatusOK, "discount calculated", gin.H{
		"original_price":  req.OriginalPrice,
		"discount_amount": discountAmount,
		"final_price":     finalPrice,
		"savings":         discountAmount,
	})
}

// GetCampaignDetails retrieves detailed campaign information
func (h *CampaignHandler) GetCampaignDetails(c *gin.Context) {
	campaignIDStr := c.Param("id")
	campaignID, err := strconv.ParseInt(campaignIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid campaign ID", err)
		return
	}

	campaign, err := h.campaignService.GetCampaign(c.Request.Context(), campaignID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "campaign not found", err)
		return
	}

	// Add calculated fields
	details := gin.H{
		"campaign":         campaign,
		"is_active":        h.campaignService.IsCampaignActive(campaign),
		"usage_percentage": h.campaignService.GetCampaignUsagePercentage(campaign),
		"remaining_uses":   h.campaignService.GetRemainingUses(campaign),
		"days_remaining":   h.campaignService.GetCampaignDaysRemaining(campaign),
	}

	response.Success(c, http.StatusOK, "campaign details retrieved", details)
}

// CheckCampaignAvailability checks if a promotional code is available
func (h *CampaignHandler) CheckCampaignAvailability(c *gin.Context) {
	promoCode := c.Query("code")
	if promoCode == "" {
		response.Error(c, http.StatusBadRequest, "promotional code is required", nil)
		return
	}

	campaign, err := h.campaignService.GetCampaignByCode(c.Request.Context(), promoCode)
	if err != nil {
		response.Success(c, http.StatusOK, "promotional code not found", gin.H{
			"available": false,
			"message":   "Invalid promotional code",
		})
		return
	}

	isActive := h.campaignService.IsCampaignActive(campaign)

	message := "Promotional code is available"
	if !isActive {
		if campaign.Status != "active" {
			message = "Promotional code is not active"
		} else if campaign.MaxUses.Valid && campaign.CurrentUses >= int(campaign.MaxUses.Int32) {
			message = "Promotional code has reached its usage limit"
		} else {
			message = "Promotional code is expired or not yet started"
		}
	}

	response.Success(c, http.StatusOK, message, gin.H{
		"available":      isActive,
		"promotional_code": campaign.PromotionalCode,
		"campaign_name":  campaign.Name,
		"discount_type":  campaign.DiscountType,
		"discount_value": campaign.DiscountValue,
		"message":        message,
	})
}