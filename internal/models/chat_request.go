package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ChatRequest struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	SenderID   primitive.ObjectID `bson:"sender_id" json:"sender_id"`
	ReceiverID primitive.ObjectID `bson:"receiver_id" json:"receiver_id"`
	Status     string             `bson:"status" json:"status"` // "pending", "accepted", "rejected"
	CreatedAt  int64              `bson:"created_at" json:"created_at"` // Epoch milliseconds
	UpdatedAt  int64              `bson:"updated_at" json:"updated_at"` // Epoch milliseconds

	// Populated for response ease
	Sender   *User `bson:"-" json:"sender,omitempty"`
	Receiver *User `bson:"-" json:"receiver,omitempty"`
}

type SendChatRequestInput struct {
	ReceiverID string `json:"receiver_id" binding:"required"`
}

type RespondChatRequestInput struct {
	RequestID string `json:"request_id" binding:"required"`
	Status    string `json:"status" binding:"required,oneof=accepted rejected"`
}
