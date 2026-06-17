package service

import (
	"avito-easy-report/internal/config"
	"avito-easy-report/internal/database"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret []byte

func InitAuth(cfg *config.Config) {
	jwtSecret = []byte(cfg.JWTSecret)
}

// Claims — JWT payload
type Claims struct {
	UserID   uint   `json:"userId"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// Register создаёт нового пользователя
func Register(username, password, role string) (*database.User, error) {
	if username == "" || password == "" {
		return nil, errors.New("имя пользователя и пароль обязательны")
	}
	if len(password) < 4 {
		return nil, errors.New("пароль должен быть не менее 4 символов")
	}

	// Проверка уникальности
	var existing database.User
	if err := database.DB.Where("username = ?", username).First(&existing).Error; err == nil {
		return nil, errors.New("пользователь с таким именем уже существует")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("ошибка хеширования пароля")
	}

	if role == "" {
		role = "guest"
	}

	user := database.User{
		Username: username,
		Password: string(hash),
		Role:     role,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		return nil, errors.New("ошибка создания пользователя: " + err.Error())
	}

	return &user, nil
}

// Login проверяет учётные данные и возвращает JWT-токен
func Login(username, password string) (string, *database.User, error) {
	var user database.User
	if err := database.DB.Where("username = ?", username).First(&user).Error; err != nil {
		return "", nil, errors.New("неверное имя пользователя или пароль")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", nil, errors.New("неверное имя пользователя или пароль")
	}

	token, err := generateToken(&user)
	if err != nil {
		return "", nil, errors.New("ошибка генерации токена")
	}

	return token, &user, nil
}

// generateToken создаёт JWT-токен
func generateToken(user *database.User) (string, error) {
	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(72 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ValidateToken проверяет и парсит JWT-токен
func ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("неподдерживаемый метод подписи")
		}
		return jwtSecret, nil
	})
	if err != nil {
		return nil, errors.New("недействительный токен")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("недействительный токен")
	}

	return claims, nil
}

// GetAllUsers возвращает всех пользователей (для админа)
func GetAllUsers() ([]database.User, error) {
	var users []database.User
	if err := database.DB.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// UpdateUserRole обновляет роль пользователя
func UpdateUserRole(userID uint, role string) error {
	if role != "admin" && role != "manager" && role != "guest" {
		return errors.New("недопустимая роль. Допустимые: admin, manager, guest")
	}
	return database.DB.Model(&database.User{}).Where("id = ?", userID).Update("role", role).Error
}

// DeleteUser удаляет пользователя
func DeleteUser(userID uint) error {
	// Не даём удалить последнего админа
	var count int64
	database.DB.Model(&database.User{}).Where("role = ?", "admin").Count(&count)
	var user database.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return errors.New("пользователь не найден")
	}
	if user.Role == "admin" && count <= 1 {
		return errors.New("нельзя удалить последнего администратора")
	}
	return database.DB.Delete(&database.User{}, userID).Error
}
