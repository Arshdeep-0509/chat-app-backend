package controllers

import (
	"net/http"

	"chat-app-backend/internal/models"
	"chat-app-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// UserController handles user-related HTTP requests
type UserController struct {
	userService services.UserService
}

// NewUserController creates a new instance of UserController
func NewUserController(userService services.UserService) *UserController {
	return &UserController{
		userService: userService,
	}
}

// Register handles user registration
func (ctrl *UserController) Register(c *gin.Context) {
	var req models.RegisterRequest

	// Validate request body
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Call service to register user
	user, err := ctrl.userService.Register(c.Request.Context(), &req)
	if err != nil {
		// Differentiate between validation errors and internal errors if needed
		if err.Error() == "email already in use" || err.Error() == "username already in use" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register user: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully",
		"user":    user,
	})
}

// Login handles user authentication
func (ctrl *UserController) Login(c *gin.Context) {
	var req models.LoginRequest

	// Validate request body
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Call service to login
	token, err := ctrl.userService.Login(c.Request.Context(), &req)
	if err != nil {
		if err.Error() == "invalid email or password" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to login: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"token":   token,
	})
}

// ListUsers handles retrieving the list of all users, optionally filtered by a search query
func (ctrl *UserController) ListUsers(c *gin.Context) {
	search := c.Query("search")

	userIDVal, exists := c.Get("user_id")
	var loggedInUserID string
	if exists {
		if idStr, ok := userIDVal.(string); ok {
			loggedInUserID = idStr
		}
	}

	users, err := ctrl.userService.ListUsers(c.Request.Context(), loggedInUserID, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}

// GetProfile handles retrieving the full profile details of the currently authenticated user
func (ctrl *UserController) GetProfile(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userID, ok := userIDVal.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID type in context"})
		return
	}

	user, err := ctrl.userService.GetUserProfile(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get profile: " + err.Error()})
		return
	}

	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

