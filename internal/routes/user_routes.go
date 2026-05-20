package routes

import (
	"chat-app-backend/internal/controllers"
	"chat-app-backend/internal/middlewares"

	"github.com/gin-gonic/gin"
)

// SetupUserRoutes registers user-related endpoints
func SetupUserRoutes(router *gin.RouterGroup, userController *controllers.UserController) {
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/register", userController.Register)
		authGroup.POST("/login", userController.Login)
	}

	// Authenticated routes
	usersGroup := router.Group("/users")
	usersGroup.Use(middlewares.AuthMiddleware())
	{
		usersGroup.GET("", userController.ListUsers)
		usersGroup.GET("/profile", userController.GetProfile)
	}
}

