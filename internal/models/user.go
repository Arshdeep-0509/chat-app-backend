package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents the structure of the user document in MongoDB.
type User struct {
	ID             primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	Name           string               `bson:"name" json:"name"`
	Username       string               `bson:"username" json:"username"`
	Email          string               `bson:"email" json:"email"`
	Password       string               `bson:"password" json:"-"` // Not serialized in JSON
	ProfilePicture string               `bson:"profile_picture" json:"profile_picture"`
	Bio            string               `bson:"bio" json:"bio"`
	IsOnline       bool                 `bson:"is_online" json:"is_online"`
	LastSeen       time.Time            `bson:"last_seen" json:"last_seen"`
	BlockedUsers   []primitive.ObjectID `bson:"blocked_users" json:"blocked_users"`
	CreatedAt      time.Time            `bson:"created_at" json:"created_at"`
	UpdatedAt      time.Time            `bson:"updated_at" json:"updated_at"`

	// Relationship flags (populated dynamically based on current user session)
	RequestStatus  string               `bson:"-" json:"request_status,omitempty"`
}

// RegisterRequest is the data required to register a new user.
type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Username string `json:"username" binding:"required,min=3"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// LoginRequest is the data required to log in.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}
