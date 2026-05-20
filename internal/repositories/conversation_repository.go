package repositories

import (
	"context"
	"errors"

	"chat-app-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ConversationRepository interface {
	Create(ctx context.Context, conv *models.Conversation) error
	FindByID(ctx context.Context, id string) (*models.Conversation, error)
	FindBetweenParticipants(ctx context.Context, user1, user2 primitive.ObjectID) (*models.Conversation, error)
	FindAllForUser(ctx context.Context, userID primitive.ObjectID) ([]*models.Conversation, error)
	UpdateTimestamp(ctx context.Context, id primitive.ObjectID, updatedAt int64) error
}

type conversationRepository struct {
	collection *mongo.Collection
}

func NewConversationRepository(db *mongo.Database) ConversationRepository {
	return &conversationRepository{
		collection: db.Collection("conversations"),
	}
}

func (r *conversationRepository) Create(ctx context.Context, conv *models.Conversation) error {
	_, err := r.collection.InsertOne(ctx, conv)
	return err
}

func (r *conversationRepository) FindByID(ctx context.Context, id string) (*models.Conversation, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var conv models.Conversation
	err = r.collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&conv)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &conv, nil
}

func (r *conversationRepository) FindBetweenParticipants(ctx context.Context, user1, user2 primitive.ObjectID) (*models.Conversation, error) {
	filter := bson.M{
		"participants": bson.M{
			"$all": []primitive.ObjectID{user1, user2},
		},
	}

	var conv models.Conversation
	err := r.collection.FindOne(ctx, filter).Decode(&conv)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &conv, nil
}

func (r *conversationRepository) FindAllForUser(ctx context.Context, userID primitive.ObjectID) ([]*models.Conversation, error) {
	filter := bson.M{
		"participants": userID,
	}

	// Sort conversations by updated_at descending to show latest conversations first
	opts := options.Find().SetSort(bson.M{"updated_at": -1})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var list []*models.Conversation
	if err = cursor.All(ctx, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func (r *conversationRepository) UpdateTimestamp(ctx context.Context, id primitive.ObjectID, updatedAt int64) error {
	update := bson.M{
		"$set": bson.M{
			"updated_at": updatedAt,
		},
	}
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}
