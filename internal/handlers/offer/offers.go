// internal/handlers/offer/offer_handler.go
package offer

import (
	"net/http"
	"strconv"

	"bingwa-service/internal/domain/offer"
	"bingwa-service/internal/middleware"
	xerrors "bingwa-service/internal/pkg/errors"
	"bingwa-service/internal/pkg/response"
	service "bingwa-service/internal/service/offer"

	"github.com/gin-gonic/gin"
)

type OfferHandler struct {
	offerService *service.OfferService
}

func NewOfferHandler(offerService *service.OfferService) *OfferHandler {
	return &OfferHandler{
		offerService: offerService,
	}
}

// ========== Agent Endpoints ==========

// CreateOffer creates a new offer
func (h *OfferHandler) CreateOffer(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	var req offer.CreateOfferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	result, err := h.offerService.CreateOffer(c.Request.Context(), agentID, &req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "failed to create offer", err)
		return
	}

	response.Success(c, http.StatusCreated, "offer created successfully", result)
}

// GetOffer retrieves an offer by ID
func (h *OfferHandler) GetOffer(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	offerIDStr := c.Param("id")
	offerID, err := strconv.ParseInt(offerIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid offer ID", err)
		return
	}

	result, err := h.offerService.GetOffer(c.Request.Context(), agentID, offerID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "offer not found", err)
		return
	}

	response.Success(c, http.StatusOK, "offer retrieved", result)
}

// GetOfferByCode retrieves an offer by offer code
func (h *OfferHandler) GetOfferByCode(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	offerCode := c.Param("code")
	if offerCode == "" {
		response.Error(c, http.StatusBadRequest, "offer code is required", nil)
		return
	}

	result, err := h.offerService.GetOfferByCode(c.Request.Context(), agentID, offerCode)
	if err != nil {
		response.Error(c, http.StatusNotFound, "offer not found", err)
		return
	}

	response.Success(c, http.StatusOK, "offer retrieved", result)
}

// ListOffers retrieves offers with filters
func (h *OfferHandler) ListOffers(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	var filters offer.OfferListFilters
	if err := c.ShouldBindQuery(&filters); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid query parameters", err)
		return
	}

	result, err := h.offerService.ListOffers(c.Request.Context(), agentID, &filters)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list offers", err)
		return
	}

	response.Success(c, http.StatusOK, "offers retrieved", result)
}

// GetFeaturedOffers retrieves featured offers
func (h *OfferHandler) GetFeaturedOffers(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 50 {
		limit = 10
	}

	offers, err := h.offerService.GetFeaturedOffers(c.Request.Context(), agentID, limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get featured offers", err)
		return
	}

	response.Success(c, http.StatusOK, "featured offers retrieved", gin.H{
		"offers": offers,
		"count":  len(offers),
	})
}

// UpdateOffer updates an offer
func (h *OfferHandler) UpdateOffer(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	offerIDStr := c.Param("id")
	offerID, err := strconv.ParseInt(offerIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid offer ID", err)
		return
	}

	var req offer.UpdateOfferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	result, err := h.offerService.UpdateOffer(c.Request.Context(), agentID, offerID, &req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "failed to update offer", err)
		return
	}

	response.Success(c, http.StatusOK, "offer updated successfully", result)
}

// ActivateOffer activates an offer
func (h *OfferHandler) ActivateOffer(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	offerIDStr := c.Param("id")
	offerID, err := strconv.ParseInt(offerIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid offer ID", err)
		return
	}

	if err := h.offerService.ActivateOffer(c.Request.Context(), agentID, offerID); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to activate offer", err)
		return
	}

	response.Success(c, http.StatusOK, "offer activated successfully", nil)
}

// DeactivateOffer deactivates an offer
func (h *OfferHandler) DeactivateOffer(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	offerIDStr := c.Param("id")
	offerID, err := strconv.ParseInt(offerIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid offer ID", err)
		return
	}

	if err := h.offerService.DeactivateOffer(c.Request.Context(), agentID, offerID); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to deactivate offer", err)
		return
	}

	response.Success(c, http.StatusOK, "offer deactivated successfully", nil)
}

// PauseOffer pauses an offer
func (h *OfferHandler) PauseOffer(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	offerIDStr := c.Param("id")
	offerID, err := strconv.ParseInt(offerIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid offer ID", err)
		return
	}

	if err := h.offerService.PauseOffer(c.Request.Context(), agentID, offerID); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to pause offer", err)
		return
	}

	response.Success(c, http.StatusOK, "offer paused successfully", nil)
}

// DeleteOffer soft deletes an offer
func (h *OfferHandler) DeleteOffer(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	offerIDStr := c.Param("id")
	offerID, err := strconv.ParseInt(offerIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid offer ID", err)
		return
	}

	if err := h.offerService.DeleteOffer(c.Request.Context(), agentID, offerID); err != nil {
		response.Error(c, http.StatusBadRequest, "failed to delete offer", err)
		return
	}

	response.Success(c, http.StatusOK, "offer deleted successfully", nil)
}

// GetOfferStats retrieves offer statistics
func (h *OfferHandler) GetOfferStats(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	stats, err := h.offerService.GetOfferStats(c.Request.Context(), agentID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get offer stats", err)
		return
	}

	response.Success(c, http.StatusOK, "offer stats retrieved", stats)
}

// SearchOffers searches offers
func (h *OfferHandler) SearchOffers(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	query := c.Query("q")
	if query == "" {
		response.Error(c, http.StatusBadRequest, "search query is required", nil)
		return
	}

	results, err := h.offerService.SearchOffers(c.Request.Context(), agentID, query)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to search offers", err)
		return
	}

	response.Success(c, http.StatusOK, "search results", gin.H{
		"offers": results,
		"count":  len(results),
	})
}

// CloneOffer clones an existing offer
func (h *OfferHandler) CloneOffer(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	offerIDStr := c.Param("id")
	offerID, err := strconv.ParseInt(offerIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid offer ID", err)
		return
	}

	var req struct {
		NewName string `json:"new_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	result, err := h.offerService.CloneOffer(c.Request.Context(), agentID, offerID, req.NewName)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "failed to clone offer", err)
		return
	}

	response.Success(c, http.StatusCreated, "offer cloned successfully", result)
}

// GenerateUSSDCode generates USSD code for an offer
func (h *OfferHandler) GenerateUSSDCode(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	offerIDStr := c.Param("id")
	offerID, err := strconv.ParseInt(offerIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid offer ID", err)
		return
	}

	phoneNumber := c.Query("phone")
	if phoneNumber == "" {
		response.Error(c, http.StatusBadRequest, "phone number is required", nil)
		return
	}

	// Get offer
	o, err := h.offerService.GetOffer(c.Request.Context(), agentID, offerID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "offer not found", err)
		return
	}

	// Generate USSD code
	ussdCode := h.offerService.GenerateUSSDCode(o, phoneNumber)

	response.Success(c, http.StatusOK, "USSD code generated", gin.H{
		"ussd_code":    ussdCode,
		"phone_number": phoneNumber,
		"offer_id":     offerID,
		"offer_code":   o.OfferCode,
	})
}

// CalculateOfferPrice calculates discounted price for an offer
func (h *OfferHandler) CalculateOfferPrice(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	offerIDStr := c.Param("id")
	offerID, err := strconv.ParseInt(offerIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid offer ID", err)
		return
	}

	// Get offer
	o, err := h.offerService.GetOffer(c.Request.Context(), agentID, offerID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "offer not found", err)
		return
	}

	// Calculate discounted price
	discountedPrice := h.offerService.CalculateDiscountedPrice(o)

	response.Success(c, http.StatusOK, "price calculated", gin.H{
		"original_price":    o.Price,
		"discount_percent":  o.DiscountPercentage,
		"discounted_price":  discountedPrice,
		"savings":           o.Price - discountedPrice,
		"currency":          o.Currency,
	})
}

// CheckOfferAvailability checks if an offer is currently available
func (h *OfferHandler) CheckOfferAvailability(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	offerIDStr := c.Param("id")
	offerID, err := strconv.ParseInt(offerIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid offer ID", err)
		return
	}

	// Get offer
	o, err := h.offerService.GetOffer(c.Request.Context(), agentID, offerID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "offer not found", err)
		return
	}

	// Check availability
	isAvailable := h.offerService.IsOfferAvailable(o)

	response.Success(c, http.StatusOK, "availability checked", gin.H{
		"offer_id":     offerID,
		"offer_code":   o.OfferCode,
		"is_available": isAvailable,
		"status":       o.Status,
		"available_from": o.AvailableFrom,
		"available_until": o.AvailableUntil,
	})
}

// GetOffersByAmount retrieves offers by amount
func (h *OfferHandler) GetOffersByAmount(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	amountStr := c.Query("amount")
	if amountStr == "" {
		response.Error(c, http.StatusBadRequest, "amount is required", nil)
		return
	}

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid amount", err)
		return
	}

	offers, err := h.offerService.GetOffersByAmount(c.Request.Context(), agentID, amount)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get offers", err)
		return
	}

	response.Success(c, http.StatusOK, "offers retrieved", offers)
}

// GetOffersByAmountRange retrieves offers by amount range
func (h *OfferHandler) GetOffersByAmountRange(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	minAmountStr := c.Query("min_amount")
	maxAmountStr := c.Query("max_amount")

	if minAmountStr == "" || maxAmountStr == "" {
		response.Error(c, http.StatusBadRequest, "min_amount and max_amount are required", nil)
		return
	}

	minAmount, err := strconv.ParseFloat(minAmountStr, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid min_amount", err)
		return
	}

	maxAmount, err := strconv.ParseFloat(maxAmountStr, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid max_amount", err)
		return
	}

	offers, err := h.offerService.GetOffersByAmountRange(c.Request.Context(), agentID, minAmount, maxAmount)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get offers", err)
		return
	}

	response.Success(c, http.StatusOK, "offers retrieved", offers)
}

// GetOffersByTypeAndAmount retrieves offers by type and amount
func (h *OfferHandler) GetOffersByTypeAndAmount(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	offerType := c.Query("type")
	amountStr := c.Query("amount")

	if offerType == "" || amountStr == "" {
		response.Error(c, http.StatusBadRequest, "type and amount are required", nil)
		return
	}

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid amount", err)
		return
	}

	offers, err := h.offerService.GetOffersByTypeAndAmount(c.Request.Context(), agentID, offer.OfferType(offerType), amount)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get offers", err)
		return
	}

	response.Success(c, http.StatusOK, "offers retrieved", offers)
}

// GetOfferByPrice retrieves a single offer by price
func (h *OfferHandler) GetOfferByPrice(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	priceStr := c.Query("price")
	if priceStr == "" {
		response.Error(c, http.StatusBadRequest, "price is required", nil)
		return
	}

	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid price", err)
		return
	}

	offer, err := h.offerService.GetOfferByPrice(c.Request.Context(), agentID, price)
	if err != nil {
		if err == xerrors.ErrNotFound {
			response.Error(c, http.StatusNotFound, "no offer found with this price", err)
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to get offer", err)
		return
	}

	response.Success(c, http.StatusOK, "offer retrieved", offer)
}

// GetOfferByPriceAndType retrieves a single offer by price and type
func (h *OfferHandler) GetOfferByPriceAndType(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	priceStr := c.Query("price")
	offerType := c.Query("type")

	if priceStr == "" || offerType == "" {
		response.Error(c, http.StatusBadRequest, "price and type are required", nil)
		return
	}

	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid price", err)
		return
	}

	offer, err := h.offerService.GetOfferByPriceAndType(c.Request.Context(), agentID, price, offer.OfferType(offerType))
	if err != nil {
		if err == xerrors.ErrNotFound {
			response.Error(c, http.StatusNotFound, "no offer found with this price and type", err)
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to get offer", err)
		return
	}

	response.Success(c, http.StatusOK, "offer retrieved", offer)
}