// internal/handlers/transaction/transaction_handler.go
package transaction

import (
	"fmt"
	"net/http"
	"strconv"

	"bingwa-service/internal/domain/transaction"
	"bingwa-service/internal/middleware"
	"bingwa-service/internal/pkg/response"
	service "bingwa-service/internal/service/transaction"

	"github.com/gin-gonic/gin"
)

type TransactionHandler struct {
	transactionService *service.TransactionService
}

func NewTransactionHandler(transactionService *service.TransactionService) *TransactionHandler {
	return &TransactionHandler{
		transactionService: transactionService,
	}
}

// ========== Offer Request Endpoints ==========

// CreateOfferRequest creates a new offer request
func (h *TransactionHandler) CreateOfferRequest(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	var req transaction.CreateOfferRequestInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	offerRequest, redemption, err := h.transactionService.CreateOfferRequest(c.Request.Context(), agentID, &req)
	if err != nil {
		// Check if it's a subscription error
		if err.Error() == "No active subscription available" {
			response.Error(c, http.StatusPaymentRequired, "subscription required", err)
			return
		}
		response.Error(c, http.StatusBadRequest, "failed to create offer request", err)
		return
	}

	response.Success(c, http.StatusCreated, "offer request created successfully", gin.H{
		"offer_request": offerRequest,
		"redemption":    redemption,
	})
}

// GetOfferRequest retrieves an offer request by ID
func (h *TransactionHandler) GetOfferRequest(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	requestIDStr := c.Param("id")
	requestID, err := strconv.ParseInt(requestIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request ID", err)
		return
	}

	result, err := h.transactionService.GetOfferRequest(c.Request.Context(), agentID, requestID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "offer request not found", err)
		return
	}

	response.Success(c, http.StatusOK, "offer request retrieved", result)
}

// ListOfferRequests retrieves offer requests with filters
func (h *TransactionHandler) ListOfferRequests(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	var filters transaction.OfferRequestListFilters
	if err := c.ShouldBindQuery(&filters); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid query parameters", err)
		return
	}

	result, err := h.transactionService.ListOfferRequests(c.Request.Context(), agentID, &filters)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list offer requests", err)
		return
	}

	response.Success(c, http.StatusOK, "offer requests retrieved", result)
}

// GetPendingRequests retrieves pending offer requests for processing
func (h *TransactionHandler) GetPendingRequests(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 50
	}

	requests, err := h.transactionService.GetPendingRequests(c.Request.Context(), agentID, limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get pending requests", err)
		return
	}

	response.Success(c, http.StatusOK, "pending requests retrieved", gin.H{
		"requests": requests,
		"count":    len(requests),
	})
}

// GetFailedRequests retrieves failed offer requests for retry
func (h *TransactionHandler) GetFailedRequests(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 50
	}

	requests, err := h.transactionService.GetFailedRequests(c.Request.Context(), agentID, limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get failed requests", err)
		return
	}

	response.Success(c, http.StatusOK, "failed requests retrieved", gin.H{
		"requests": requests,
		"count":    len(requests),
	})
}

// GetProcessingRequests retrieves processing offer requests
func (h *TransactionHandler) GetProcessingRequests(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 50
	}

	requests, err := h.transactionService.GetProcessingRequests(c.Request.Context(), agentID, limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get processing requests", err)
		return
	}

	response.Success(c, http.StatusOK, "processing requests retrieved", gin.H{
		"requests": requests,
		"count":    len(requests),
	})
}

// UpdateOfferRequestStatus updates the status of an offer request
func (h *TransactionHandler) UpdateOfferRequestStatus(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	requestIDStr := c.Param("id")
	requestID, err := strconv.ParseInt(requestIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request ID", err)
		return
	}

	var req struct {
		Status        transaction.TransactionStatus          `json:"status" binding:"required"`
		USSDResponse  *transaction.UpdateUSSDResponseInput   `json:"ussd_response"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	if err := h.transactionService.UpdateOfferRequestStatus(c.Request.Context(), agentID, requestID, req.Status, req.USSDResponse); err != nil {
		response.Error(c, http.StatusBadRequest, "failed to update status", err)
		return
	}

	response.Success(c, http.StatusOK, "offer request status updated successfully", nil)
}

// CompleteOfferRequest completes an offer request with USSD response
func (h *TransactionHandler) CompleteOfferRequest(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	requestIDStr := c.Param("id")
	requestID, err := strconv.ParseInt(requestIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request ID", err)
		return
	}

	var req transaction.UpdateUSSDResponseInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	// Default to success if not provided
	if req.Status == "" {
		req.Status = transaction.TransactionStatusSuccess
	}

	if err := h.transactionService.UpdateOfferRequestStatus(c.Request.Context(), agentID, requestID, req.Status, &req); err != nil {
		response.Error(c, http.StatusBadRequest, "failed to complete request", err)
		return
	}

	response.Success(c, http.StatusOK, "offer request completed successfully", nil)
}

// MarkAsProcessing marks a request as processing
func (h *TransactionHandler) MarkAsProcessing(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	requestIDStr := c.Param("id")
	requestID, err := strconv.ParseInt(requestIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request ID", err)
		return
	}

	if err := h.transactionService.UpdateOfferRequestStatus(c.Request.Context(), agentID, requestID, transaction.TransactionStatusProcessing, nil); err != nil {
		response.Error(c, http.StatusBadRequest, "failed to mark as processing", err)
		return
	}

	response.Success(c, http.StatusOK, "offer request marked as processing", nil)
}

// RetryFailedRequest retries a failed request
func (h *TransactionHandler) RetryFailedRequest(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	requestIDStr := c.Param("id")
	requestID, err := strconv.ParseInt(requestIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request ID", err)
		return
	}

	if err := h.transactionService.RetryFailedRequest(c.Request.Context(), agentID, requestID); err != nil {
		response.Error(c, http.StatusBadRequest, "failed to retry request", err)
		return
	}

	response.Success(c, http.StatusOK, "offer request queued for retry", nil)
}

// ========== Redemption Endpoints ==========

// GetOfferRedemption retrieves a redemption by ID
func (h *TransactionHandler) GetOfferRedemption(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	redemptionIDStr := c.Param("id")
	redemptionID, err := strconv.ParseInt(redemptionIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid redemption ID", err)
		return
	}

	result, err := h.transactionService.GetOfferRedemption(c.Request.Context(), agentID, redemptionID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "redemption not found", err)
		return
	}

	response.Success(c, http.StatusOK, "redemption retrieved", result)
}

// ListOfferRedemptions retrieves redemptions with filters
func (h *TransactionHandler) ListOfferRedemptions(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	var filters transaction.RedemptionListFilters
	if err := c.ShouldBindQuery(&filters); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid query parameters", err)
		return
	}

	result, err := h.transactionService.ListOfferRedemptions(c.Request.Context(), agentID, &filters)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list redemptions", err)
		return
	}

	response.Success(c, http.StatusOK, "redemptions retrieved", result)
}

// ========== Statistics ==========

// GetTransactionStats retrieves transaction statistics
func (h *TransactionHandler) GetTransactionStats(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	stats, err := h.transactionService.GetTransactionStats(c.Request.Context(), agentID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get statistics", err)
		return
	}

	response.Success(c, http.StatusOK, "transaction statistics retrieved", stats)
}

// GetRequestsByStatus retrieves counts by status
func (h *TransactionHandler) GetRequestsByStatus(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	// Get counts for each status
	pendingCount := 0
	processingCount := 0
	successCount := 0
	failedCount := 0

	// Get pending
	pending, _ := h.transactionService.GetPendingRequests(c.Request.Context(), agentID, 1000)
	pendingCount = len(pending)

	// Get processing
	processing, _ := h.transactionService.GetProcessingRequests(c.Request.Context(), agentID, 1000)
	processingCount = len(processing)

	// Get failed
	failed, _ := h.transactionService.GetFailedRequests(c.Request.Context(), agentID, 1000)
	failedCount = len(failed)

	// Get stats for success count
	stats, _ := h.transactionService.GetTransactionStats(c.Request.Context(), agentID)
	if stats != nil {
		successCount = int(stats.SuccessfulRequests)
	}

	response.Success(c, http.StatusOK, "request counts by status", gin.H{
		"pending":    pendingCount,
		"processing": processingCount,
		"success":    successCount,
		"failed":     failedCount,
		"total":      pendingCount + processingCount + successCount + failedCount,
	})
}

// ========== Batch Operations ==========

// GetBatchPendingForDevice retrieves pending requests for a specific device
func (h *TransactionHandler) GetBatchPendingForDevice(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 50 {
		limit = 10
	}

	requests, err := h.transactionService.GetPendingRequests(c.Request.Context(), agentID, limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get pending requests", err)
		return
	}

	// Format for mobile client with USSD codes ready
	type BatchRequest struct {
		RequestID   int64  `json:"request_id"`
		RedemptionID int64 `json:"redemption_id"`
		USSDCode    string `json:"ussd_code"`
		CustomerPhone string `json:"customer_phone"`
		Amount      float64 `json:"amount"`
		OfferName   string  `json:"offer_name"`
	}

	batchRequests := []BatchRequest{}
	for _, req := range requests {
		// Get redemption to get USSD code
		// This is simplified - in production you'd optimize this
		batchRequests = append(batchRequests, BatchRequest{
			RequestID:     req.ID,
			CustomerPhone: req.CustomerPhone,
			Amount:        req.AmountPaid,
			// USSDCode and other fields would be populated from redemption
		})
	}

	response.Success(c, http.StatusOK, "batch pending requests retrieved", gin.H{
		"requests": batchRequests,
		"count":    len(batchRequests),
		"limit":    limit,
	})
}

// BatchUpdateRequests updates multiple requests at once
func (h *TransactionHandler) BatchUpdateRequests(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	var req struct {
		Updates []struct {
			RequestID    int64                              `json:"request_id" binding:"required"`
			Status       transaction.TransactionStatus      `json:"status" binding:"required"`
			USSDResponse *transaction.UpdateUSSDResponseInput `json:"ussd_response"`
		} `json:"updates" binding:"required,min=1,max=50"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	successCount := 0
	failureCount := 0
	errors := []string{}

	for _, update := range req.Updates {
		if err := h.transactionService.UpdateOfferRequestStatus(c.Request.Context(), agentID, update.RequestID, update.Status, update.USSDResponse); err != nil {
			failureCount++
			errors = append(errors, fmt.Sprintf("Request %d: %s", update.RequestID, err.Error()))
		} else {
			successCount++
		}
	}

	response.Success(c, http.StatusOK, "batch update completed", gin.H{
		"total":         len(req.Updates),
		"success_count": successCount,
		"failure_count": failureCount,
		"errors":        errors,
	})
}