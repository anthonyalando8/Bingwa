// internal/handlers/subscription/plan_handler.go
package subscription

import (
	"net/http"
	"strconv"

	"bingwa-service/internal/domain/subscription"
	//"bingwa-service/internal/middleware"
	"bingwa-service/internal/pkg/response"
	service "bingwa-service/internal/service/subscription_plans"

	"github.com/gin-gonic/gin"
)

type PlanHandler struct {
	planService *service.PlanService
}

func NewPlanHandler(planService *service.PlanService) *PlanHandler {
	return &PlanHandler{
		planService: planService,
	}
}

// ========== Public/Agent Endpoints ==========

// ListPlans retrieves subscription plans with filters
func (h *PlanHandler) ListPlans(c *gin.Context) {
	var filters subscription.PlanListFilters
	if err := c.ShouldBindQuery(&filters); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid query parameters", err)
		return
	}

	result, err := h.planService.ListPlans(c.Request.Context(), &filters)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list plans", err)
		return
	}

	response.Success(c, http.StatusOK, "plans retrieved", result)
}

// ListPublicPlans retrieves only public (subscribable) plans
func (h *PlanHandler) ListPublicPlans(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	result, err := h.planService.ListPublicPlans(c.Request.Context(), page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list public plans", err)
		return
	}

	response.Success(c, http.StatusOK, "public plans retrieved", result)
}

// GetPlan retrieves a single subscription plan by ID
func (h *PlanHandler) GetPlan(c *gin.Context) {
	planIDStr := c.Param("id")
	planID, err := strconv.ParseInt(planIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid plan ID", err)
		return
	}

	plan, err := h.planService.GetPlan(c.Request.Context(), planID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "plan not found", err)
		return
	}

	response.Success(c, http.StatusOK, "plan retrieved", plan)
}

// GetPlanByCode retrieves a subscription plan by plan code
func (h *PlanHandler) GetPlanByCode(c *gin.Context) {
	planCode := c.Param("code")
	if planCode == "" {
		response.Error(c, http.StatusBadRequest, "plan code is required", nil)
		return
	}

	plan, err := h.planService.GetPlanByCode(c.Request.Context(), planCode)
	if err != nil {
		response.Error(c, http.StatusNotFound, "plan not found", err)
		return
	}

	response.Success(c, http.StatusOK, "plan retrieved", plan)
}

// ComparePlans compares two subscription plans
func (h *PlanHandler) ComparePlans(c *gin.Context) {
	plan1IDStr := c.Query("plan1")
	plan2IDStr := c.Query("plan2")

	if plan1IDStr == "" || plan2IDStr == "" {
		response.Error(c, http.StatusBadRequest, "both plan IDs are required", nil)
		return
	}

	plan1ID, err := strconv.ParseInt(plan1IDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid plan1 ID", err)
		return
	}

	plan2ID, err := strconv.ParseInt(plan2IDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid plan2 ID", err)
		return
	}

	comparison, err := h.planService.ComparePlans(c.Request.Context(), plan1ID, plan2ID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to compare plans", err)
		return
	}

	response.Success(c, http.StatusOK, "plans compared", comparison)
}

// ========== Admin Only Endpoints ==========

// CreatePlan creates a new subscription plan (admin only)
func (h *PlanHandler) CreatePlan(c *gin.Context) {
	var req subscription.CreatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	plan, err := h.planService.CreatePlan(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "failed to create plan", err)
		return
	}

	response.Success(c, http.StatusCreated, "plan created successfully", plan)
}

// UpdatePlan updates a subscription plan (admin only)
func (h *PlanHandler) UpdatePlan(c *gin.Context) {
	planIDStr := c.Param("id")
	planID, err := strconv.ParseInt(planIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid plan ID", err)
		return
	}

	var req subscription.UpdatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	plan, err := h.planService.UpdatePlan(c.Request.Context(), planID, &req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "failed to update plan", err)
		return
	}

	response.Success(c, http.StatusOK, "plan updated successfully", plan)
}

// ActivatePlan activates a subscription plan (admin only)
func (h *PlanHandler) ActivatePlan(c *gin.Context) {
	planIDStr := c.Param("id")
	planID, err := strconv.ParseInt(planIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid plan ID", err)
		return
	}

	if err := h.planService.ActivatePlan(c.Request.Context(), planID); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to activate plan", err)
		return
	}

	response.Success(c, http.StatusOK, "plan activated successfully", nil)
}

// DeactivatePlan deactivates a subscription plan (admin only)
func (h *PlanHandler) DeactivatePlan(c *gin.Context) {
	planIDStr := c.Param("id")
	planID, err := strconv.ParseInt(planIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid plan ID", err)
		return
	}

	if err := h.planService.DeactivatePlan(c.Request.Context(), planID); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to deactivate plan", err)
		return
	}

	response.Success(c, http.StatusOK, "plan deactivated successfully", nil)
}

// DeletePlan deletes a subscription plan (admin only)
func (h *PlanHandler) DeletePlan(c *gin.Context) {
	planIDStr := c.Param("id")
	planID, err := strconv.ParseInt(planIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid plan ID", err)
		return
	}

	if err := h.planService.DeletePlan(c.Request.Context(), planID); err != nil {
		response.Error(c, http.StatusBadRequest, "failed to delete plan", err)
		return
	}

	response.Success(c, http.StatusOK, "plan deleted successfully", nil)
}

// GetPlanStats retrieves subscription plan statistics (admin only)
func (h *PlanHandler) GetPlanStats(c *gin.Context) {
	stats, err := h.planService.GetStats(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get stats", err)
		return
	}

	response.Success(c, http.StatusOK, "stats retrieved", stats)
}