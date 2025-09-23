package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/last-emo-boy/infra-core/pkg/auth"
	"github.com/last-emo-boy/infra-core/pkg/database"
)

// UserHandler handles user-related API endpoints
type UserHandler struct {
	auth *auth.Auth
	db   *database.DB
}

// NewUserHandler creates a new UserHandler
func NewUserHandler(auth *auth.Auth, db *database.DB) *UserHandler {
	return &UserHandler{
		auth: auth,
		db:   db,
	}
}

// RegisterRequest represents user registration data
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role,omitempty"`
}

// Register creates a new user account
func (h *UserHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default role is 'user'
	if req.Role == "" {
		req.Role = "user"
	}

	// Hash password
	hashedPassword, err := h.auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password"})
		return
	}

	// Create user
	user := &database.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Role:         req.Role,
	}

	repo := h.db.UserRepository()
	if err := repo.Create(user); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Username or email already exists"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "User created successfully",
		"user_id":  user.ID,
		"username": user.Username,
		"role":     user.Role,
	})
}

// Login authenticates a user and returns a JWT token
func (h *UserHandler) Login(c *gin.Context) {
	var req auth.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find user by username
	repo := h.db.UserRepository()
	user, err := repo.GetByUsername(req.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Check password
	if err := h.auth.CheckPassword(req.Password, user.PasswordHash); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Create SSO session
	_, sessionHash, err := h.auth.GenerateSessionToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}

	// Store session in database
	ssoSession := &database.SSOSession{
		UserID:    user.ID,
		TokenHash: sessionHash,
		ExpiresAt: time.Now().Add(24 * time.Hour), // 24 hour session
		IPAddress: c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
		IsActive:  true,
		LastUsed:  time.Now(),
	}

	sessionRepo := h.db.SSOSessionRepository()
	if err := sessionRepo.Create(ssoSession); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}

	// Get user services for token
	permRepo := h.db.UserServicePermissionRepository()
	userServices, err := permRepo.ListUserServices(user.ID)
	if err != nil {
		userServices = []*database.RegisteredService{} // Empty on error
	}

	services := make([]string, len(userServices))
	for i, service := range userServices {
		services[i] = service.Name
	}

	// Generate JWT token with session and services
	token, expiresAt, err := h.auth.GenerateTokenWithSession(
		user.ID, 
		user.Username, 
		user.Role, 
		ssoSession.ID,
		[]string{}, // permissions - could be enhanced later
		services,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Update last login
	if err := repo.UpdateLastLogin(user.ID); err != nil {
		// Log error but don't fail the login
		fmt.Printf("Failed to update last login for user %d: %v\n", user.ID, err)
	}

	response := auth.LoginResponse{
		Token:     token,
		UserID:    user.ID,
		Username:  user.Username,
		Role:      user.Role,
		ExpiresAt: expiresAt,
	}

	c.JSON(http.StatusOK, response)
}

// GetProfile returns the current user's profile
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	repo := h.db.UserRepository()
	user, err := repo.GetByID(userID.(int))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":    user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"role":       user.Role,
		"created_at": user.CreatedAt,
		"last_login": user.LastLogin,
	})
}

// ListUsers returns a list of all users (admin only)
func (h *UserHandler) ListUsers(c *gin.Context) {
	repo := h.db.UserRepository()
	users, err := repo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	// Remove sensitive data
	var userList []gin.H
	for _, user := range users {
		userList = append(userList, gin.H{
			"user_id":    user.ID,
			"username":   user.Username,
			"email":      user.Email,
			"role":       user.Role,
			"created_at": user.CreatedAt,
			"last_login": user.LastLogin,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"users": userList,
		"total": len(userList),
	})
}

// UpdateUser updates user information (admin or self)
func (h *UserHandler) UpdateUser(c *gin.Context) {
	userIDParam := c.Param("id")
	targetUserID, err := strconv.Atoi(userIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	currentUserID := c.GetInt("user_id")
	currentUserRole := c.GetString("role")

	// Check permissions: admin can update anyone, users can only update themselves
	if currentUserRole != "admin" && currentUserID != targetUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}

	var req struct {
		Email string `json:"email,omitempty"`
		Role  string `json:"role,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	repo := h.db.UserRepository()
	user, err := repo.GetByID(targetUserID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Update fields
	if req.Email != "" {
		user.Email = req.Email
	}

	// Only admin can change roles
	if req.Role != "" && currentUserRole == "admin" {
		user.Role = req.Role
	}

	if err := repo.Update(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "User updated successfully",
		"user_id":  user.ID,
		"username": user.Username,
		"email":    user.Email,
		"role":     user.Role,
	})
}

// DeleteUser deletes a user account (admin only)
func (h *UserHandler) DeleteUser(c *gin.Context) {
	userIDParam := c.Param("id")
	userID, err := strconv.Atoi(userIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	repo := h.db.UserRepository()
	if err := repo.Delete(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}
