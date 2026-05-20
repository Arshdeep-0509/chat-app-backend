package socket

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"chat-app-backend/internal/models"
	"chat-app-backend/internal/repositories"

	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 1024
)

type Client struct {
	Hub              *Hub
	Conn             *websocket.Conn
	Send             chan []byte
	UserID           string
	UserRepo         repositories.UserRepository
	MessageRepo      repositories.MessageRepository
	ConversationRepo repositories.ConversationRepository
}

// ReadPump pumps messages from the websocket connection to the hub and database.
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
		// Broadcast user offline status in a non-blocking goroutine
		go c.Hub.BroadcastUserStatus(c.UserID, false, c.UserRepo, c.ConversationRepo)
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, messageBytes, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error reading message: %v", err)
			}
			break
		}

		// Parse the incoming message
		var request models.WSRequest
		if err := json.Unmarshal(messageBytes, &request); err != nil {
			log.Printf("invalid json request: %v", err)
			continue
		}

		senderID, err := primitive.ObjectIDFromHex(c.UserID)
		if err != nil {
			log.Printf("invalid sender ID hex: %s", c.UserID)
			continue
		}

		ctx := context.Background()

		event := request.Event
		if event == "" && request.ConversationID != "" && request.Content != "" {
			event = "send_message"
		}

		switch event {
		case "send_message":
			if request.ConversationID == "" || request.Content == "" {
				continue
			}

			convID, err := primitive.ObjectIDFromHex(request.ConversationID)
			if err != nil {
				continue
			}

			// Validate conversation membership
			conv, err := c.ConversationRepo.FindByID(ctx, request.ConversationID)
			if err != nil || conv == nil {
				continue
			}

			isParticipant := false
			for _, p := range conv.Participants {
				if p == senderID {
					isParticipant = true
					break
				}
			}
			if !isParticipant {
				continue
			}

			var parentID *primitive.ObjectID
			if request.ParentID != "" {
				pID, err := primitive.ObjectIDFromHex(request.ParentID)
				if err == nil {
					// Verify parent message exists and belongs to the same conversation
					parentMsg, err := c.MessageRepo.FindByID(ctx, request.ParentID)
					if err == nil && parentMsg != nil && parentMsg.ConversationID == convID {
						parentID = &pID
					}
				}
			}

			// Create and store message
			now := time.Now().UnixMilli()
			msg := &models.Message{
				ID:             primitive.NewObjectID(),
				ConversationID: convID,
				SenderID:       senderID,
				Content:        request.Content,
				CreatedAt:      now,
				ParentID:       parentID,
			}

			if err := c.MessageRepo.Create(ctx, msg); err != nil {
				log.Printf("failed to save message: %v", err)
				continue
			}

			// Update conversation updatedAt timestamp
			_ = c.ConversationRepo.UpdateTimestamp(ctx, convID, now)

			// Broadcast message
			response := models.WSResponse{
				Event:          "new_message",
				ConversationID: request.ConversationID,
				Message:        msg,
			}

			responseBytes, err := json.Marshal(response)
			if err != nil {
				continue
			}

			for _, p := range conv.Participants {
				c.Hub.BroadcastToUser(p.Hex(), responseBytes)
			}

		case "typing":
			if request.ConversationID == "" {
				continue
			}

			conv, err := c.ConversationRepo.FindByID(ctx, request.ConversationID)
			if err != nil || conv == nil {
				continue
			}

			response := models.WSResponse{
				Event:          "typing",
				ConversationID: request.ConversationID,
				UserID:         c.UserID,
				IsTyping:       request.IsTyping,
			}

			responseBytes, err := json.Marshal(response)
			if err != nil {
				continue
			}

			for _, p := range conv.Participants {
				if p == senderID {
					continue // Don't send typing notification back to the sender
				}
				c.Hub.BroadcastToUser(p.Hex(), responseBytes)
			}

		case "edit_message":
			if request.MessageID == "" || request.Content == "" {
				continue
			}

			// Fetch original message to verify owner
			msg, err := c.MessageRepo.FindByID(ctx, request.MessageID)
			if err != nil || msg == nil {
				continue
			}

			if msg.SenderID != senderID {
				continue // Only sender can edit their message
			}

			// Update content in DB
			if err := c.MessageRepo.UpdateContent(ctx, request.MessageID, request.Content); err != nil {
				continue
			}

			msg.Content = request.Content

			// Find conversation to get participants
			conv, err := c.ConversationRepo.FindByID(ctx, msg.ConversationID.Hex())
			if err != nil || conv == nil {
				continue
			}

			response := models.WSResponse{
				Event:          "edit_message",
				ConversationID: conv.ID.Hex(),
				MessageID:      request.MessageID,
				Message:        msg,
			}

			responseBytes, err := json.Marshal(response)
			if err != nil {
				continue
			}

			for _, p := range conv.Participants {
				c.Hub.BroadcastToUser(p.Hex(), responseBytes)
			}

		case "delete_message":
			if request.MessageID == "" {
				continue
			}

			// Fetch original message to verify owner
			msg, err := c.MessageRepo.FindByID(ctx, request.MessageID)
			if err != nil || msg == nil {
				continue
			}

			if msg.SenderID != senderID {
				continue // Only sender can delete their message
			}

			// Delete message from DB
			if err := c.MessageRepo.Delete(ctx, request.MessageID); err != nil {
				continue
			}

			// Find conversation to get participants
			conv, err := c.ConversationRepo.FindByID(ctx, msg.ConversationID.Hex())
			if err != nil || conv == nil {
				continue
			}

			response := models.WSResponse{
				Event:          "delete_message",
				ConversationID: conv.ID.Hex(),
				MessageID:      request.MessageID,
			}

			responseBytes, err := json.Marshal(response)
			if err != nil {
				continue
			}

			for _, p := range conv.Participants {
				c.Hub.BroadcastToUser(p.Hex(), responseBytes)
			}
		}
	}
}

// WritePump pumps messages from the hub to the websocket connection.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
