package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Conversation struct {
	ID           primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	Participants []primitive.ObjectID `bson:"participants" json:"participants"`
	CreatedAt    int64                `bson:"created_at" json:"created_at"` // Epoch milliseconds
	UpdatedAt    int64                `bson:"updated_at" json:"updated_at"` // Epoch milliseconds

	// Populated details for the UI
	ParticipantDetails []*User  `bson:"-" json:"participant_details,omitempty"`
	LastMessage        *Message `bson:"-" json:"last_message,omitempty"`
}
