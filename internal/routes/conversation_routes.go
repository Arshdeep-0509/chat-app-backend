package routes

import (
	"chat-app-backend/internal/controllers"
	"chat-app-backend/internal/middlewares"

	"github.com/gin-gonic/gin"
)

func SetupConversationRoutes(router *gin.RouterGroup, ctrl *controllers.ConversationController) {
	group := router.Group("/conversations")
	group.Use(middlewares.AuthMiddleware())
	{
		group.GET("", ctrl.ListConversations)
		group.GET("/:id/messages", ctrl.GetMessages)
		group.PUT("/messages/:message_id", ctrl.EditMessage)
		group.DELETE("/messages/:message_id", ctrl.DeleteMessage)
	}
}
