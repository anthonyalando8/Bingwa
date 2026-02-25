// internal/app/server.go
package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"bingwa-service/internal/config"
	"bingwa-service/internal/db"
	authHandler "bingwa-service/internal/handlers/auth"
	campaignHandler "bingwa-service/internal/handlers/campaign"
	configHandler "bingwa-service/internal/handlers/config"
	customerHandler "bingwa-service/internal/handlers/customer"
	notifyH "bingwa-service/internal/handlers/notification"
	offerHandler "bingwa-service/internal/handlers/offer"
	scheduleHandler "bingwa-service/internal/handlers/schedule"
	subscriptionHandler "bingwa-service/internal/handlers/subscription"
	subhandler "bingwa-service/internal/handlers/subscription_plans"
	transactionHandler "bingwa-service/internal/handlers/transaction"
	wsHandler "bingwa-service/internal/handlers/websocket"
	"bingwa-service/internal/middleware"
	"bingwa-service/internal/pkg/jwt"
	"bingwa-service/internal/pkg/session"
	"bingwa-service/internal/repository/postgres"
	authUsecase "bingwa-service/internal/service/auth"
	campaignUsecase "bingwa-service/internal/service/campaign"
	configUsecase "bingwa-service/internal/service/config"
	customersvc "bingwa-service/internal/service/customer"
	"bingwa-service/internal/service/email"
	notifyUsecase "bingwa-service/internal/service/notification"
	offerservice "bingwa-service/internal/service/offer"
	scheduleUsecase "bingwa-service/internal/service/schedule"
	subscriptionUsecase "bingwa-service/internal/service/subscription"
	subscription "bingwa-service/internal/service/subscription_plans"
	transactionUsecase "bingwa-service/internal/service/transaction"
	"bingwa-service/internal/websocket"
	wsHandlers "bingwa-service/internal/websocket/handler"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Server struct {
	cfg         config.AppConfig
	engine      *gin.Engine
	logger      *zap.Logger
	authService *authUsecase.AuthService
}

func NewServer() *Server {
	cfg := config.Load()
	engine := gin.Default()
	return &Server{cfg: cfg, engine: engine}
}

func (s *Server) Start() error {
	ctx := context.Background()

	// ----- PostgreSQL -----
	pool, err := db.ConnectDB()
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	// ----- Redis -----
	redisCfg := db.RedisConfig{
		ClusterMode: false,
		Addresses:   []string{s.cfg.RedisAddr},
		Password:    s.cfg.RedisPass,
		DB:          0,
		PoolSize:    10,
	}

	redisClient, err := db.NewRedisClient(redisCfg)
	if err != nil {
		log.Fatalf("[REDIS] ❌ Failed to connect to Redis: %v", err)
	}
	log.Println("[REDIS] ✅ Connected successfully")

	// ----- Logger -----
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	s.logger = logger // Store logger in server

	// ----- JWT Manager -----
	jwtManager, err := jwt.LoadAndBuild(s.cfg.JWT)
	if err != nil {
		return fmt.Errorf("failed to load JWT manager: %w", err)
	}

	// ----- Session Manager & Rate Limiter -----
	sessionManager := session.NewManager(redisClient, nil) // Will set authRepo later
	rateLimiter := session.NewRateLimiter(redisClient)

	// ----- Email -----
	emailSender := email.NewEmailSender(
		s.cfg.SMTPHost,
		s.cfg.SMTPPort,
		s.cfg.SMTPUser,
		s.cfg.SMTPPass,
		s.cfg.SMTPFromName,
		s.cfg.SMTPSecure,
	)

	// ----- Repositories -----
	ussdCodeRepo := postgres.NewOfferUSSDCodeRepository(pool)
	dbWrapper := postgres.NewDB(pool)
	authRepo := postgres.NewAuthRepository(pool)
	notifyRepo := postgres.NewNotificationRepository(pool)
	planRepo := postgres.NewSubscriptionPlanRepository(pool)
	customerRepo := postgres.NewAgentCustomerRepository(pool)
	offerRepo := postgres.NewAgentOfferRepository(pool, ussdCodeRepo, dbWrapper)
	configRepo := postgres.NewAgentConfigRepository(pool)
	campaignRepo := postgres.NewPromotionalCampaignRepository(pool)
	requestRepo := postgres.NewOfferRequestRepository(pool)
	redemptionRepo := postgres.NewOfferRedemptionRepository(pool)
	scheduleRepo := postgres.NewScheduledOfferRepository(pool)
	scheduleHistoryRepo := postgres.NewScheduledOfferHistoryRepository(pool)
	agentSubscriptionRepo := postgres.NewAgentSubscriptionRepository(pool)

	// Update session manager with auth repo
	sessionManager = session.NewManager(redisClient, authRepo)

	// ----- WebSocket Hub -----
	hub := websocket.NewHub(jwtManager.Verifier, sessionManager)

	// Register WebSocket handlers
	notificationWSHandler := wsHandlers.NewNotificationHandler(notifyRepo)
	hub.RegisterHandler(notificationWSHandler)

	// Start hub
	go hub.Run(context.Background())

	// ----- Services (Usecases) -----
	authService := authUsecase.NewAuthService(
		authRepo,
		jwtManager,
		sessionManager,
		rateLimiter,
		emailSender,
		hub,
		redisClient,
		logger,
	)
	s.authService = authService // Store authService in server

	notifService := notifyUsecase.NewNotificationService(notifyRepo, hub)
	planService := subscription.NewPlanService(planRepo, logger)
	customerService := customersvc.NewCustomerService(customerRepo, logger)
	offerService := offerservice.NewOfferService(offerRepo, ussdCodeRepo, logger)
	configService := configUsecase.NewConfigService(configRepo, logger)
	campaignService := campaignUsecase.NewCampaignService(campaignRepo, logger)
	agentSubscriptionService := subscriptionUsecase.NewSubscriptionService(
		agentSubscriptionRepo,
		planRepo,
		campaignRepo,
		dbWrapper,
		logger,
	)
	transactionService := transactionUsecase.NewTransactionService(
		requestRepo,
		redemptionRepo,
		offerRepo,
		customerRepo,
		offerService,
		customerService,
		agentSubscriptionService,
		dbWrapper,
		logger,
	)
	scheduleService := scheduleUsecase.NewScheduleService(
		scheduleRepo,
		scheduleHistoryRepo,
		redemptionRepo,
		offerRepo,
		customerRepo,
		customerService,
		dbWrapper,
		offerService,
		logger,
	)
	

	// ----- Initialize Super Admin -----
	if err := s.initializeSuperAdmin(); err != nil {
		logger.Error("failed to initialize super admin", zap.Error(err))
		// Don't fail startup, just log the error
	}

	// ----- Handlers -----
	authHandlerInst := authHandler.NewAuthHandler(authService, logger)
	notifHandler := notifyH.NewNotificationHandler(notifService)
	planHandler := subhandler.NewPlanHandler(planService)
	customerHandlerInst := customerHandler.NewCustomerHandler(customerService)
	offerHandlerInst := offerHandler.NewOfferHandler(offerService)
	configHandlerInst := configHandler.NewConfigHandler(configService)
	campaignHandlerInst := campaignHandler.NewCampaignHandler(campaignService)
	transactionHandlerInst := transactionHandler.NewTransactionHandler(transactionService)
	wsHandlerInst := wsHandler.NewWebSocketHandler(hub, logger)
	scheduleHandlerInst := scheduleHandler.NewScheduleHandler(scheduleService)
	agentSubscriptionHandlerInst := subscriptionHandler.NewAgentSubscriptionHandler(agentSubscriptionService)

	// ----- Middlewares -----
	authMiddleware := middleware.NewAuthMiddleware(authService)

	s.engine.Use(
		middleware.RecoveryMiddleware(logger),
		middleware.LoggingMiddleware(logger),
		middleware.CORSMiddleware(),
	)

	// ----- Router -----
	handlers := &Handlers{
		AuthHandler:              authHandlerInst,
		NotifHandler:             notifHandler,
		PlanHandler:              planHandler,
		CustomerHandler:          customerHandlerInst,
		OfferHandler:             offerHandlerInst,
		ConfigHandler:            configHandlerInst,
		CampaignHandler:          campaignHandlerInst,
		TransactionHandler:       transactionHandlerInst,
		ScheduleHandler:          scheduleHandlerInst,
		AgentSubscriptionHandler: agentSubscriptionHandlerInst,
		WSHandler:                wsHandlerInst,
		AuthMiddleware:           authMiddleware,
	}
	SetupRouter(s.engine, logger, handlers)

	// ----- Start HTTP -----
	log.Printf("🚀 Server running on %s", s.cfg.HTTPAddr)
	return s.engine.Run(s.cfg.HTTPAddr)
}

// initializeSuperAdmin creates super admin if it doesn't exist
func (s *Server) initializeSuperAdmin() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get super admin credentials from environment
	email := os.Getenv("SUPER_ADMIN_EMAIL")
	password := os.Getenv("SUPER_ADMIN_PASSWORD")
	fullName := os.Getenv("SUPER_ADMIN_NAME")

	// Use defaults if not provided (for development only)
	if email == "" {
		email = "admin@bingwa.app"
		s.logger.Warn("SUPER_ADMIN_EMAIL not set, using default", zap.String("email", email))
	}
	if password == "" {
		password = "HappyOwl58&" // Strong default
		s.logger.Warn("SUPER_ADMIN_PASSWORD not set, using default password")
	}
	if fullName == "" {
		fullName = "Super Administrator"
		s.logger.Warn("SUPER_ADMIN_NAME not set, using default", zap.String("name", fullName))
	}

	// Validate password strength (optional but recommended)
	if len(password) < 8 {
		s.logger.Error("super admin password is too weak (minimum 8 characters)")
		return fmt.Errorf("super admin password must be at least 8 characters")
	}

	// Create super admin
	if err := s.authService.EnsureSuperAdminExists(ctx, email, password, fullName); err != nil {
		return fmt.Errorf("failed to ensure super admin exists: %w", err)
	}

	return nil
}