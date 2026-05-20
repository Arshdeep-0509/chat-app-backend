package socket

import (
	"context"
	"encoding/json"
	"sync"

	"chat-app-backend/internal/models"
	"chat-app-backend/internal/repositories"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Hub struct {
	// Registered clients.
	Clients map[string]*Client // Map of UserID (hex string) to *Client
	// Register requests from the clients.
	Register chan *Client
	// Unregister requests from clients.
	Unregister chan *Client
	// Mutex for safety
	mu sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[string]*Client),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			// Unregister any existing connection for this user first
			if oldClient, ok := h.Clients[client.UserID]; ok {
				close(oldClient.Send)
				oldClient.Conn.Close()
			}
			h.Clients[client.UserID] = client
			h.mu.Unlock()
		case client := <-h.Unregister:
			h.mu.Lock()
			if c, ok := h.Clients[client.UserID]; ok && c == client {
				delete(h.Clients, client.UserID)
				close(client.Send)
			}
			h.mu.Unlock()
		}
	}
}

func (h *Hub) GetClient(userID string) (*Client, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	c, ok := h.Clients[userID]
	return c, ok
}

// BroadcastToUser sends a byte payload to a specific user if they are online
func (h *Hub) BroadcastToUser(userID string, messageBytes []byte) {
	h.mu.RLock()
	client, ok := h.Clients[userID]
	h.mu.RUnlock()
	if ok {
		select {
		case client.Send <- messageBytes:
		default:
			close(client.Send)
			h.mu.Lock()
			delete(h.Clients, userID)
			h.mu.Unlock()
		}
	}
}

// BroadcastUserStatus updates the DB and broadcasts user status change to all their conversation participants
func (h *Hub) BroadcastUserStatus(userID string, isOnline bool, userRepo repositories.UserRepository, convRepo repositories.ConversationRepository) {
	ctx := context.Background()
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return
	}

	// Update DB status
	_ = userRepo.UpdateOnlineStatus(ctx, userID, isOnline)

	// Find conversations
	conversations, err := convRepo.FindAllForUser(ctx, objID)
	if err != nil {
		return
	}

	response := models.WSResponse{
		Event:    "user_status",
		UserID:   userID,
		IsOnline: isOnline,
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		return
	}

	// Send to all participants of those conversations who are online
	visited := make(map[string]bool)
	for _, conv := range conversations {
		for _, partID := range conv.Participants {
			partHex := partID.Hex()
			if partHex == userID {
				continue
			}
			if !visited[partHex] {
				visited[partHex] = true
				h.BroadcastToUser(partHex, responseBytes)
			}
		}
	}
}
