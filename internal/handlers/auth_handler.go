package handlers

import (
	"net/http"

	"eventplanner-backend/internal/models"
	"eventplanner-backend/internal/services"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	users services.UserService
}

func NewAuthHandler(users services.UserService) *AuthHandler {
	return &AuthHandler{users: users}
}

func (h *AuthHandler) Signup(c *gin.Context) {
	var req models.SignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, err := h.users.Signup(c, req.Name, req.Email, req.Password)
	if err != nil {
		status := http.StatusInternalServerError
		if err == services.ErrUserExists {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "User created successfully",
		"user": gin.H{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
		},
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, err := h.users.Login(c, req.Email, req.Password)
	if err != nil {
		status := http.StatusUnauthorized
		if err != services.ErrInvalidCredentials {
			status = http.StatusInternalServerError
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token": "mock-jwt-token",
		"id":    user.ID,
		"name":  user.Name,
		"email": user.Email,
	})
}

func (h *AuthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
