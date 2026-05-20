package services

import (
	"context"
	"errors"
	"time"

	"chat-app-backend/internal/models"
	"chat-app-backend/internal/repositories"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ChatRequestService interface {
	SendRequest(ctx context.Context, senderIDHex, receiverIDHex string) (*models.ChatRequest, error)
	ListRequests(ctx context.Context, userIDHex, direction string) ([]*models.ChatRequest, error)
	RespondRequest(ctx context.Context, requestIDHex, receiverIDHex, status string) (*models.ChatRequest, error)
}

type chatRequestService struct {
	chatRequestRepo repositories.ChatRequestRepository
	conversationRepo repositories.ConversationRepository
	userRepo         repositories.UserRepository
}

func NewChatRequestService(
	chatRequestRepo repositories.ChatRequestRepository,
	conversationRepo repositories.ConversationRepository,
	userRepo repositories.UserRepository,
) ChatRequestService {
	return &chatRequestService{
		chatRequestRepo:  chatRequestRepo,
		conversationRepo: conversationRepo,
		userRepo:         userRepo,
	}
}

func (s *chatRequestService) SendRequest(ctx context.Context, senderIDHex, receiverIDHex string) (*models.ChatRequest, error) {
	senderID, err := primitive.ObjectIDFromHex(senderIDHex)
	if err != nil {
		return nil, errors.New("invalid sender ID")
	}

	receiverID, err := primitive.ObjectIDFromHex(receiverIDHex)
	if err != nil {
		return nil, errors.New("invalid receiver ID")
	}

	if senderID == receiverID {
		return nil, errors.New("cannot send chat request to yourself")
	}

	// 1. Verify receiver exists
	receiver, err := s.userRepo.FindByID(ctx, receiverIDHex)
	if err != nil {
		return nil, err
	}
	if receiver == nil {
		return nil, errors.New("receiver user not found")
	}

	// 2. Check if active conversation already exists
	conv, err := s.conversationRepo.FindBetweenParticipants(ctx, senderID, receiverID)
	if err != nil {
		return nil, err
	}
	if conv != nil {
		return nil, errors.New("a conversation already exists between these users")
	}

	// 3. Check if there is already a pending request from sender to receiver
	pendingOutgoing, err := s.chatRequestRepo.FindPending(ctx, senderID, receiverID)
	if err != nil {
		return nil, err
	}
	if pendingOutgoing != nil {
		return nil, errors.New("chat request already pending")
	}

	// 4. Check if there is already a pending request from receiver to sender (incoming)
	pendingIncoming, err := s.chatRequestRepo.FindPending(ctx, receiverID, senderID)
	if err != nil {
		return nil, err
	}
	if pendingIncoming != nil {
		return nil, errors.New("the other user has already sent you a chat request. check your requests list")
	}

	// 5. Create request
	now := time.Now().UnixMilli()
	req := &models.ChatRequest{
		ID:         primitive.NewObjectID(),
		SenderID:   senderID,
		ReceiverID: receiverID,
		Status:     "pending",
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.chatRequestRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	// Populate receiver info for response convenience
	req.Receiver = receiver
	return req, nil
}

func (s *chatRequestService) ListRequests(ctx context.Context, userIDHex, direction string) ([]*models.ChatRequest, error) {
	userID, err := primitive.ObjectIDFromHex(userIDHex)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	requests, err := s.chatRequestRepo.FindList(ctx, userID, direction)
	if err != nil {
		return nil, err
	}

	// Populate profiles
	for _, req := range requests {
		if direction == "outgoing" {
			receiver, err := s.userRepo.FindByID(ctx, req.ReceiverID.Hex())
			if err == nil && receiver != nil {
				req.Receiver = receiver
			}
		} else {
			sender, err := s.userRepo.FindByID(ctx, req.SenderID.Hex())
			if err == nil && sender != nil {
				req.Sender = sender
			}
		}
	}

	return requests, nil
}

func (s *chatRequestService) RespondRequest(ctx context.Context, requestIDHex, receiverIDHex, status string) (*models.ChatRequest, error) {
	requestID, err := primitive.ObjectIDFromHex(requestIDHex)
	if err != nil {
		return nil, errors.New("invalid request ID")
	}

	receiverID, err := primitive.ObjectIDFromHex(receiverIDHex)
	if err != nil {
		return nil, errors.New("invalid receiver ID")
	}

	// 1. Find Chat Request
	req, err := s.chatRequestRepo.FindByID(ctx, requestIDHex)
	if err != nil {
		return nil, err
	}
	if req == nil {
		// Try to look up by Sender ID (requestID) and Receiver ID (receiverID)
		req, err = s.chatRequestRepo.FindPending(ctx, requestID, receiverID)
		if err != nil {
			return nil, err
		}
		if req == nil {
			return nil, errors.New("chat request not found")
		}
		// Update requestID to match the actual ChatRequest document ID
		requestID = req.ID
	}

	// 2. Validate receiver
	if req.ReceiverID != receiverID {
		return nil, errors.New("unauthorized to respond to this request")
	}

	// 3. Check status is pending
	if req.Status != "pending" {
		return nil, errors.New("chat request has already been processed")
	}

	// 4. Update status
	now := time.Now().UnixMilli()
	if err := s.chatRequestRepo.UpdateStatus(ctx, requestID, status, now); err != nil {
		return nil, err
	}
	req.Status = status
	req.UpdatedAt = now

	// 5. If accepted, create conversation
	if status == "accepted" {
		existingConv, err := s.conversationRepo.FindBetweenParticipants(ctx, req.SenderID, req.ReceiverID)
		if err != nil {
			return nil, err
		}
		if existingConv == nil {
			conv := &models.Conversation{
				ID:           primitive.NewObjectID(),
				Participants: []primitive.ObjectID{req.SenderID, req.ReceiverID},
				CreatedAt:    now,
				UpdatedAt:    now,
			}
			if err := s.conversationRepo.Create(ctx, conv); err != nil {
				return nil, err
			}
		}
	}

	// Populate sender details for return message
	sender, err := s.userRepo.FindByID(ctx, req.SenderID.Hex())
	if err == nil && sender != nil {
		req.Sender = sender
	}

	return req, nil
}
