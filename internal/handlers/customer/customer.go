// internal/handlers/customer/customer_handler.go
package customer

import (
	"fmt"
	"net/http"
	"strconv"

	"bingwa-service/internal/domain/customer"
	"bingwa-service/internal/middleware"
	"bingwa-service/internal/pkg/response"
	service "bingwa-service/internal/service/customer"

	"github.com/gin-gonic/gin"
)

type CustomerHandler struct {
	customerService *service.CustomerService
}

func NewCustomerHandler(customerService *service.CustomerService) *CustomerHandler {
	return &CustomerHandler{
		customerService: customerService,
	}
}

// ========== Agent/Admin Endpoints ==========

// CreateCustomer creates a new customer
func (h *CustomerHandler) CreateCustomer(c *gin.Context) {
	// Get agent ID from authenticated user or query param (for admin)
	agentID, err := h.getAgentID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid agent ID", err)
		return
	}

	var req customer.CreateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	result, err := h.customerService.CreateCustomer(c.Request.Context(), agentID, &req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "failed to create customer", err)
		return
	}

	response.Success(c, http.StatusCreated, "customer created successfully", result)
}

// GetCustomer retrieves a customer by ID
func (h *CustomerHandler) GetCustomer(c *gin.Context) {
	agentID, err := h.getAgentID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid agent ID", err)
		return
	}

	customerIDStr := c.Param("id")
	customerID, err := strconv.ParseInt(customerIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid customer ID", err)
		return
	}

	result, err := h.customerService.GetCustomer(c.Request.Context(), agentID, customerID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "customer not found", err)
		return
	}

	response.Success(c, http.StatusOK, "customer retrieved", result)
}

// GetCustomerByReference retrieves a customer by reference
func (h *CustomerHandler) GetCustomerByReference(c *gin.Context) {
	agentID, err := h.getAgentID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid agent ID", err)
		return
	}

	reference := c.Param("reference")
	if reference == "" {
		response.Error(c, http.StatusBadRequest, "customer reference is required", nil)
		return
	}

	result, err := h.customerService.GetCustomerByReference(c.Request.Context(), agentID, reference)
	if err != nil {
		response.Error(c, http.StatusNotFound, "customer not found", err)
		return
	}

	response.Success(c, http.StatusOK, "customer retrieved", result)
}

// GetCustomerByPhone retrieves a customer by phone number
func (h *CustomerHandler) GetCustomerByPhone(c *gin.Context) {
	agentID, err := h.getAgentID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid agent ID", err)
		return
	}

	phone := c.Query("phone")
	if phone == "" {
		response.Error(c, http.StatusBadRequest, "phone number is required", nil)
		return
	}

	result, err := h.customerService.GetCustomerByPhone(c.Request.Context(), agentID, phone)
	if err != nil {
		response.Error(c, http.StatusNotFound, "customer not found", err)
		return
	}

	response.Success(c, http.StatusOK, "customer retrieved", result)
}

// ListCustomers retrieves customers with filters
func (h *CustomerHandler) ListCustomers(c *gin.Context) {
	agentID, err := h.getAgentID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid agent ID", err)
		return
	}

	var filters customer.CustomerListFilters
	if err := c.ShouldBindQuery(&filters); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid query parameters", err)
		return
	}

	result, err := h.customerService.ListCustomers(c.Request.Context(), agentID, &filters)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list customers", err)
		return
	}

	response.Success(c, http.StatusOK, "customers retrieved", result)
}

// UpdateCustomer updates a customer
func (h *CustomerHandler) UpdateCustomer(c *gin.Context) {
	agentID, err := h.getAgentID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid agent ID", err)
		return
	}

	customerIDStr := c.Param("id")
	customerID, err := strconv.ParseInt(customerIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid customer ID", err)
		return
	}

	var req customer.UpdateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	result, err := h.customerService.UpdateCustomer(c.Request.Context(), agentID, customerID, &req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "failed to update customer", err)
		return
	}

	response.Success(c, http.StatusOK, "customer updated successfully", result)
}

// ActivateCustomer activates a customer
func (h *CustomerHandler) ActivateCustomer(c *gin.Context) {
	agentID, err := h.getAgentID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid agent ID", err)
		return
	}

	customerIDStr := c.Param("id")
	customerID, err := strconv.ParseInt(customerIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid customer ID", err)
		return
	}

	if err := h.customerService.ActivateCustomer(c.Request.Context(), agentID, customerID); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to activate customer", err)
		return
	}

	response.Success(c, http.StatusOK, "customer activated successfully", nil)
}

// DeactivateCustomer deactivates a customer
func (h *CustomerHandler) DeactivateCustomer(c *gin.Context) {
	agentID, err := h.getAgentID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid agent ID", err)
		return
	}

	customerIDStr := c.Param("id")
	customerID, err := strconv.ParseInt(customerIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid customer ID", err)
		return
	}

	if err := h.customerService.DeactivateCustomer(c.Request.Context(), agentID, customerID); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to deactivate customer", err)
		return
	}

	response.Success(c, http.StatusOK, "customer deactivated successfully", nil)
}

// VerifyCustomer marks a customer as verified
func (h *CustomerHandler) VerifyCustomer(c *gin.Context) {
	agentID, err := h.getAgentID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid agent ID", err)
		return
	}

	customerIDStr := c.Param("id")
	customerID, err := strconv.ParseInt(customerIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid customer ID", err)
		return
	}

	if err := h.customerService.VerifyCustomer(c.Request.Context(), agentID, customerID); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to verify customer", err)
		return
	}

	response.Success(c, http.StatusOK, "customer verified successfully", nil)
}

// DeleteCustomer soft deletes a customer
func (h *CustomerHandler) DeleteCustomer(c *gin.Context) {
	agentID, err := h.getAgentID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid agent ID", err)
		return
	}

	customerIDStr := c.Param("id")
	customerID, err := strconv.ParseInt(customerIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid customer ID", err)
		return
	}

	if err := h.customerService.DeleteCustomer(c.Request.Context(), agentID, customerID); err != nil {
		response.Error(c, http.StatusBadRequest, "failed to delete customer", err)
		return
	}

	response.Success(c, http.StatusOK, "customer deleted successfully", nil)
}

// GetCustomerStats retrieves customer statistics
func (h *CustomerHandler) GetCustomerStats(c *gin.Context) {
	agentID, err := h.getAgentID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid agent ID", err)
		return
	}

	stats, err := h.customerService.GetCustomerStats(c.Request.Context(), agentID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get customer stats", err)
		return
	}

	response.Success(c, http.StatusOK, "customer stats retrieved", stats)
}

// AddTag adds a tag to a customer
func (h *CustomerHandler) AddTag(c *gin.Context) {
	agentID, err := h.getAgentID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid agent ID", err)
		return
	}

	customerIDStr := c.Param("id")
	customerID, err := strconv.ParseInt(customerIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid customer ID", err)
		return
	}

	var req struct {
		Tag string `json:"tag" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	if err := h.customerService.AddTag(c.Request.Context(), agentID, customerID, req.Tag); err != nil {
		response.Error(c, http.StatusBadRequest, "failed to add tag", err)
		return
	}

	response.Success(c, http.StatusOK, "tag added successfully", nil)
}

// RemoveTag removes a tag from a customer
func (h *CustomerHandler) RemoveTag(c *gin.Context) {
	agentID, err := h.getAgentID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid agent ID", err)
		return
	}

	customerIDStr := c.Param("id")
	customerID, err := strconv.ParseInt(customerIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid customer ID", err)
		return
	}

	tag := c.Query("tag")
	if tag == "" {
		response.Error(c, http.StatusBadRequest, "tag is required", nil)
		return
	}

	if err := h.customerService.RemoveTag(c.Request.Context(), agentID, customerID, tag); err != nil {
		response.Error(c, http.StatusBadRequest, "failed to remove tag", err)
		return
	}

	response.Success(c, http.StatusOK, "tag removed successfully", nil)
}

// BulkImportCustomers imports multiple customers
func (h *CustomerHandler) BulkImportCustomers(c *gin.Context) {
	agentID, err := h.getAgentID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid agent ID", err)
		return
	}

	var req struct {
		Customers []customer.CreateCustomerRequest `json:"customers" binding:"required,min=1,max=100"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	createdIDs, errors := h.customerService.BulkImportCustomers(c.Request.Context(), agentID, req.Customers)

	// Count successes and failures
	successCount := 0
	failureCount := 0
	for _, err := range errors {
		if err == nil {
			successCount++
		} else {
			failureCount++
		}
	}

	response.Success(c, http.StatusCreated, "bulk import completed", gin.H{
		"total":         len(req.Customers),
		"success_count": successCount,
		"failure_count": failureCount,
		"created_ids":   createdIDs,
		"errors":        errors,
	})
}

// SearchCustomers searches customers
func (h *CustomerHandler) SearchCustomers(c *gin.Context) {
	agentID, err := h.getAgentID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid agent ID", err)
		return
	}

	query := c.Query("q")
	if query == "" {
		response.Error(c, http.StatusBadRequest, "search query is required", nil)
		return
	}

	results, err := h.customerService.SearchCustomers(c.Request.Context(), agentID, query)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to search customers", err)
		return
	}

	response.Success(c, http.StatusOK, "search results", gin.H{
		"customers": results,
		"count":     len(results),
	})
}

// ========== Helper Methods ==========

// getAgentID extracts agent ID from context or query params
// For agents: uses authenticated identity_id
// For admins: requires agent_id query parameter
func (h *CustomerHandler) getAgentID(c *gin.Context) (int64, error) {
	// Check if user is admin
	isAdmin := middleware.HasRole(c, "admin") || middleware.HasRole(c, "super_admin")

	if isAdmin {
		// Admin must provide agent_id as query parameter
		agentIDStr := c.Query("agent_id")
		if agentIDStr == "" {
			return 0, fmt.Errorf("agent_id query parameter is required for admin access")
		}

		agentID, err := strconv.ParseInt(agentIDStr, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid agent_id format")
		}

		return agentID, nil
	}

	// For regular agents, use their own identity_id
	identityID, exists := middleware.GetIdentityID(c)
	if !exists {
		return 0, fmt.Errorf("identity_id not found in context")
	}

	return identityID, nil
}