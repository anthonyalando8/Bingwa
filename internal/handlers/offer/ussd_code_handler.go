// internal/handlers/offer/ussd_code_handler.go
package offer

import (
	"fmt"
	"net/http"
	"strconv"

	"bingwa-service/internal/domain/offer"
	"bingwa-service/internal/middleware"
	"bingwa-service/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// ========== USSD Code Management ==========

// AddUSSDCode adds a new USSD code to an offer
func (h *OfferHandler) AddUSSDCode(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	offerIDStr := c.Param("id")
	offerID, err := strconv.ParseInt(offerIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid offer ID", err)
		return
	}

	var req offer.AddUSSDCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	result, err := h.offerService.AddUSSDCode(c.Request.Context(), agentID, offerID, &req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "failed to add USSD code", err)
		return
	}

	response.Success(c, http.StatusCreated, "USSD code added successfully", result)
}

// ListUSSDCodes lists all USSD codes for an offer
func (h *OfferHandler) ListUSSDCodes(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	offerIDStr := c.Param("id")
	offerID, err := strconv.ParseInt(offerIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid offer ID", err)
		return
	}

	codes, err := h.offerService.ListUSSDCodes(c.Request.Context(), agentID, offerID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list USSD codes", err)
		return
	}

	response.Success(c, http.StatusOK, "USSD codes retrieved", gin.H{
		"codes": codes,
		"count": len(codes),
	})
}

// GetActiveUSSDCodes gets active USSD codes sorted by priority
func (h *OfferHandler) GetActiveUSSDCodes(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	offerIDStr := c.Param("id")
	offerID, err := strconv.ParseInt(offerIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid offer ID", err)
		return
	}

	codes, err := h.offerService.GetActiveUSSDCodes(c.Request.Context(), agentID, offerID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get active USSD codes", err)
		return
	}

	response.Success(c, http.StatusOK, "active USSD codes retrieved", gin.H{
		"codes": codes,
		"count": len(codes),
	})
}

// GetPrimaryUSSDCode gets the primary USSD code
func (h *OfferHandler) GetPrimaryUSSDCode(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	offerIDStr := c.Param("id")
	offerID, err := strconv.ParseInt(offerIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid offer ID", err)
		return
	}

	code, err := h.offerService.GetPrimaryUSSDCode(c.Request.Context(), agentID, offerID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "no primary USSD code found", err)
		return
	}

	response.Success(c, http.StatusOK, "primary USSD code retrieved", code)
}

// UpdateUSSDCode updates a USSD code
func (h *OfferHandler) UpdateUSSDCode(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	offerIDStr := c.Param("id")
	offerID, err := strconv.ParseInt(offerIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid offer ID", err)
		return
	}

	ussdCodeIDStr := c.Param("ussd_code_id")
	ussdCodeID, err := strconv.ParseInt(ussdCodeIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid USSD code ID", err)
		return
	}

	var req offer.UpdateUSSDCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	result, err := h.offerService.UpdateUSSDCode(c.Request.Context(), agentID, offerID, ussdCodeID, &req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "failed to update USSD code", err)
		return
	}

	response.Success(c, http.StatusOK, "USSD code updated successfully", result)
}

// SetUSSDCodeAsPrimary sets a USSD code as primary (priority 1)
func (h *OfferHandler) SetUSSDCodeAsPrimary(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	offerIDStr := c.Param("id")
	offerID, err := strconv.ParseInt(offerIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid offer ID", err)
		return
	}

	ussdCodeIDStr := c.Param("ussd_code_id")
	ussdCodeID, err := strconv.ParseInt(ussdCodeIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid USSD code ID", err)
		return
	}

	if err := h.offerService.SetUSSDCodeAsPrimary(c.Request.Context(), agentID, offerID, ussdCodeID); err != nil {
		response.Error(c, http.StatusBadRequest, "failed to set as primary", err)
		return
	}

	response.Success(c, http.StatusOK, "USSD code set as primary successfully", nil)
}

// ReorderUSSDCodes reorders USSD codes
func (h *OfferHandler) ReorderUSSDCodes(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	offerIDStr := c.Param("id")
	offerID, err := strconv.ParseInt(offerIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid offer ID", err)
		return
	}

	var req offer.ReorderUSSDCodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	if err := h.offerService.ReorderUSSDCodes(c.Request.Context(), agentID, offerID, &req); err != nil {
		response.Error(c, http.StatusBadRequest, "failed to reorder USSD codes", err)
		return
	}

	response.Success(c, http.StatusOK, "USSD codes reordered successfully", nil)
}

// ToggleUSSDCodeStatus toggles USSD code active status
func (h *OfferHandler) ToggleUSSDCodeStatus(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	offerIDStr := c.Param("id")
	offerID, err := strconv.ParseInt(offerIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid offer ID", err)
		return
	}

	ussdCodeIDStr := c.Param("ussd_code_id")
	ussdCodeID, err := strconv.ParseInt(ussdCodeIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid USSD code ID", err)
		return
	}

	var req struct {
		IsActive bool `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	if err := h.offerService.ToggleUSSDCodeStatus(c.Request.Context(), agentID, offerID, ussdCodeID, req.IsActive); err != nil {
		response.Error(c, http.StatusBadRequest, "failed to toggle status", err)
		return
	}

	status := "deactivated"
	if req.IsActive {
		status = "activated"
	}

	response.Success(c, http.StatusOK, fmt.Sprintf("USSD code %s successfully", status), nil)
}

// DeleteUSSDCode deletes a USSD code
func (h *OfferHandler) DeleteUSSDCode(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	offerIDStr := c.Param("id")
	offerID, err := strconv.ParseInt(offerIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid offer ID", err)
		return
	}

	ussdCodeIDStr := c.Param("ussd_code_id")
	ussdCodeID, err := strconv.ParseInt(ussdCodeIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid USSD code ID", err)
		return
	}

	if err := h.offerService.DeleteUSSDCode(c.Request.Context(), agentID, offerID, ussdCodeID); err != nil {
		response.Error(c, http.StatusBadRequest, "failed to delete USSD code", err)
		return
	}

	response.Success(c, http.StatusOK, "USSD code deleted successfully", nil)
}

// RecordUSSDResult records the result of a USSD execution
func (h *OfferHandler) RecordUSSDResult(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	offerIDStr := c.Param("id")
	offerID, err := strconv.ParseInt(offerIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid offer ID", err)
		return
	}

	var req offer.RecordUSSDResultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	if err := h.offerService.RecordUSSDResult(c.Request.Context(), agentID, offerID, &req); err != nil {
		response.Error(c, http.StatusBadRequest, "failed to record result", err)
		return
	}

	response.Success(c, http.StatusOK, "USSD result recorded successfully", nil)
}

// GetUSSDCodeStats gets statistics for USSD codes
func (h *OfferHandler) GetUSSDCodeStats(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	offerIDStr := c.Param("id")
	offerID, err := strconv.ParseInt(offerIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid offer ID", err)
		return
	}

	stats, err := h.offerService.GetUSSDCodeStats(c.Request.Context(), agentID, offerID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get stats", err)
		return
	}

	response.Success(c, http.StatusOK, "USSD code statistics retrieved", stats)
}