package handler

import (
	"avito-easy-report/internal/middleware"
	"avito-easy-report/internal/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// AuthHandler — обработчики аутентификации и управления пользователями
type AuthHandler struct{}

// NewAuthHandler создаёт AuthHandler
func NewAuthHandler() *AuthHandler {
	return &AuthHandler{}
}

// --- Структуры запросов/ответов ---

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type registerRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role"`
}

type authResponse struct {
	Token    string `json:"token"`
	UserID   uint   `json:"userId"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// Login — POST /api/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "имя пользователя и пароль обязательны"})
		return
	}

	token, user, err := service.Login(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, authResponse{
		Token:    token,
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
	})
}

// Register — POST /api/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "имя пользователя и пароль обязательны"})
		return
	}

	// Гости могут создавать только guest-аккаунты
	claims := middleware.GetClaims(c)
	if claims != nil && claims.Role != "admin" {
		req.Role = "guest"
	}
	if req.Role == "" {
		req.Role = "guest"
	}
	// Только админ может создавать admin/manager
	if req.Role != "guest" && (claims == nil || claims.Role != "admin") {
		req.Role = "guest"
	}

	user, err := service.Register(req.Username, req.Password, req.Role)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"role":     user.Role,
	})
}

// Me — GET /api/auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "не авторизован"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"userId":   claims.UserID,
		"username": claims.Username,
		"role":     claims.Role,
	})
}

// --- Admin: управление пользователями ---

// ListUsers — GET /api/admin/users
func (h *AuthHandler) ListUsers(c *gin.Context) {
	users, err := service.GetAllUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка получения пользователей"})
		return
	}

	type userResponse struct {
		ID       uint   `json:"id"`
		Username string `json:"username"`
		Role     string `json:"role"`
	}

	result := make([]userResponse, len(users))
	for i, u := range users {
		result[i] = userResponse{
			ID:       u.ID,
			Username: u.Username,
			Role:     u.Role,
		}
	}

	c.JSON(http.StatusOK, result)
}

type updateRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

// UpdateUserRole — PUT /api/admin/users/:id/role
func (h *AuthHandler) UpdateUserRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный ID пользователя"})
		return
	}

	var req updateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "роль обязательна"})
		return
	}

	if err := service.UpdateUserRole(uint(id), req.Role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "роль обновлена"})
}

// DeleteUser — DELETE /api/admin/users/:id
func (h *AuthHandler) DeleteUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный ID пользователя"})
		return
	}

	if err := service.DeleteUser(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "пользователь удалён"})
}
