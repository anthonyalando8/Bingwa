// internal/app/router.go
package app

import (
	authHandler "bingwa-service/internal/handlers/auth"
	campaignHandler "bingwa-service/internal/handlers/campaign"
	configHandler "bingwa-service/internal/handlers/config"
	customerHandler "bingwa-service/internal/handlers/customer"
	notifyHandler "bingwa-service/internal/handlers/notification"
	offerHandler "bingwa-service/internal/handlers/offer"
	scheduleHandler "bingwa-service/internal/handlers/schedule"
	agentSubscriptionHandler "bingwa-service/internal/handlers/subscription"
	planHandler "bingwa-service/internal/handlers/subscription_plans"
	transactionHandler "bingwa-service/internal/handlers/transaction"
	wsHandler "bingwa-service/internal/handlers/websocket"
	"bingwa-service/internal/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handlers struct {
	AuthHandler              *authHandler.AuthHandler
	NotifHandler             *notifyHandler.NotificationHandler
	PlanHandler              *planHandler.PlanHandler
	CustomerHandler          *customerHandler.CustomerHandler
	OfferHandler             *offerHandler.OfferHandler
	ConfigHandler            *configHandler.ConfigHandler
	CampaignHandler          *campaignHandler.CampaignHandler
	TransactionHandler       *transactionHandler.TransactionHandler
	ScheduleHandler          *scheduleHandler.ScheduleHandler
	AgentSubscriptionHandler *agentSubscriptionHandler.AgentSubscriptionHandler
	WSHandler                *wsHandler.WebSocketHandler
	AuthMiddleware           *middleware.AuthMiddleware
}

func SetupRouter(r *gin.Engine, logger *zap.Logger, h *Handlers) {
	api := r.Group("/api/v1")

	// ==================== Health Check ====================
	api.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "version": "1.0.0"})
	})

	// ==================== WebSocket ====================
	r.GET("/ws", h.WSHandler.HandleConnection)

	// ==================== Public Auth Routes ====================
	authPublic := api.Group("/auth")
	{
		authPublic.POST("/register", h.AuthHandler.Register)
		authPublic.POST("/login", h.AuthHandler.Login)
		authPublic.POST("/forgot-password", h.AuthHandler.ForgotPassword)
		authPublic.POST("/reset-password", h.AuthHandler.ResetPassword)
		authPublic.GET("/verify-email", h.AuthHandler.VerifyEmail)
	}

	// ==================== Authenticated Auth Routes ====================
	authProtected := api.Group("/auth")
	authProtected.Use(h.AuthMiddleware.Auth())
	{
		authProtected.POST("/logout", h.AuthHandler.Logout)
		authProtected.POST("/logout-all", h.AuthHandler.LogoutAll)
		authProtected.PUT("/change-password", h.AuthHandler.ChangePassword)
		authProtected.GET("/me", h.AuthHandler.GetMe)
		authProtected.PUT("/profile", h.AuthHandler.UpdateProfile)
		authProtected.POST("/resend-verification", h.AuthHandler.ResendVerificationEmail)
		authProtected.GET("/sessions", h.AuthHandler.GetActiveSessions)
		authProtected.DELETE("/sessions/:session_id", h.AuthHandler.RevokeSession)
	}

	// ==================== Notifications ====================
	notifications := api.Group("/notifications")
	notifications.Use(h.AuthMiddleware.Auth())
	{
		notifications.GET("", h.NotifHandler.GetNotifications)
		notifications.GET("/latest", h.NotifHandler.GetLatestNotifications)
		notifications.GET("/:id", h.NotifHandler.GetNotification)
		notifications.GET("/count/unread", h.NotifHandler.GetUnreadCount)
		notifications.GET("/summary", h.NotifHandler.GetSummary)
		notifications.PUT("/:id/read", h.NotifHandler.MarkAsRead)
		notifications.PUT("/read-all", h.NotifHandler.MarkAllAsRead)
		notifications.DELETE("/:id", h.NotifHandler.DeleteNotification)
	}

	// ==================== Subscription Plans ====================
	plans := api.Group("/plans")
	{
		// Public endpoints - no auth required
		plans.GET("/public", h.PlanHandler.ListPublicPlans)
		plans.GET("/compare", h.PlanHandler.ComparePlans)
		
		// Authenticated endpoints
		plansAuth := plans.Group("")
		plansAuth.Use(h.AuthMiddleware.Auth())
		{
			plansAuth.GET("", h.PlanHandler.ListPlans)
			plansAuth.GET("/:id", h.PlanHandler.GetPlan)
			plansAuth.GET("/code/:code", h.PlanHandler.GetPlanByCode)
		}
	}

	// ==================== Agent Customers ====================
	customers := api.Group("/customers")
	customers.Use(h.AuthMiddleware.Auth())
	{
		// List and search
		customers.GET("", h.CustomerHandler.ListCustomers)
		customers.GET("/search", h.CustomerHandler.SearchCustomers)
		customers.GET("/stats", h.CustomerHandler.GetCustomerStats)
		
		// Get by identifiers
		customers.GET("/:id", h.CustomerHandler.GetCustomer)
		customers.GET("/reference/:reference", h.CustomerHandler.GetCustomerByReference)
		customers.GET("/phone", h.CustomerHandler.GetCustomerByPhone) // ?phone=xxx
		
		// Create, update, delete
		customers.POST("", h.CustomerHandler.CreateCustomer)
		customers.PUT("/:id", h.CustomerHandler.UpdateCustomer)
		customers.DELETE("/:id", h.CustomerHandler.DeleteCustomer)
		
		// Status management
		customers.PUT("/:id/activate", h.CustomerHandler.ActivateCustomer)
		customers.PUT("/:id/deactivate", h.CustomerHandler.DeactivateCustomer)
		customers.PUT("/:id/verify", h.CustomerHandler.VerifyCustomer)
		
		// Tag management
		customers.POST("/:id/tags", h.CustomerHandler.AddTag)
		customers.DELETE("/:id/tags", h.CustomerHandler.RemoveTag) // ?tag=xxx
		
		// Bulk operations
		customers.POST("/bulk-import", h.CustomerHandler.BulkImportCustomers)
	}

	// ==================== Agent Offers ====================
	offers := api.Group("/offers")
	offers.Use(h.AuthMiddleware.Auth())
	{
		// List and search
		offers.GET("", h.OfferHandler.ListOffers)
		offers.GET("/featured", h.OfferHandler.GetFeaturedOffers)
		offers.GET("/search", h.OfferHandler.SearchOffers)
		offers.GET("/stats", h.OfferHandler.GetOfferStats)
		
		// Get by identifiers
		offers.GET("/:id", h.OfferHandler.GetOffer)
		offers.GET("/code/:code", h.OfferHandler.GetOfferByCode)
		offers.GET("/by-amount", h.OfferHandler.GetOffersByAmount) // ?amount=5
		offers.GET("/by-amount-range", h.OfferHandler.GetOffersByAmountRange) // ?min_amount=1&max_amount=10
		offers.GET("/by-type-and-amount", h.OfferHandler.GetOffersByTypeAndAmount) // ?type=data&amount=5

		// Get by price (new)
		offers.GET("/by-price", h.OfferHandler.GetOfferByPrice) // ?price=500
		offers.GET("/by-price-and-type", h.OfferHandler.GetOfferByPriceAndType) // ?price=500&type=data
		
		// Create, update, delete
		offers.POST("", h.OfferHandler.CreateOffer)
		offers.PUT("/:id", h.OfferHandler.UpdateOffer)
		offers.DELETE("/:id", h.OfferHandler.DeleteOffer)
		
		// Status management
		offers.PUT("/:id/activate", h.OfferHandler.ActivateOffer)
		offers.PUT("/:id/deactivate", h.OfferHandler.DeactivateOffer)
		offers.PUT("/:id/pause", h.OfferHandler.PauseOffer)
		
		// Utilities
		offers.POST("/:id/clone", h.OfferHandler.CloneOffer)
		offers.GET("/:id/ussd-code", h.OfferHandler.GenerateUSSDCode) // ?phone=xxx (deprecated, kept for backward compatibility)
		offers.GET("/:id/ussd-code/execute", h.OfferHandler.GetUSSDCodeForExecution) // ?phone=xxx (new endpoint)
		offers.GET("/:id/price", h.OfferHandler.CalculateOfferPrice)
		offers.GET("/:id/availability", h.OfferHandler.CheckOfferAvailability)

		ussdCodes := offers.Group("/:id/ussd-codes")
		{
			// List and retrieve
			ussdCodes.GET("", h.OfferHandler.ListUSSDCodes)
			ussdCodes.GET("/active", h.OfferHandler.GetActiveUSSDCodes)
			ussdCodes.GET("/primary", h.OfferHandler.GetPrimaryUSSDCode)
			ussdCodes.GET("/stats", h.OfferHandler.GetUSSDCodeStats)
			
			// Create and update
			ussdCodes.POST("", h.OfferHandler.AddUSSDCode)
			ussdCodes.PUT("/:ussd_code_id", h.OfferHandler.UpdateUSSDCode)
			
			// Priority management
			ussdCodes.PUT("/:ussd_code_id/set-primary", h.OfferHandler.SetUSSDCodeAsPrimary)
			ussdCodes.PUT("/reorder", h.OfferHandler.ReorderUSSDCodes)
			
			// Status and deletion
			ussdCodes.PUT("/:ussd_code_id/toggle-status", h.OfferHandler.ToggleUSSDCodeStatus)
			ussdCodes.DELETE("/:ussd_code_id", h.OfferHandler.DeleteUSSDCode)
			
			// Usage tracking
			ussdCodes.POST("/record-result", h.OfferHandler.RecordUSSDResult)
		}
	}

	// ==================== Agent Configurations ====================
	configs := api.Group("/configs")
	configs.Use(h.AuthMiddleware.Auth())
	{
		// General config management
		configs.POST("", h.ConfigHandler.CreateConfig)
		configs.GET("", h.ConfigHandler.ListConfigs)
		configs.GET("/all", h.ConfigHandler.GetAllConfigs)
		configs.GET("/global", h.ConfigHandler.GetGlobalConfigs)
		configs.GET("/:id", h.ConfigHandler.GetConfig)
		configs.GET("/key/:key", h.ConfigHandler.GetConfigByKey) // ?device_id=xxx
		configs.PUT("/:id", h.ConfigHandler.UpdateConfig)
		configs.DELETE("/:id", h.ConfigHandler.DeleteConfig)
		
		// Device-specific configs
		configs.GET("/devices/:device_id", h.ConfigHandler.GetDeviceConfigs)
		
		// Specific config types
		configTypes := configs.Group("/types")
		{
			// Notification settings
			configTypes.GET("/notifications", h.ConfigHandler.GetNotificationConfig)
			configTypes.PUT("/notifications", h.ConfigHandler.SetNotificationConfig)
			
			// USSD settings
			configTypes.GET("/ussd", h.ConfigHandler.GetUSSDConfig)
			configTypes.PUT("/ussd", h.ConfigHandler.SetUSSDConfig)
			
			// Android device settings
			configTypes.GET("/android/:device_id", h.ConfigHandler.GetAndroidDeviceConfig)
			configTypes.PUT("/android/:device_id", h.ConfigHandler.SetAndroidDeviceConfig)
			
			// Business settings
			configTypes.GET("/business", h.ConfigHandler.GetBusinessConfig)
			configTypes.PUT("/business", h.ConfigHandler.SetBusinessConfig)
			
			// Display settings
			configTypes.GET("/display", h.ConfigHandler.GetDisplayConfig)
			configTypes.PUT("/display", h.ConfigHandler.SetDisplayConfig)
			
			// Security settings
			configTypes.GET("/security", h.ConfigHandler.GetSecurityConfig)
			configTypes.PUT("/security", h.ConfigHandler.SetSecurityConfig)
		}
	}

	// ==================== Promotional Campaigns (Read Only for Users) ====================
	campaigns := api.Group("/campaigns")
	campaigns.Use(h.AuthMiddleware.Auth())
	{
		// List and view campaigns
		campaigns.GET("", h.CampaignHandler.ListCampaigns)
		campaigns.GET("/active", h.CampaignHandler.GetActiveCampaigns)
		campaigns.GET("/:id", h.CampaignHandler.GetCampaign)
		campaigns.GET("/:id/details", h.CampaignHandler.GetCampaignDetails)
		campaigns.GET("/code/:code", h.CampaignHandler.GetCampaignByCode)
		
		// Validation and application
		campaigns.POST("/validate", h.CampaignHandler.ValidateCampaign)
		campaigns.POST("/:id/apply", h.CampaignHandler.ApplyCampaign)
		campaigns.GET("/check-availability", h.CampaignHandler.CheckCampaignAvailability) // ?code=xxx
	}

	// ==================== Offer Requests & Redemptions ====================
	transactions := api.Group("/transactions")
	transactions.Use(h.AuthMiddleware.Auth())
	{
		// Offer Requests
		requests := transactions.Group("/requests")
		{
			// Create and list
			requests.POST("", h.TransactionHandler.CreateOfferRequest)
			requests.GET("", h.TransactionHandler.ListOfferRequests)
			requests.GET("/:id", h.TransactionHandler.GetOfferRequest)
			
			// Status-based retrieval
			requests.GET("/pending", h.TransactionHandler.GetPendingRequests)
			requests.GET("/failed", h.TransactionHandler.GetFailedRequests)
			requests.GET("/processing", h.TransactionHandler.GetProcessingRequests)
			requests.GET("/by-status", h.TransactionHandler.GetRequestsByStatus)
			
			// Status updates
			requests.PUT("/:id/status", h.TransactionHandler.UpdateOfferRequestStatus)
			requests.PUT("/:id/complete", h.TransactionHandler.CompleteOfferRequest)
			requests.PUT("/:id/processing", h.TransactionHandler.MarkAsProcessing)
			requests.POST("/:id/retry", h.TransactionHandler.RetryFailedRequest)
			
			// Batch operations
			requests.GET("/batch/pending", h.TransactionHandler.GetBatchPendingForDevice)
			requests.PUT("/batch/update", h.TransactionHandler.BatchUpdateRequests)
		}
		
		// Redemptions
		redemptions := transactions.Group("/redemptions")
		{
			redemptions.GET("", h.TransactionHandler.ListOfferRedemptions)
			redemptions.GET("/:id", h.TransactionHandler.GetOfferRedemption)
		}
		
		// Statistics
		transactions.GET("/stats", h.TransactionHandler.GetTransactionStats)
	}

	// ==================== Scheduled Offers ====================
	schedules := api.Group("/schedules")
	schedules.Use(h.AuthMiddleware.Auth())
	{
		// CRUD operations
		schedules.POST("", h.ScheduleHandler.CreateScheduledOffer)
		schedules.GET("", h.ScheduleHandler.ListScheduledOffers)
		schedules.GET("/:id", h.ScheduleHandler.GetScheduledOffer)
		schedules.PUT("/:id", h.ScheduleHandler.UpdateScheduledOffer)
		
		// Status management
		schedules.PUT("/:id/pause", h.ScheduleHandler.PauseScheduledOffer)
		schedules.PUT("/:id/resume", h.ScheduleHandler.ResumeScheduledOffer)
		schedules.PUT("/:id/cancel", h.ScheduleHandler.CancelScheduledOffer)
		
		// Execution
		schedules.POST("/:id/execute", h.ScheduleHandler.ExecuteScheduledOffer)
		schedules.GET("/due", h.ScheduleHandler.GetDueSchedules)
		
		// History
		schedules.GET("/:id/history", h.ScheduleHandler.GetScheduleHistory)
		
		// Statistics
		schedules.GET("/stats/overview", h.ScheduleHandler.GetScheduleStats)
		schedules.GET("/stats/by-status", h.ScheduleHandler.GetSchedulesByStatus)
		
		// Batch operations (for mobile app)
		schedules.GET("/batch/due", h.ScheduleHandler.GetBatchDueSchedules)
		schedules.POST("/batch/execute", h.ScheduleHandler.BatchExecuteSchedules)
	}

	// ==================== Agent Subscriptions ====================
	subscriptions := api.Group("/subscriptions")
	subscriptions.Use(h.AuthMiddleware.Auth())
	{
		// Create and renew (from mobile USSD payment)
		subscriptions.POST("", h.AgentSubscriptionHandler.CreateSubscription)
		subscriptions.POST("/renew", h.AgentSubscriptionHandler.RenewSubscription)
		
		// View subscriptions
		subscriptions.GET("", h.AgentSubscriptionHandler.ListSubscriptions)
		subscriptions.GET("/active", h.AgentSubscriptionHandler.GetActiveSubscription)
		subscriptions.GET("/:id", h.AgentSubscriptionHandler.GetSubscription)
		
		// Update and cancel
		subscriptions.PUT("/:id", h.AgentSubscriptionHandler.UpdateSubscription)
		subscriptions.POST("/:id/cancel", h.AgentSubscriptionHandler.CancelSubscription)
		
		// Usage and access
		subscriptions.GET("/usage/current", h.AgentSubscriptionHandler.GetSubscriptionUsage)
		subscriptions.GET("/access/check", h.AgentSubscriptionHandler.CheckSubscriptionAccess)
		
		// Statistics
		subscriptions.GET("/stats/overview", h.AgentSubscriptionHandler.GetSubscriptionStats)
		subscriptions.GET("/stats/by-status", h.AgentSubscriptionHandler.GetSubscriptionsByStatus)
	}

	// ==================== ADMIN ROUTES ====================
	admin := api.Group("/admin")
	{
		// Super Admin Only Routes
		superAdmin := admin.Group("")
		superAdmin.Use(h.AuthMiddleware.SuperAdminOnly()...)
		{
			superAdmin.POST("/admins", h.AuthHandler.CreateAdmin)
			superAdmin.GET("/admins", h.AuthHandler.ListAdmins)
			superAdmin.DELETE("/admins/:id", h.AuthHandler.DeactivateAdmin)
			superAdmin.GET("/ws/stats", h.WSHandler.GetStats)
		}

		// Any Admin Routes
		adminAuth := admin.Group("")
		adminAuth.Use(h.AuthMiddleware.AdminOnly()...)
		{
			// Notifications Management
			adminNotifications := adminAuth.Group("/notifications")
			{
				adminNotifications.POST("", h.NotifHandler.CreateNotification)
				adminNotifications.POST("/bulk", h.NotifHandler.SendBulkNotifications)
				adminNotifications.POST("/broadcast", h.NotifHandler.BroadcastNotification)
			}

			// Subscription Plans Management
			adminPlans := adminAuth.Group("/plans")
			{
				adminPlans.POST("", h.PlanHandler.CreatePlan)
				adminPlans.PUT("/:id", h.PlanHandler.UpdatePlan)
				adminPlans.PUT("/:id/activate", h.PlanHandler.ActivatePlan)
				adminPlans.PUT("/:id/deactivate", h.PlanHandler.DeactivatePlan)
				adminPlans.DELETE("/:id", h.PlanHandler.DeletePlan)
				adminPlans.GET("/stats", h.PlanHandler.GetPlanStats)
			}

			// Customer Management
			adminCustomers := adminAuth.Group("/customers")
			{
				// Admin can access any agent's customers with agent_id query param
				adminCustomers.GET("", h.CustomerHandler.ListCustomers)        // ?agent_id=1
				adminCustomers.GET("/stats", h.CustomerHandler.GetCustomerStats) // ?agent_id=1
				adminCustomers.GET("/:id", h.CustomerHandler.GetCustomer)      // ?agent_id=1
				adminCustomers.PUT("/:id", h.CustomerHandler.UpdateCustomer)   // ?agent_id=1
				adminCustomers.DELETE("/:id", h.CustomerHandler.DeleteCustomer) // ?agent_id=1
			}

			// Campaign Management
			adminCampaigns := adminAuth.Group("/campaigns")
			{
				// CRUD operations
				adminCampaigns.POST("", h.CampaignHandler.CreateCampaign)
				adminCampaigns.PUT("/:id", h.CampaignHandler.UpdateCampaign)
				adminCampaigns.DELETE("/:id", h.CampaignHandler.DeleteCampaign)
				
				// Status management
				adminCampaigns.PUT("/:id/activate", h.CampaignHandler.ActivateCampaign)
				adminCampaigns.PUT("/:id/deactivate", h.CampaignHandler.DeactivateCampaign)
				adminCampaigns.PUT("/:id/extend", h.CampaignHandler.ExtendCampaign)
				
				// Statistics
				adminCampaigns.GET("/stats", h.CampaignHandler.GetCampaignStats)
			}

			// Agent Subscription Management
			adminSubscriptions := adminAuth.Group("/subscriptions")
			{
				// View all subscriptions
				adminSubscriptions.GET("", h.AgentSubscriptionHandler.AdminListSubscriptions)
				adminSubscriptions.GET("/:id", h.AgentSubscriptionHandler.AdminGetSubscription)
				adminSubscriptions.GET("/expiring", h.AgentSubscriptionHandler.AdminGetExpiringSubscriptions)
				
				// Status management
				adminSubscriptions.PUT("/:id/deactivate", h.AgentSubscriptionHandler.AdminDeactivateSubscription)
				adminSubscriptions.PUT("/:id/suspend", h.AgentSubscriptionHandler.AdminSuspendSubscription)
				adminSubscriptions.PUT("/:id/reactivate", h.AgentSubscriptionHandler.AdminReactivateSubscription)
				adminSubscriptions.POST("/:id/cancel", h.AgentSubscriptionHandler.AdminCancelSubscription)
				
				// Statistics
				adminSubscriptions.GET("/stats", h.AgentSubscriptionHandler.AdminGetSubscriptionStats)
			}
		}
	}
}