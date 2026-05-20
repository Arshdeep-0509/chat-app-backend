package routes

import (
	"chat-app-backend/internal/controllers"
	"chat-app-backend/internal/middlewares"

	"github.com/gin-gonic/gin"
)

func SetupChatRequestRoutes(router *gin.RouterGroup, ctrl *controllers.ChatRequestController) {
	group := router.Group("/chat-requests")
	group.Use(middlewares.AuthMiddleware())
	{
		group.POST("/send", ctrl.SendChatRequest)
		group.GET("", ctrl.ListChatRequests)
		group.POST("/respond", ctrl.RespondChatRequest)
	}
}
