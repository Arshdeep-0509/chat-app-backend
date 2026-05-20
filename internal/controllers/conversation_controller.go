package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"chat-app-backend/internal/models"
	"chat-app-backend/internal/repositories"
	"chat-app-backend/internal/services"
	"chat-app-backend/internal/socket"

	"github.com/gin-gonic/gin"
)

type ConversationController struct {
	conversationService services.ConversationService
	conversationRepo    repositories.ConversationRepository
	hub                 *socket.Hub
}

func NewConversationController(
	conversationService services.ConversationService,
	conversationRepo repositories.ConversationRepository,
	hub *socket.Hub,
) *ConversationController {
	return &ConversationController{
		conversationService: conversationService,
		conversationRepo:    conversationRepo,
		hub:                 hub,
	}
}

func (ctrl *ConversationController) ListConversations(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := userIDVal.(string)

	conversations, err := ctrl.conversationService.ListConversations(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, conversations)
}

func (ctrl *ConversationController) GetMessages(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := userIDVal.(string)

	conversationID := c.Param("id")
	if conversationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "conversation ID is required"})
		return
	}

	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 20
	}

	messages, err := ctrl.conversationService.GetMessages(c.Request.Context(), conversationID, userID, page, limit)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, messages)
}

type EditMessageInput struct {
	Content string `json:"content" binding:"required"`
}

func (ctrl *ConversationController) EditMessage(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := userIDVal.(string)

	messageID := c.Param("message_id")
	if messageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message ID is required"})
		return
	}

	var input EditMessageInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	msg, err := ctrl.conversationService.EditMessage(c.Request.Context(), messageID, input.Content, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Fetch conversation participants for broadcasting
	conv, err := ctrl.conversationRepo.FindByID(c.Request.Context(), msg.ConversationID.Hex())
	if err == nil && conv != nil {
		response := models.WSResponse{
			Event:          "edit_message",
			ConversationID: conv.ID.Hex(),
			MessageID:      messageID,
			Message:        msg,
		}
		responseBytes, _ := json.Marshal(response)
		for _, p := range conv.Participants {
			ctrl.hub.BroadcastToUser(p.Hex(), responseBytes)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Message updated successfully",
		"data":    msg,
	})
}

func (ctrl *ConversationController) DeleteMessage(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := userIDVal.(string)

	messageID := c.Param("message_id")
	if messageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message ID is required"})
		return
	}

	msg, err := ctrl.conversationService.DeleteMessage(c.Request.Context(), messageID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Fetch conversation participants for broadcasting
	conv, err := ctrl.conversationRepo.FindByID(c.Request.Context(), msg.ConversationID.Hex())
	if err == nil && conv != nil {
		response := models.WSResponse{
			Event:          "delete_message",
			ConversationID: conv.ID.Hex(),
			MessageID:      messageID,
		}
		responseBytes, _ := json.Marshal(response)
		for _, p := range conv.Participants {
			ctrl.hub.BroadcastToUser(p.Hex(), responseBytes)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Message deleted successfully",
	})
}
