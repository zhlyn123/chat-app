package websocket

import (
	"chat-app/internal/application/service"
	"chat-app/internal/infrastructure/config"
	"chat-app/internal/infrastructure/logger"
	"context"
	"encoding/json"
	"os"

	"github.com/google/uuid"
)

const redisChatMessageChannel = "chat-message"

var InstanceID = loadInstanceID()

func loadInstanceID() string {
	id := os.Getenv("INSTANCE_ID")
	if id != "" {
		return id
	}
	return uuid.NewString()
}

type PubSubMesssage struct {
	Origin  string    `json:"origin"`
	Message WSMessage `json:"message"`
}

func PublishMessage(msg WSMessage) {
	if config.Redis == nil {
		return
	}

	event := PubSubMesssage{
		Origin:  InstanceID,
		Message: msg,
	}

	data, err := json.Marshal(event)
	if err != nil {
		logger.Logger.Error("marshal pubsub message failed", "error", err)
		return
	}

	if err := config.Redis.Publish(
		context.Background(),
		redisChatMessageChannel,
		data,
	).Err(); err != nil {
		logger.Logger.Error("publish chat message failed", "error", err)
	}
}

func SubscribeMessages(hub *Hub) {
	if config.Redis == nil {
		logger.Logger.Warn("redis unavailable, pubsub disabled")
		return
	}

	pubsub := config.Redis.Subscribe(context.Background(), redisChatMessageChannel)
	defer pubsub.Close()

	ch := pubsub.Channel()

	for redisMsg := range ch {
		var event PubSubMesssage
		if err := json.Unmarshal([]byte(redisMsg.Payload), &event); err != nil {
			logger.Logger.Error("unmarshal pubsub message failed", "error", err)
			continue
		}

		if event.Origin == InstanceID {
			continue
		}

		msg := event.Message

		if msg.IsGroup {
			deliverGroupMessageFromPubSub(hub, msg)
		} else {
			deliverDirectMessageFromPubSub(hub, msg)
		}
	}
}

func deliverDirectMessageFromPubSub(hub *Hub, msg WSMessage) {
	ok := hub.SendToUser(msg.ReceiverID, msg)
	if ok {
		logger.Logger.Info("pubsub direct message delivered",
			"sender_id", msg.SenderID,
			"receiver_id", msg.ReceiverID,
		)
	}
}

func deliverGroupMessageFromPubSub(hub *Hub, msg WSMessage) {
	members := service.GetGroupMembers(msg.ReceiverID)

	for _, uid := range members {
		if uid == msg.SenderID {
			continue
		}

		hub.SendToUser(uid, msg)
	}
}
