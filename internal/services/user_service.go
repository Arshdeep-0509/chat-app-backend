package services

import (
	"context"
	"errors"
	"time"

	"chat-app-backend/internal/models"
	"chat-app-backend/internal/repositories"
	"chat-app-backend/internal/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserService interface defines the methods for user business logic
type UserService interface {
	Register(ctx context.Context, req *models.RegisterRequest) (*models.User, error)
	Login(ctx context.Context, req *models.LoginRequest) (string, error)
	ListUsers(ctx context.Context, search string) ([]*models.User, error)
	GetUserProfile(ctx context.Context, userID string) (*models.User, error)
}

type userService struct {
	userRepo repositories.UserRepository
}

// NewUserService creates a new instance of UserService
func NewUserService(userRepo repositories.UserRepository) UserService {
	return &userService{
		userRepo: userRepo,
	}
}

// ListUsers retrieves all users matching the search query
func (s *userService) ListUsers(ctx context.Context, search string) ([]*models.User, error) {
	return s.userRepo.FindAll(ctx, search)
}

// GetUserProfile retrieves complete information about a user by ID
func (s *userService) GetUserProfile(ctx context.Context, userID string) (*models.User, error) {
	return s.userRepo.FindByID(ctx, userID)
}

// Register creates a new user account
func (s *userService) Register(ctx context.Context, req *models.RegisterRequest) (*models.User, error) {
	// Check if email already exists
	existingUserByEmail, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if existingUserByEmail != nil {
		return nil, errors.New("email already in use")
	}

	// Check if username already exists
	existingUserByUsername, err := s.userRepo.FindByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if existingUserByUsername != nil {
		return nil, errors.New("username already in use")
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// Create user model
	now := time.Now()
	user := &models.User{
		ID:             primitive.NewObjectID(),
		Name:           req.Name,
		Username:       req.Username,
		Email:          req.Email,
		Password:       hashedPassword,
		ProfilePicture: "", // Default empty
		Bio:            "", // Default empty
		IsOnline:       true, // User is online upon registering usually or explicitly login
		LastSeen:       now,
		BlockedUsers:   []primitive.ObjectID{}, // Empty slice
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// Save to DB
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// Login authenticates a user and returns a JWT
func (s *userService) Login(ctx context.Context, req *models.LoginRequest) (string, error) {
	// Find user by email
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", errors.New("invalid email or password")
	}

	// Check password
	if !utils.CheckPasswordHash(req.Password, user.Password) {
		return "", errors.New("invalid email or password")
	}

	// Generate JWT
	token, err := utils.GenerateToken(user.ID.Hex())
	if err != nil {
		return "", err
	}

	return token, nil
}
