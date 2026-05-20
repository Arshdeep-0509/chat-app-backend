package controllers

import (
	"net/http"

	"chat-app-backend/internal/models"
	"chat-app-backend/internal/services"

	"github.com/gin-gonic/gin"
)

type ChatRequestController struct {
	chatRequestService services.ChatRequestService
}

func NewChatRequestController(chatRequestService services.ChatRequestService) *ChatRequestController {
	return &ChatRequestController{
		chatRequestService: chatRequestService,
	}
}

func (ctrl *ChatRequestController) SendChatRequest(c *gin.Context) {
	senderIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	senderID := senderIDVal.(string)

	var input models.SendChatRequestInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req, err := ctrl.chatRequestService.SendRequest(c.Request.Context(), senderID, input.ReceiverID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":      "Chat request sent successfully",
		"chat_request": req,
	})
}

func (ctrl *ChatRequestController) ListChatRequests(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := userIDVal.(string)

	direction := c.Query("direction")
	if direction != "outgoing" {
		direction = "incoming"
	}

	requests, err := ctrl.chatRequestService.ListRequests(c.Request.Context(), userID, direction)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, requests)
}

func (ctrl *ChatRequestController) RespondChatRequest(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := userIDVal.(string)

	var input models.RespondChatRequestInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req, err := ctrl.chatRequestService.RespondRequest(c.Request.Context(), input.RequestID, userID, input.Status)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	message := "Chat request accepted"
	if input.Status == "rejected" {
		message = "Chat request rejected"
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      message,
		"chat_request": req,
	})
}
