// internal/handlers/schedule/schedule_handler.go
package schedule

import (
	"net/http"
	"strconv"

	"bingwa-service/internal/domain/schedule"
	"bingwa-service/internal/middleware"
	"bingwa-service/internal/pkg/response"
	service "bingwa-service/internal/service/schedule"

	"github.com/gin-gonic/gin"
)

type ScheduleHandler struct {
	scheduleService *service.ScheduleService
}

func NewScheduleHandler(scheduleService *service.ScheduleService) *ScheduleHandler {
	return &ScheduleHandler{
		scheduleService: scheduleService,
	}
}

// ========== Scheduled Offer Endpoints ==========

// CreateScheduledOffer creates a new scheduled offer
func (h *ScheduleHandler) CreateScheduledOffer(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	var req schedule.CreateScheduledOfferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	result, err := h.scheduleService.CreateScheduledOffer(c.Request.Context(), agentID, &req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "failed to create scheduled offer", err)
		return
	}

	response.Success(c, http.StatusCreated, "scheduled offer created successfully", result)
}

// GetScheduledOffer retrieves a scheduled offer by ID
func (h *ScheduleHandler) GetScheduledOffer(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	scheduleIDStr := c.Param("id")
	scheduleID, err := strconv.ParseInt(scheduleIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid schedule ID", err)
		return
	}

	result, err := h.scheduleService.GetScheduledOffer(c.Request.Context(), agentID, scheduleID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "scheduled offer not found", err)
		return
	}

	response.Success(c, http.StatusOK, "scheduled offer retrieved", result)
}

// ListScheduledOffers retrieves scheduled offers with filters
func (h *ScheduleHandler) ListScheduledOffers(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	var filters schedule.ScheduledOfferListFilters
	if err := c.ShouldBindQuery(&filters); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid query parameters", err)
		return
	}

	result, err := h.scheduleService.ListScheduledOffers(c.Request.Context(), agentID, &filters)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list scheduled offers", err)
		return
	}

	response.Success(c, http.StatusOK, "scheduled offers retrieved", result)
}

// GetDueSchedules retrieves schedules due for execution
func (h *ScheduleHandler) GetDueSchedules(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	schedules, err := h.scheduleService.GetDueSchedules(c.Request.Context(), agentID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get due schedules", err)
		return
	}

	response.Success(c, http.StatusOK, "due schedules retrieved", gin.H{
		"schedules": schedules,
		"count":     len(schedules),
	})
}

// UpdateScheduledOffer updates a scheduled offer
func (h *ScheduleHandler) UpdateScheduledOffer(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	scheduleIDStr := c.Param("id")
	scheduleID, err := strconv.ParseInt(scheduleIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid schedule ID", err)
		return
	}

	var req schedule.UpdateScheduledOfferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	result, err := h.scheduleService.UpdateScheduledOffer(c.Request.Context(), agentID, scheduleID, &req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "failed to update scheduled offer", err)
		return
	}

	response.Success(c, http.StatusOK, "scheduled offer updated successfully", result)
}

// PauseScheduledOffer pauses a scheduled offer
func (h *ScheduleHandler) PauseScheduledOffer(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	scheduleIDStr := c.Param("id")
	scheduleID, err := strconv.ParseInt(scheduleIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid schedule ID", err)
		return
	}

	if err := h.scheduleService.PauseScheduledOffer(c.Request.Context(), agentID, scheduleID); err != nil {
		response.Error(c, http.StatusBadRequest, "failed to pause scheduled offer", err)
		return
	}

	response.Success(c, http.StatusOK, "scheduled offer paused successfully", nil)
}

// ResumeScheduledOffer resumes a paused scheduled offer
func (h *ScheduleHandler) ResumeScheduledOffer(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	scheduleIDStr := c.Param("id")
	scheduleID, err := strconv.ParseInt(scheduleIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid schedule ID", err)
		return
	}

	if err := h.scheduleService.ResumeScheduledOffer(c.Request.Context(), agentID, scheduleID); err != nil {
		response.Error(c, http.StatusBadRequest, "failed to resume scheduled offer", err)
		return
	}

	response.Success(c, http.StatusOK, "scheduled offer resumed successfully", nil)
}

// CancelScheduledOffer cancels a scheduled offer
func (h *ScheduleHandler) CancelScheduledOffer(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	scheduleIDStr := c.Param("id")
	scheduleID, err := strconv.ParseInt(scheduleIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid schedule ID", err)
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	if err := h.scheduleService.CancelScheduledOffer(c.Request.Context(), agentID, scheduleID, req.Reason); err != nil {
		response.Error(c, http.StatusBadRequest, "failed to cancel scheduled offer", err)
		return
	}

	response.Success(c, http.StatusOK, "scheduled offer cancelled successfully", nil)
}

// ExecuteScheduledOffer executes a scheduled offer (mobile app)
func (h *ScheduleHandler) ExecuteScheduledOffer(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	scheduleIDStr := c.Param("id")
	scheduleID, err := strconv.ParseInt(scheduleIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid schedule ID", err)
		return
	}

	var req schedule.ExecuteScheduledOfferInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	redemption, history, err := h.scheduleService.ExecuteScheduledOffer(c.Request.Context(), agentID, scheduleID, &req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "failed to execute scheduled offer", err)
		return
	}

	response.Success(c, http.StatusOK, "scheduled offer executed successfully", gin.H{
		"redemption": redemption,
		"history":    history,
	})
}

// ========== History Endpoints ==========

// GetScheduleHistory retrieves execution history for a schedule
func (h *ScheduleHandler) GetScheduleHistory(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	scheduleIDStr := c.Param("id")
	scheduleID, err := strconv.ParseInt(scheduleIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid schedule ID", err)
		return
	}

	var filters schedule.ScheduleHistoryListFilters
	if err := c.ShouldBindQuery(&filters); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid query parameters", err)
		return
	}

	result, err := h.scheduleService.GetScheduleHistory(c.Request.Context(), agentID, scheduleID, &filters)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get schedule history", err)
		return
	}

	response.Success(c, http.StatusOK, "schedule history retrieved", result)
}

// ========== Statistics ==========

// GetScheduleStats retrieves schedule statistics
func (h *ScheduleHandler) GetScheduleStats(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	stats, err := h.scheduleService.GetScheduleStats(c.Request.Context(), agentID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get schedule stats", err)
		return
	}

	response.Success(c, http.StatusOK, "schedule statistics retrieved", stats)
}

// ========== Batch Operations for Mobile App ==========

// GetBatchDueSchedules retrieves batch of due schedules for mobile execution
func (h *ScheduleHandler) GetBatchDueSchedules(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	schedules, err := h.scheduleService.GetDueSchedules(c.Request.Context(), agentID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get due schedules", err)
		return
	}

	// Format for mobile client
	type BatchSchedule struct {
		ScheduleID      int64  `json:"schedule_id"`
		OfferID         int64  `json:"offer_id"`
		OfferName       string `json:"offer_name"`
		CustomerPhone   string `json:"customer_phone"`
		USSDCode        string `json:"ussd_code"`
		RenewalNumber   int    `json:"renewal_number"`
		NextRenewalDate string `json:"next_renewal_date"`
	}

	batchSchedules := []BatchSchedule{}
	for _, sched := range schedules {
		batch := BatchSchedule{
			ScheduleID:    sched.ID,
			OfferID:       sched.OfferID,
			CustomerPhone: sched.CustomerPhone,
			RenewalNumber: sched.RenewalCount + 1,
		}
		
		if sched.NextRenewalDate.Valid {
			batch.NextRenewalDate = sched.NextRenewalDate.Time.Format("2006-01-02 15:04:05")
		}
		
		batchSchedules = append(batchSchedules, batch)
	}

	response.Success(c, http.StatusOK, "batch due schedules retrieved", gin.H{
		"schedules": batchSchedules,
		"count":     len(batchSchedules),
	})
}

// BatchExecuteSchedules executes multiple schedules at once
func (h *ScheduleHandler) BatchExecuteSchedules(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	var req struct {
		Executions []struct {
			ScheduleID         int64  `json:"schedule_id" binding:"required"`
			USSDResponse       string `json:"ussd_response"`
			USSDSessionID      string `json:"ussd_session_id"`
			USSDProcessingTime int32  `json:"ussd_processing_time"`
			Status             string `json:"status" binding:"required"`
			FailureReason      string `json:"failure_reason"`
		} `json:"executions" binding:"required,min=1,max=50"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	successCount := 0
	failureCount := 0
	results := []gin.H{}

	for _, exec := range req.Executions {
		input := &schedule.ExecuteScheduledOfferInput{
			USSDResponse:       exec.USSDResponse,
			USSDSessionID:      exec.USSDSessionID,
			USSDProcessingTime: exec.USSDProcessingTime,
			Status:             exec.Status,
			FailureReason:      exec.FailureReason,
		}

		redemption, history, err := h.scheduleService.ExecuteScheduledOffer(c.Request.Context(), agentID, exec.ScheduleID, input)
		if err != nil {
			failureCount++
			results = append(results, gin.H{
				"schedule_id": exec.ScheduleID,
				"success":     false,
				"error":       err.Error(),
			})
		} else {
			successCount++
			results = append(results, gin.H{
				"schedule_id":    exec.ScheduleID,
				"success":        true,
				"redemption_id":  redemption.ID,
				"history_id":     history.ID,
			})
		}
	}

	response.Success(c, http.StatusOK, "batch execution completed", gin.H{
		"total":         len(req.Executions),
		"success_count": successCount,
		"failure_count": failureCount,
		"results":       results,
	})
}

// GetSchedulesByStatus retrieves schedules grouped by status
func (h *ScheduleHandler) GetSchedulesByStatus(c *gin.Context) {
	agentID := middleware.MustGetIdentityID(c)

	// Get counts for each status
	activeStatus := schedule.ScheduleStatusActive
	activeFilters := &schedule.ScheduledOfferListFilters{Status: &activeStatus, Page: 1, PageSize: 1}
	activeResult, _ := h.scheduleService.ListScheduledOffers(c.Request.Context(), agentID, activeFilters)
	activeCount := int64(0)
	if activeResult != nil {
		activeCount = activeResult.Total
	}

	pausedStatus := schedule.ScheduleStatusPaused
	pausedFilters := &schedule.ScheduledOfferListFilters{Status: &pausedStatus, Page: 1, PageSize: 1}
	pausedResult, _ := h.scheduleService.ListScheduledOffers(c.Request.Context(), agentID, pausedFilters)
	pausedCount := int64(0)
	if pausedResult != nil {
		pausedCount = pausedResult.Total
	}

	cancelledStatus := schedule.ScheduleStatusCancelled
	cancelledFilters := &schedule.ScheduledOfferListFilters{Status: &cancelledStatus, Page: 1, PageSize: 1}
	cancelledResult, _ := h.scheduleService.ListScheduledOffers(c.Request.Context(), agentID, cancelledFilters)
	cancelledCount := int64(0)
	if cancelledResult != nil {
		cancelledCount = cancelledResult.Total
	}

	completedStatus := schedule.ScheduleStatusCompleted
	completedFilters := &schedule.ScheduledOfferListFilters{Status: &completedStatus, Page: 1, PageSize: 1}
	completedResult, _ := h.scheduleService.ListScheduledOffers(c.Request.Context(), agentID, completedFilters)
	completedCount := int64(0)
	if completedResult != nil {
		completedCount = completedResult.Total
	}

	response.Success(c, http.StatusOK, "schedules by status retrieved", gin.H{
		"active":    activeCount,
		"paused":    pausedCount,
		"cancelled": cancelledCount,
		"completed": completedCount,
		"total":     activeCount + pausedCount + cancelledCount + completedCount,
	})
}