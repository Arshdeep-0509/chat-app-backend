package repositories

import (
	"context"
	"errors"
	"time"

	"chat-app-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// UserRepository interface defines the methods for interacting with the user collection
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByUsername(ctx context.Context, username string) (*models.User, error)
	FindAll(ctx context.Context, search string) ([]*models.User, error)
	FindByID(ctx context.Context, id string) (*models.User, error)
	UpdateOnlineStatus(ctx context.Context, id string, isOnline bool) error
}

type userRepository struct {
	collection *mongo.Collection
}

// NewUserRepository creates a new instance of UserRepository
func NewUserRepository(db *mongo.Database) UserRepository {
	return &userRepository{
		collection: db.Collection("users"),
	}
}

// Create inserts a new user into the database
func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	_, err := r.collection.InsertOne(ctx, user)
	return err
}

// FindByEmail finds a user by their email address
func (r *userRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil // Return nil, nil when user is not found to distinguish from query errors
		}
		return nil, err
	}
	return &user, nil
}

// FindByUsername finds a user by their username
func (r *userRepository) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	err := r.collection.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil // Return nil, nil when user is not found
		}
		return nil, err
	}
	return &user, nil
}

// FindAll returns all users, optionally filtered by name, username, or email (case-insensitive regex search)
func (r *userRepository) FindAll(ctx context.Context, search string) ([]*models.User, error) {
	filter := bson.M{}
	if search != "" {
		regexPattern := primitive.Regex{Pattern: search, Options: "i"}
		filter = bson.M{
			"$or": []bson.M{
				{"name": regexPattern},
				{"username": regexPattern},
				{"email": regexPattern},
			},
		}
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	users := []*models.User{}
	if err = cursor.All(ctx, &users); err != nil {
		return nil, err
	}

	return users, nil
}

// FindByID finds a user by their string ObjectID
func (r *userRepository) FindByID(ctx context.Context, id string) (*models.User, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var user models.User
	err = r.collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// UpdateOnlineStatus updates the user's is_online and last_seen values in the DB
func (r *userRepository) UpdateOnlineStatus(ctx context.Context, id string, isOnline bool) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"is_online": isOnline,
			"last_seen": time.Now(),
		},
	}

	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": objID}, update)
	return err
}

