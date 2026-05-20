package main

import (
	"log"
	"os"

	"chat-app-backend/internal/config"
	"chat-app-backend/internal/controllers"
	"chat-app-backend/internal/database"
	"chat-app-backend/internal/middlewares"
	"chat-app-backend/internal/repositories"
	"chat-app-backend/internal/routes"
	"chat-app-backend/internal/services"
	"chat-app-backend/internal/socket"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load environment variables
	config.LoadEnv()

	// Connect to MongoDB
	db := database.ConnectDB()

	// Initialize Repositories
	userRepo := repositories.NewUserRepository(db)
	chatRequestRepo := repositories.NewChatRequestRepository(db)
	conversationRepo := repositories.NewConversationRepository(db)
	messageRepo := repositories.NewMessageRepository(db)

	// Initialize Services
	userService := services.NewUserService(userRepo)
	chatRequestService := services.NewChatRequestService(chatRequestRepo, conversationRepo, userRepo)
	conversationService := services.NewConversationService(conversationRepo, messageRepo, userRepo)

	// Initialize WebSocket Server
	hub := socket.NewHub()
	go hub.Run()
	wsHandler := socket.NewWebSocketHandler(hub, userRepo, messageRepo, conversationRepo)

	// Initialize Controllers
	userController := controllers.NewUserController(userService)
	chatRequestController := controllers.NewChatRequestController(chatRequestService)
	conversationController := controllers.NewConversationController(conversationService, conversationRepo, hub)

	// Setup Gin router
	router := gin.Default()

	// Enable CORS
	router.Use(middlewares.CORSMiddleware())

	// Health check route
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "server running",
		})
	})

	// Setup API routes
	apiGroup := router.Group("/api")
	routes.SetupUserRoutes(apiGroup, userController)
	routes.SetupChatRequestRoutes(apiGroup, chatRequestController)
	routes.SetupConversationRoutes(apiGroup, conversationController)

	// Setup WebSocket endpoint
	router.GET("/ws", wsHandler.HandleWS)

	// Get PORT from env or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start server
	log.Printf("Server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}