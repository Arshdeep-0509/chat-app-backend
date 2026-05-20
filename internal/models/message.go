package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Message struct {
	ID             primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	ConversationID primitive.ObjectID  `bson:"conversation_id" json:"conversation_id"`
	SenderID       primitive.ObjectID  `bson:"sender_id" json:"sender_id"`
	Content        string              `bson:"content" json:"content"`
	CreatedAt      int64               `bson:"created_at" json:"created_at"` // Epoch milliseconds
	ParentID       *primitive.ObjectID `bson:"parent_id,omitempty" json:"parent_id,omitempty"` // For replies
}
