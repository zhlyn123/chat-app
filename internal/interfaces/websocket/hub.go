package websocket

import (
	"chat-app/internal/application/service"
	"chat-app/internal/domain/model"
	"chat-app/internal/infrastructure/logger"
	"encoding/json"
)

type Hub struct {
	clients    map[uint]*Client
	register   chan *Client
	unregister chan *Client
	direct     chan WSMessage
	group      chan WSMessage
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[uint]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		direct:     make(chan WSMessage, 256),
		group:      make(chan WSMessage, 256),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			if oldClient, ok := h.clients[client.UserID]; ok {
				logger.Logger.Info("replace old websocket connection", "user_id", client.UserID)
				close(oldClient.Send)
				oldClient.Conn.Close()
			}

			h.clients[client.UserID] = client
			service.UserOnline(client.UserID, client.Username)
			logger.Logger.Info("user online", "user_id", client.UserID, "username", client.Username)
			h.BroadcastOnlineUsers()

		case client := <-h.unregister:
			if current, ok := h.clients[client.UserID]; ok && current == client {
				delete(h.clients, client.UserID)
				close(client.Send)
				service.UserOffline(client.UserID)
				logger.Logger.Info("user offline", "user_id", client.UserID, "username", client.Username)
				h.BroadcastOnlineUsers()
			}

		case msg := <-h.direct:
			delivered := h.SendToUser(msg.ReceiverID, msg)
			PublishMessage(msg)

			if !delivered && !service.IsUserOnline(msg.ReceiverID) {
				service.IncrementUnreadMessage(msg.ReceiverID, msg.SenderID)
			}

		case msg := <-h.group:
			members := service.GetGroupMembers(msg.ReceiverID)
			for _, uid := range members {
				if uid == msg.SenderID {
					continue
				}
				delivered := h.SendToUser(uid, msg)
				if !delivered && !service.IsUserOnline(uid) {
					service.IncrementGroupUnreadMessage(uid, msg.ReceiverID)
				}
			}

			PublishMessage(msg)
		}
	}
}

func (h *Hub) BroadcastRevoke(msg model.Message) {
	payload := map[string]any{
		"type":        "revoke",
		"message_id":  msg.ID,
		"sender_id":   msg.SenderID,
		"receiver_id": msg.ReceiverID,
		"is_group":    msg.IsGroup,
	}

	if msg.IsGroup {
		for _, uid := range service.GetGroupMembers(msg.ReceiverID) {
			h.SendToUser(uid, payload)
		}
		return
	}

	h.SendToUser(msg.SenderID, payload)
	h.SendToUser(msg.ReceiverID, payload)
}

func (h *Hub) SendToUser(userID uint, msg any) bool {
	client, ok := h.clients[userID]
	if !ok {
		return false
	}

	data, err := json.Marshal(msg)
	if err != nil {
		logger.Logger.Error("marshal websocket message failed", "error", err, "user_id", userID)
		return false
	}

	select {
	case client.Send <- data:
		return true
	default:
		logger.Logger.Warn("client send queue full, closing connection", "user_id", userID)
		close(client.Send)
		delete(h.clients, client.UserID)
		client.Conn.Close()
		service.UserOffline(client.UserID)
		return false
	}
}

func (h *Hub) BroadcastOnlineUsers() {
	msg := map[string]any{
		"type":       "online",
		"onlineUser": service.GetOnlineUsers(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		logger.Logger.Error("marshal online users failed", "error", err)
		return
	}

	for userID, client := range h.clients {
		select {
		case client.Send <- data:
		default:
			logger.Logger.Warn("client send queue full while broadcasting online users", "user_id", userID)
			close(client.Send)
			delete(h.clients, userID)
			client.Conn.Close()
			service.UserOffline(userID)
		}
	}
}
