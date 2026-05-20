package repositories

import (
	"context"
	"errors"

	"chat-app-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ChatRequestRepository interface {
	Create(ctx context.Context, req *models.ChatRequest) error
	FindByID(ctx context.Context, id string) (*models.ChatRequest, error)
	FindPending(ctx context.Context, senderID, receiverID primitive.ObjectID) (*models.ChatRequest, error)
	FindList(ctx context.Context, userID primitive.ObjectID, direction string) ([]*models.ChatRequest, error)
	UpdateStatus(ctx context.Context, id primitive.ObjectID, status string, updatedAt int64) error
}

type chatRequestRepository struct {
	collection *mongo.Collection
}

func NewChatRequestRepository(db *mongo.Database) ChatRequestRepository {
	return &chatRequestRepository{
		collection: db.Collection("chat_requests"),
	}
}

func (r *chatRequestRepository) Create(ctx context.Context, req *models.ChatRequest) error {
	_, err := r.collection.InsertOne(ctx, req)
	return err
}

func (r *chatRequestRepository) FindByID(ctx context.Context, id string) (*models.ChatRequest, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var req models.ChatRequest
	err = r.collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&req)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &req, nil
}

func (r *chatRequestRepository) FindPending(ctx context.Context, senderID, receiverID primitive.ObjectID) (*models.ChatRequest, error) {
	filter := bson.M{
		"sender_id":   senderID,
		"receiver_id": receiverID,
		"status":      "pending",
	}

	var req models.ChatRequest
	err := r.collection.FindOne(ctx, filter).Decode(&req)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &req, nil
}

func (r *chatRequestRepository) FindList(ctx context.Context, userID primitive.ObjectID, direction string) ([]*models.ChatRequest, error) {
	filter := bson.M{}
	if direction == "outgoing" {
		filter["sender_id"] = userID
	} else { // default to incoming
		filter["receiver_id"] = userID
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var list []*models.ChatRequest
	if err = cursor.All(ctx, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func (r *chatRequestRepository) UpdateStatus(ctx context.Context, id primitive.ObjectID, status string, updatedAt int64) error {
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": updatedAt,
		},
	}
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}
