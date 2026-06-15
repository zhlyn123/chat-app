package websocket

import (
	"chat-app/internal/application/service"
	"chat-app/internal/infrastructure/logger"
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = 50 * time.Second
	maxMessageSize = 4096
)

type Client struct {
	UserID   uint
	Username string
	Conn     *websocket.Conn
	Send     chan []byte
	Hub      *Hub
}

func (c *Client) readPump() {
	defer func() {
		logger.Logger.Info("websocket read pump closed", "user_id", c.UserID)
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		var msg WSMessage
		if err := c.Conn.ReadJSON(&msg); err != nil {
			logger.Logger.Info("websocket read failed", "user_id", c.UserID, "error", err)
			break
		}

		msg.SenderID = c.UserID

		savedMsg, err := service.SaveMessage(
			c.UserID,
			msg.ReceiverID,
			msg.Content,
			msg.IsGroup,
		)
		if err != nil {
			logger.Logger.Error(
				"save message failed",
				"error", err,
				"sender_id", c.UserID,
				"receiver_id", msg.ReceiverID,
				"is_group", msg.IsGroup,
				"content_len", len(msg.Content),
			)
			continue
		}

		msg.ID = savedMsg.ID
		msg.IsRead = savedMsg.IsRead
		msg.CreatedAt = savedMsg.CreatedAt.Format(time.RFC3339Nano)
		if msg.IsGroup {
			c.Hub.group <- msg
		} else {
			c.Hub.direct <- msg
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				logger.Logger.Info("websocket write failed", "user_id", c.UserID, "error", err)
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Logger.Info("websocket ping failed", "user_id", c.UserID, "error", err)
				return
			}
		}
	}
}

func (c *Client) sendOfflineMessages() {
	messages, err := service.GetUnreadMessages(c.UserID)
	if err != nil {
		logger.Logger.Error("get unread messages failed", "error", err, "user_id", c.UserID)
		return
	}

	for _, msg := range messages {
		data, err := json.Marshal(msg)
		if err != nil {
			logger.Logger.Error("marshal offline message failed", "error", err, "user_id", c.UserID, "message_id", msg.ID)
			continue
		}

		select {
		case c.Send <- data:
		default:
			logger.Logger.Warn("offline message queue full", "user_id", c.UserID)
			return
		}
	}
}
