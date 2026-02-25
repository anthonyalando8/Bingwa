// internal/handlers/subscription/agent_subscription_handler.go
package subscription

import (
	"net/http"
	"strconv"

	"bingwa-service/internal/domain/subscription"
	"bingwa-service/internal/middleware"
	"bingwa-service/internal/pkg/response"
	service "bingwa-service/internal/service/subscription"

	"github.com/gin-gonic/gin"
)

type AgentSubscriptionHandler struct {
	subscriptionService *service.SubscriptionService
}

func NewAgentSubscriptionHandler(subscriptionService *service.SubscriptionService) *AgentSubscriptionHandler {
	return &AgentSubscriptionHandler{
		subscriptionService: subscriptionService,
	}
}

// ========== Agent Endpoints ==========

// CreateSubscription creates a new subscription (from mobile USSD payment)
func (h *AgentSubscriptionHandler) CreateSubscription(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	var req subscription.CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	result, err := h.subscriptionService.CreateSubscription(c.Request.Context(), agentID, &req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "failed to create subscription", err)
		return
	}

	response.Success(c, http.StatusCreated, "subscription created successfully", result)
}

// RenewSubscription renews an existing subscription (from mobile USSD payment)
func (h *AgentSubscriptionHandler) RenewSubscription(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	var req subscription.RenewSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	result, err := h.subscriptionService.RenewSubscription(c.Request.Context(), agentID, &req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "failed to renew subscription", err)
		return
	}

	response.Success(c, http.StatusOK, "subscription renewed successfully", result)
}

// GetSubscription retrieves a subscription by ID
func (h *AgentSubscriptionHandler) GetSubscription(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	subscriptionIDStr := c.Param("id")
	subscriptionID, err := strconv.ParseInt(subscriptionIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid subscription ID", err)
		return
	}

	result, err := h.subscriptionService.GetSubscription(c.Request.Context(), agentID, subscriptionID, false)
	if err != nil {
		response.Error(c, http.StatusNotFound, "subscription not found", err)
		return
	}

	response.Success(c, http.StatusOK, "subscription retrieved", result)
}

// GetActiveSubscription retrieves the active subscription for the agent
func (h *AgentSubscriptionHandler) GetActiveSubscription(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	result, err := h.subscriptionService.GetActiveSubscription(c.Request.Context(), agentID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "no active subscription found", err)
		return
	}

	response.Success(c, http.StatusOK, "active subscription retrieved", result)
}

// ListSubscriptions retrieves subscriptions with filters
func (h *AgentSubscriptionHandler) ListSubscriptions(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	var filters subscription.SubscriptionListFilters
	if err := c.ShouldBindQuery(&filters); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid query parameters", err)
		return
	}

	result, err := h.subscriptionService.ListSubscriptions(c.Request.Context(), agentID, &filters, false)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list subscriptions", err)
		return
	}

	response.Success(c, http.StatusOK, "subscriptions retrieved", result)
}

// UpdateSubscription updates a subscription
func (h *AgentSubscriptionHandler) UpdateSubscription(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	subscriptionIDStr := c.Param("id")
	subscriptionID, err := strconv.ParseInt(subscriptionIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid subscription ID", err)
		return
	}

	var req subscription.UpdateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	result, err := h.subscriptionService.UpdateSubscription(c.Request.Context(), agentID, subscriptionID, &req, false)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "failed to update subscription", err)
		return
	}

	response.Success(c, http.StatusOK, "subscription updated successfully", result)
}

// CancelSubscription cancels a subscription
func (h *AgentSubscriptionHandler) CancelSubscription(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	subscriptionIDStr := c.Param("id")
	subscriptionID, err := strconv.ParseInt(subscriptionIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid subscription ID", err)
		return
	}

	var req subscription.CancelSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	if err := h.subscriptionService.CancelSubscription(c.Request.Context(), agentID, subscriptionID, &req, false); err != nil {
		response.Error(c, http.StatusBadRequest, "failed to cancel subscription", err)
		return
	}

	response.Success(c, http.StatusOK, "subscription cancelled successfully", nil)
}

// GetSubscriptionUsage retrieves usage information
func (h *AgentSubscriptionHandler) GetSubscriptionUsage(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	usage, err := h.subscriptionService.GetSubscriptionUsage(c.Request.Context(), agentID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "failed to get subscription usage", err)
		return
	}

	response.Success(c, http.StatusOK, "subscription usage retrieved", usage)
}

// CheckSubscriptionAccess checks if agent has active subscription access
func (h *AgentSubscriptionHandler) CheckSubscriptionAccess(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	hasAccess, err := h.subscriptionService.CheckSubscriptionAccess(c.Request.Context(), agentID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to check subscription access", err)
		return
	}

	response.Success(c, http.StatusOK, "subscription access checked", gin.H{
		"has_access": hasAccess,
	})
}

// GetSubscriptionStats retrieves subscription statistics
func (h *AgentSubscriptionHandler) GetSubscriptionStats(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	stats, err := h.subscriptionService.GetSubscriptionStats(c.Request.Context(), agentID, false)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get subscription stats", err)
		return
	}

	response.Success(c, http.StatusOK, "subscription statistics retrieved", stats)
}

// ========== Admin Endpoints ==========

// AdminGetSubscription retrieves any subscription by ID (admin only)
func (h *AgentSubscriptionHandler) AdminGetSubscription(c *gin.Context) {
	subscriptionIDStr := c.Param("id")
	subscriptionID, err := strconv.ParseInt(subscriptionIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid subscription ID", err)
		return
	}

	result, err := h.subscriptionService.GetSubscription(c.Request.Context(), 0, subscriptionID, true)
	if err != nil {
		response.Error(c, http.StatusNotFound, "subscription not found", err)
		return
	}

	response.Success(c, http.StatusOK, "subscription retrieved", result)
}

// AdminListSubscriptions lists all subscriptions (admin only)
func (h *AgentSubscriptionHandler) AdminListSubscriptions(c *gin.Context) {
	var filters subscription.SubscriptionListFilters
	if err := c.ShouldBindQuery(&filters); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid query parameters", err)
		return
	}

	// For admin, list all subscriptions (pass 0 as agentID)
	result, err := h.subscriptionService.ListSubscriptions(c.Request.Context(), 0, &filters, true)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list subscriptions", err)
		return
	}

	response.Success(c, http.StatusOK, "subscriptions retrieved", result)
}

// AdminDeactivateSubscription deactivates a subscription (admin only)
func (h *AgentSubscriptionHandler) AdminDeactivateSubscription(c *gin.Context) {
	subscriptionIDStr := c.Param("id")
	subscriptionID, err := strconv.ParseInt(subscriptionIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid subscription ID", err)
		return
	}

	if err := h.subscriptionService.DeactivateSubscription(c.Request.Context(), subscriptionID); err != nil {
		response.Error(c, http.StatusBadRequest, "failed to deactivate subscription", err)
		return
	}

	response.Success(c, http.StatusOK, "subscription deactivated successfully", nil)
}

// AdminSuspendSubscription suspends a subscription (admin only)
func (h *AgentSubscriptionHandler) AdminSuspendSubscription(c *gin.Context) {
	subscriptionIDStr := c.Param("id")
	subscriptionID, err := strconv.ParseInt(subscriptionIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid subscription ID", err)
		return
	}

	if err := h.subscriptionService.SuspendSubscription(c.Request.Context(), subscriptionID); err != nil {
		response.Error(c, http.StatusBadRequest, "failed to suspend subscription", err)
		return
	}

	response.Success(c, http.StatusOK, "subscription suspended successfully", nil)
}

// AdminReactivateSubscription reactivates a subscription (admin only)
func (h *AgentSubscriptionHandler) AdminReactivateSubscription(c *gin.Context) {
	subscriptionIDStr := c.Param("id")
	subscriptionID, err := strconv.ParseInt(subscriptionIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid subscription ID", err)
		return
	}

	if err := h.subscriptionService.ReactivateSubscription(c.Request.Context(), subscriptionID); err != nil {
		response.Error(c, http.StatusBadRequest, "failed to reactivate subscription", err)
		return
	}

	response.Success(c, http.StatusOK, "subscription reactivated successfully", nil)
}

// AdminCancelSubscription cancels a subscription (admin only)
func (h *AgentSubscriptionHandler) AdminCancelSubscription(c *gin.Context) {
	subscriptionIDStr := c.Param("id")
	subscriptionID, err := strconv.ParseInt(subscriptionIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid subscription ID", err)
		return
	}

	var req subscription.CancelSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	// Admin cancellation defaults to immediate
	if !req.CancelImmediately {
		req.CancelImmediately = true
	}

	if err := h.subscriptionService.CancelSubscription(c.Request.Context(), 0, subscriptionID, &req, true); err != nil {
		response.Error(c, http.StatusBadRequest, "failed to cancel subscription", err)
		return
	}

	response.Success(c, http.StatusOK, "subscription cancelled successfully", nil)
}

// AdminGetExpiringSubscriptions retrieves subscriptions expiring soon (admin only)
func (h *AgentSubscriptionHandler) AdminGetExpiringSubscriptions(c *gin.Context) {
	daysStr := c.DefaultQuery("days", "7")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days < 1 || days > 30 {
		days = 7
	}

	subscriptions, err := h.subscriptionService.GetExpiringSubscriptions(c.Request.Context(), days)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get expiring subscriptions", err)
		return
	}

	response.Success(c, http.StatusOK, "expiring subscriptions retrieved", gin.H{
		"subscriptions": subscriptions,
		"days":          days,
		"count":         len(subscriptions),
	})
}

// AdminGetSubscriptionStats retrieves overall subscription statistics (admin only)
func (h *AgentSubscriptionHandler) AdminGetSubscriptionStats(c *gin.Context) {
	// Get stats for all agents (pass 0 as agentID)
	stats, err := h.subscriptionService.GetSubscriptionStats(c.Request.Context(), 0, true)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get subscription stats", err)
		return
	}

	response.Success(c, http.StatusOK, "subscription statistics retrieved", stats)
}

// GetSubscriptionsByStatus retrieves subscriptions grouped by status
func (h *AgentSubscriptionHandler) GetSubscriptionsByStatus(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	// Get counts for each status
	activeStatus := subscription.SubscriptionStatusActive
	activeFilters := &subscription.SubscriptionListFilters{Status: &activeStatus, Page: 1, PageSize: 1}
	activeResult, _ := h.subscriptionService.ListSubscriptions(c.Request.Context(), agentID, activeFilters, false)
	activeCount := int64(0)
	if activeResult != nil {
		activeCount = activeResult.Total
	}

	expiredStatus := subscription.SubscriptionStatusExpired
	expiredFilters := &subscription.SubscriptionListFilters{Status: &expiredStatus, Page: 1, PageSize: 1}
	expiredResult, _ := h.subscriptionService.ListSubscriptions(c.Request.Context(), agentID, expiredFilters, false)
	expiredCount := int64(0)
	if expiredResult != nil {
		expiredCount = expiredResult.Total
	}

	cancelledStatus := subscription.SubscriptionStatusCancelled
	cancelledFilters := &subscription.SubscriptionListFilters{Status: &cancelledStatus, Page: 1, PageSize: 1}
	cancelledResult, _ := h.subscriptionService.ListSubscriptions(c.Request.Context(), agentID, cancelledFilters, false)
	cancelledCount := int64(0)
	if cancelledResult != nil {
		cancelledCount = cancelledResult.Total
	}

	response.Success(c, http.StatusOK, "subscriptions by status retrieved", gin.H{
		"active":    activeCount,
		"expired":   expiredCount,
		"cancelled": cancelledCount,
		"total":     activeCount + expiredCount + cancelledCount,
	})
}