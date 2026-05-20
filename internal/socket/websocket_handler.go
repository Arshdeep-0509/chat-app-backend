package socket

import (
	"log"
	"net/http"

	"chat-app-backend/internal/repositories"
	"chat-app-backend/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development/testing
	},
}

type WebSocketHandler struct {
	hub              *Hub
	userRepo         repositories.UserRepository
	messageRepo      repositories.MessageRepository
	conversationRepo repositories.ConversationRepository
}

func NewWebSocketHandler(
	hub *Hub,
	userRepo repositories.UserRepository,
	messageRepo repositories.MessageRepository,
	conversationRepo repositories.ConversationRepository,
) *WebSocketHandler {
	return &WebSocketHandler{
		hub:              hub,
		userRepo:         userRepo,
		messageRepo:      messageRepo,
		conversationRepo: conversationRepo,
	}
}

func (h *WebSocketHandler) HandleWS(c *gin.Context) {
	// WebSockets usually pass token via query parameter as standard browser WebSocket client API doesn't support custom headers
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token query parameter is required (?token=...)"})
		return
	}

	userID, err := utils.ValidateToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token: " + err.Error()})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	client := &Client{
		Hub:              h.hub,
		Conn:             conn,
		Send:             make(chan []byte, 256),
		UserID:           userID,
		UserRepo:         h.userRepo,
		MessageRepo:      h.messageRepo,
		ConversationRepo: h.conversationRepo,
	}

	h.hub.Register <- client

	// Broadcast user online status in a non-blocking goroutine
	go h.hub.BroadcastUserStatus(userID, true, h.userRepo, h.conversationRepo)

	go client.WritePump()
	go client.ReadPump()
}
