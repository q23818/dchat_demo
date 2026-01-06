package main

import (
	"log"

	"github.com/everest-an/dchat-backend/internal/auth"
	"github.com/everest-an/dchat-backend/internal/config"
	"github.com/everest-an/dchat-backend/internal/database"
	"github.com/everest-an/dchat-backend/internal/handlers"
	"github.com/everest-an/dchat-backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	db, err := database.New(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize services
	jwtService := auth.NewJWTService(&cfg.JWT)
	web3Service := auth.NewWeb3Service()
	userService := auth.NewUserService(db.DB)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(userService, jwtService, web3Service)
	messageHandler := handlers.NewMessageHandler(db.DB)

	// Setup Gin router
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Apply middleware
	router.Use(middleware.CORSMiddleware())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "dChat API",
			"version": "2.0.0-go",
		})
	})

	// Public routes
	api := router.Group("/api")
	{
		api.POST("/auth/nonce", authHandler.GetNonce)
		api.POST("/auth/wallet-login", authHandler.WalletLogin)
	}

	// Protected routes
	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware(jwtService))
	{
		// User routes
		protected.GET("/user/me", authHandler.GetCurrentUser)

		// Message routes
		protected.POST("/messages", messageHandler.SendMessage)
		protected.GET("/messages/:user_id", messageHandler.GetMessages)
		protected.GET("/conversations", messageHandler.GetConversations)
		protected.PUT("/messages/read/:sender_id", messageHandler.MarkAsRead)
	}

	// Start server
	port := ":" + cfg.Server.APIPort
	log.Printf("🚀 dChat API server starting on port %s", port)
	if err := router.Run(port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
