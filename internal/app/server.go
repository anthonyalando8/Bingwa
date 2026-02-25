// internal/app/server.go
package app

import (
	"context"
	"fmt"
	"log"

	"bingwa-service/internal/config"
	"bingwa-service/internal/db"
	authHandler "bingwa-service/internal/handlers/auth"
	configHandler "bingwa-service/internal/handlers/config"
	customerHandler "bingwa-service/internal/handlers/customer"
	notifyH "bingwa-service/internal/handlers/notification"
	offerHandler "bingwa-service/internal/handlers/offer"
	subhandler "bingwa-service/internal/handlers/subscription_plans"
	wsHandler "bingwa-service/internal/handlers/websocket"
	"bingwa-service/internal/middleware"
	"bingwa-service/internal/pkg/jwt"
	"bingwa-service/internal/pkg/session"
	"bingwa-service/internal/repository/postgres"
	authUsecase "bingwa-service/internal/service/auth"
	configUsecase "bingwa-service/internal/service/config"
	customersvc "bingwa-service/internal/service/customer"
	"bingwa-service/internal/service/email"
	notifyUsecase "bingwa-service/internal/service/notification"
	offerservice "bingwa-service/internal/service/offer"
	subscription "bingwa-service/internal/service/subscription_plans"
	campaignUsecase "bingwa-service/internal/service/campaign"
	scheduleUsecase "bingwa-service/internal/service/schedule"
	transactionUsecase "bingwa-service/internal/service/transaction"
	subscriptionUsecase "bingwa-service/internal/service/subscription"
	"bingwa-service/internal/websocket"
	wsHandlers "bingwa-service/internal/websocket/handler"
	campaignHandler "bingwa-service/internal/handlers/campaign"
	transactionHandler "bingwa-service/internal/handlers/transaction"
	scheduleHandler "bingwa-service/internal/handlers/schedule"
	subscriptionHandler "bingwa-service/internal/handlers/subscription"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Server struct {
	cfg    config.AppConfig
	engine *gin.Engine
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
	db := postgres.NewDB(pool)
	authRepo := postgres.NewAuthRepository(pool)
	notifyRepo := postgres.NewNotificationRepository(pool)
	planRepo := postgres.NewSubscriptionPlanRepository(pool)
	customerRepo := postgres.NewAgentCustomerRepository(pool)
	offerRepo := postgres.NewAgentOfferRepository(pool, ussdCodeRepo, db)
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

	notifService := notifyUsecase.NewNotificationService(notifyRepo, hub)
	planService := subscription.NewPlanService(planRepo, logger)
	customerService := customersvc.NewCustomerService(customerRepo, logger)
	offerService := offerservice.NewOfferService(offerRepo, ussdCodeRepo, logger)
	configService := configUsecase.NewConfigService(configRepo, logger)
	campaignService := campaignUsecase.NewCampaignService(campaignRepo, logger)
	transactionService := transactionUsecase.NewTransactionService(
		requestRepo,
		redemptionRepo,
		offerRepo,
		customerRepo,
		db,
		logger,
	)
	scheduleService := scheduleUsecase.NewScheduleService(
		scheduleRepo,
		scheduleHistoryRepo,
		redemptionRepo,
		offerRepo,
		customerRepo,
		db,
		logger,
	)
	agentSubscriptionService := subscriptionUsecase.NewSubscriptionService(
		agentSubscriptionRepo,
		planRepo,
		campaignRepo,
		db,
		logger,
	)

	// ----- Handlers -----
	authHandlerInst := authHandler.NewAuthHandler(authService, logger)
	notifHandler := notifyH.NewNotificationHandler(notifService)
	planHandler := subhandler.NewPlanHandler(planService)
	customerHandler := customerHandler.NewCustomerHandler(customerService)
	offerHandler := offerHandler.NewOfferHandler(offerService)
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
		AuthHandler:    authHandlerInst,
		NotifHandler:   notifHandler,
		PlanHandler:    planHandler,
		CustomerHandler: customerHandler,
		OfferHandler:    offerHandler,
		ConfigHandler:   configHandlerInst,
		CampaignHandler: campaignHandlerInst,
		TransactionHandler: transactionHandlerInst,
		ScheduleHandler:    scheduleHandlerInst,
		AgentSubscriptionHandler: agentSubscriptionHandlerInst,
		WSHandler:      wsHandlerInst,
		AuthMiddleware: authMiddleware,
	}
	SetupRouter(s.engine, logger, handlers)

	// ----- Start HTTP -----
	log.Printf("🚀 Server running on %s", s.cfg.HTTPAddr)
	return s.engine.Run(s.cfg.HTTPAddr)
}
