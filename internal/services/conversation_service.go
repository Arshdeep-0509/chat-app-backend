package services

import (
	"context"
	"errors"

	"chat-app-backend/internal/models"
	"chat-app-backend/internal/repositories"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ConversationService interface {
	ListConversations(ctx context.Context, userIDHex string) ([]*models.Conversation, error)
	GetMessages(ctx context.Context, conversationIDHex, userIDHex string, page, limit int) ([]*models.Message, error)
	EditMessage(ctx context.Context, messageIDHex, content, userIDHex string) (*models.Message, error)
	DeleteMessage(ctx context.Context, messageIDHex, userIDHex string) (*models.Message, error)
}

type conversationService struct {
	conversationRepo repositories.ConversationRepository
	messageRepo      repositories.MessageRepository
	userRepo         repositories.UserRepository
}

func NewConversationService(
	conversationRepo repositories.ConversationRepository,
	messageRepo repositories.MessageRepository,
	userRepo repositories.UserRepository,
) ConversationService {
	return &conversationService{
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		userRepo:         userRepo,
	}
}

func (s *conversationService) ListConversations(ctx context.Context, userIDHex string) ([]*models.Conversation, error) {
	userID, err := primitive.ObjectIDFromHex(userIDHex)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	conversations, err := s.conversationRepo.FindAllForUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	for _, conv := range conversations {
		// Populate participants details
		var participantDetails []*models.User
		for _, partID := range conv.Participants {
			// Optional: we can skip the current user or include both.
			// Including all participants (with their statuses) is standard.
			user, err := s.userRepo.FindByID(ctx, partID.Hex())
			if err == nil && user != nil {
				participantDetails = append(participantDetails, user)
			}
		}
		conv.ParticipantDetails = participantDetails

		// Populate last message
		lastMsg, err := s.messageRepo.FindLastMessage(ctx, conv.ID)
		if err == nil && lastMsg != nil {
			conv.LastMessage = lastMsg
		}
	}

	return conversations, nil
}

func (s *conversationService) GetMessages(ctx context.Context, conversationIDHex, userIDHex string, page, limit int) ([]*models.Message, error) {
	convID, err := primitive.ObjectIDFromHex(conversationIDHex)
	if err != nil {
		return nil, errors.New("invalid conversation ID")
	}

	userID, err := primitive.ObjectIDFromHex(userIDHex)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	// 1. Fetch conversation
	conv, err := s.conversationRepo.FindByID(ctx, conversationIDHex)
	if err != nil {
		return nil, err
	}
	if conv == nil {
		return nil, errors.New("conversation not found")
	}

	// 2. Validate participant membership
	isParticipant := false
	for _, p := range conv.Participants {
		if p == userID {
			isParticipant = true
			break
		}
	}
	if !isParticipant {
		return nil, errors.New("unauthorized access to conversation messages")
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	skip := int64((page - 1) * limit)
	limit64 := int64(limit)

	// 3. Fetch message list
	return s.messageRepo.FindByConversationID(ctx, convID, skip, limit64)
}

func (s *conversationService) EditMessage(ctx context.Context, messageIDHex, content, userIDHex string) (*models.Message, error) {
	msg, err := s.messageRepo.FindByID(ctx, messageIDHex)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, errors.New("message not found")
	}

	userObjID, err := primitive.ObjectIDFromHex(userIDHex)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	if msg.SenderID != userObjID {
		return nil, errors.New("unauthorized to edit this message")
	}

	if err := s.messageRepo.UpdateContent(ctx, messageIDHex, content); err != nil {
		return nil, err
	}

	msg.Content = content
	return msg, nil
}

func (s *conversationService) DeleteMessage(ctx context.Context, messageIDHex, userIDHex string) (*models.Message, error) {
	msg, err := s.messageRepo.FindByID(ctx, messageIDHex)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, errors.New("message not found")
	}

	userObjID, err := primitive.ObjectIDFromHex(userIDHex)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	if msg.SenderID != userObjID {
		return nil, errors.New("unauthorized to delete this message")
	}

	if err := s.messageRepo.Delete(ctx, messageIDHex); err != nil {
		return nil, err
	}

	return msg, nil
}
