package websocket

import (
	"chat-app/internal/application/service"
	"chat-app/internal/infrastructure/config"
	"chat-app/internal/infrastructure/logger"
	"fmt"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WSMessage struct {
	ID         uint   `json:"id"`
	SenderID   uint   `json:"sender_id"`
	ReceiverID uint   `json:"receiver_id"`
	Content    string `json:"content"`
	IsGroup    bool   `json:"is_group"`
	IsRead     bool   `json:"is_read"`
	CreatedAt  string `json:"created_at"`
}

func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	tokenString := r.URL.Query().Get("token")
	if tokenString == "" {
		logger.Logger.Warn("websocket missing token", "remote_addr", r.RemoteAddr)
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	userID, username, err := ParseToken(tokenString)
	if err != nil {
		logger.Logger.Warn("websocket invalid token", "error", err, "remote_addr", r.RemoteAddr)
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Logger.Error("upgrade websocket failed", "error", err, "user_id", userID)
		return
	}

	logger.Logger.Info("websocket connected", "user_id", userID, "username", username)

	client := &Client{
		UserID:   userID,
		Username: username,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		Hub:      hub,
	}
	hub.register <- client

	go client.writePump()

	client.sendOfflineMessages()
	client.readPump()
}

func ParseToken(tokenString string) (userID uint, username string, err error) {
	if service.IsTokenBlacklisted(tokenString) {
		return 0, "", fmt.Errorf("invalid token")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return config.JWTSecret, nil
	})
	if err != nil || !token.Valid {
		return 0, "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, "", fmt.Errorf("invalid token claims")
	}

	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		return 0, "", fmt.Errorf("invalid user_id")
	}

	username, ok = claims["username"].(string)
	if !ok {
		return 0, "", fmt.Errorf("invalid username")
	}

	return uint(userIDFloat), username, nil
}
