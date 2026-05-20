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

type MessageRepository interface {
	Create(ctx context.Context, msg *models.Message) error
	FindByConversationID(ctx context.Context, conversationID primitive.ObjectID, skip, limit int64) ([]*models.Message, error)
	FindLastMessage(ctx context.Context, conversationID primitive.ObjectID) (*models.Message, error)
	FindByID(ctx context.Context, id string) (*models.Message, error)
	UpdateContent(ctx context.Context, id string, content string) error
	Delete(ctx context.Context, id string) error
}

type messageRepository struct {
	collection *mongo.Collection
}

func NewMessageRepository(db *mongo.Database) MessageRepository {
	return &messageRepository{
		collection: db.Collection("messages"),
	}
}

func (r *messageRepository) Create(ctx context.Context, msg *models.Message) error {
	_, err := r.collection.InsertOne(ctx, msg)
	return err
}

func (r *messageRepository) FindByConversationID(ctx context.Context, conversationID primitive.ObjectID, skip, limit int64) ([]*models.Message, error) {
	filter := bson.M{
		"conversation_id": conversationID,
	}

	// Sort messages by created_at descending to get the most recent messages first for pagination
	opts := options.Find().
		SetSort(bson.M{"created_at": -1}).
		SetSkip(skip).
		SetLimit(limit)

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var list []*models.Message
	if err = cursor.All(ctx, &list); err != nil {
		return nil, err
	}

	// Reverse the list so the page's messages are returned in chronological (ascending) order
	for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
		list[i], list[j] = list[j], list[i]
	}

	return list, nil
}

func (r *messageRepository) FindLastMessage(ctx context.Context, conversationID primitive.ObjectID) (*models.Message, error) {
	filter := bson.M{
		"conversation_id": conversationID,
	}

	opts := options.FindOne().SetSort(bson.M{"created_at": -1})

	var msg models.Message
	err := r.collection.FindOne(ctx, filter, opts).Decode(&msg)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &msg, nil
}

func (r *messageRepository) FindByID(ctx context.Context, id string) (*models.Message, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var msg models.Message
	err = r.collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&msg)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &msg, nil
}

func (r *messageRepository) UpdateContent(ctx context.Context, id string, content string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"content": content,
		},
	}
	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": objID}, update)
	return err
}

func (r *messageRepository) Delete(ctx context.Context, id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": objID})
	return err
}
