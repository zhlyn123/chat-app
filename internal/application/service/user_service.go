package service

import (
	"chat-app/internal/domain/model"
	"chat-app/internal/infrastructure/config"
	"chat-app/internal/infrastructure/logger"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type LoginResponse struct {
	Token    string `json:"token"`
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
}

const onlineUsersRedisKey = "online_users"
const tokenBlacklistRedisPrefix = "jwt_blacklist:"

var onlineUsers = struct {
	sync.RWMutex
	users map[uint]string
}{
	users: make(map[uint]string),
}

var tokenBlacklist = struct {
	sync.RWMutex
	tokens map[string]time.Time
}{
	tokens: make(map[string]time.Time),
}

func IsUserOnline(userID uint) bool {
	if config.Redis != nil {
		ok, err := config.Redis.HExists(
			context.Background(),
			"online_users",
			strconv.FormatUint(uint64(userID), 10),
		).Result()

		if err == nil {
			return ok
		}
	}

	online := GetOnlineUsers()
	return online[userID] != ""
}

func UserOnline(userID uint, username string) {
	onlineUsers.Lock()
	onlineUsers.users[userID] = username
	onlineUsers.Unlock()

	if config.Redis != nil {
		config.Redis.HSet(context.Background(), onlineUsersRedisKey, strconv.FormatUint(uint64(userID), 10), username)
	}
}

func UserOffline(userID uint) {
	onlineUsers.Lock()
	delete(onlineUsers.users, userID)
	onlineUsers.Unlock()

	if config.Redis != nil {
		config.Redis.HDel(context.Background(), onlineUsersRedisKey, strconv.FormatUint(uint64(userID), 10))
	}
}

func GetOnlineUsers() map[uint]string {
	if config.Redis != nil {
		values, err := config.Redis.HGetAll(context.Background(), onlineUsersRedisKey).Result()
		if err == nil {
			users := make(map[uint]string, len(values))
			for id, username := range values {
				userID, err := strconv.ParseUint(id, 10, 64)
				if err != nil {
					continue
				}
				users[uint(userID)] = username
			}
			return users
		}
	}

	onlineUsers.RLock()
	defer onlineUsers.RUnlock()

	copyMap := make(map[uint]string, len(onlineUsers.users))
	for k, v := range onlineUsers.users {
		copyMap[k] = v
	}
	return copyMap
}

func BlacklistToken(tokenString string) error {
	expireAt, err := GetTokenExpireAt(tokenString)
	if err != nil {
		return err
	}

	if expireAt.Before(time.Now()) {
		return nil
	}

	key := tokenBlacklistKey(tokenString)
	tokenBlacklist.Lock()
	tokenBlacklist.tokens[key] = expireAt
	tokenBlacklist.Unlock()

	if config.Redis != nil {
		ttl := time.Until(expireAt)
		if ttl > 0 {
			return config.Redis.Set(context.Background(), tokenBlacklistRedisPrefix+key, "1", ttl).Err()
		}
	}
	return nil
}

func IsTokenBlacklisted(tokenString string) bool {
	key := tokenBlacklistKey(tokenString)
	if config.Redis != nil {
		ok, err := config.Redis.Exists(context.Background(), tokenBlacklistRedisPrefix+key).Result()
		if err == nil {
			return ok > 0
		}
	}

	tokenBlacklist.RLock()
	expireAt, ok := tokenBlacklist.tokens[key]
	tokenBlacklist.RUnlock()
	if !ok {
		return false
	}
	if time.Now().After(expireAt) {
		tokenBlacklist.Lock()
		delete(tokenBlacklist.tokens, key)
		tokenBlacklist.Unlock()
		return false
	}
	return true
}

func GetTokenExpireAt(tokenString string) (time.Time, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return config.JWTSecret, nil
	})
	if err != nil || !token.Valid {
		return time.Time{}, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return time.Time{}, errors.New("invalid token claims")
	}

	expFloat, ok := claims["exp"].(float64)
	if !ok {
		return time.Time{}, errors.New("invalid token claims")
	}
	return time.Unix(int64(expFloat), 0), nil
}

func tokenBlacklistKey(tokenString string) string {
	sum := sha256.Sum256([]byte(tokenString))
	return hex.EncodeToString(sum[:])
}

func Register(username, password string) error {
	var user model.User
	if err := config.DB.Where("username = ?", username).First(&user).Error; err == nil {
		return errors.New("username already exists")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	newUser := model.User{
		Username: username,
		Password: string(hashedPassword),
	}
	if err := config.DB.Create(&newUser).Error; err != nil {
		return err
	}

	return nil
}

func Login(username, password string) (*LoginResponse, error) {
	var user model.User
	if err := config.DB.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Logger.Warn("login failed: user not found", "username", username)
			return nil, errors.New("user not found")
		}
		logger.Logger.Error("login query failed", "error", err, "username", username)
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		logger.Logger.Warn("login failed: wrong password", "username", username)
		return nil, errors.New("wrong password")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"exp":      time.Now().Add(config.TokenExpireTime).Unix(),
	})

	tokenString, err := token.SignedString(config.JWTSecret)
	if err != nil {
		logger.Logger.Error("sign jwt failed", "error", err, "user_id", user.ID)
		return nil, err
	}

	logger.Logger.Info("login success", "user_id", user.ID, "username", user.Username)

	return &LoginResponse{
		Token:    tokenString,
		UserID:   user.ID,
		Username: user.Username,
	}, nil
}

func GetUserInfo(userID uint) (*model.User, error) {
	var user model.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

func GetALLUsers(meID uint) ([]model.User, error) {
	var users []model.User
	if err := config.DB.Select("id", "username", "created_at").Where("id != ?", meID).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}
