package websocket

import (
	"chat-app/config"
	"fmt"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

// 升级器
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	}, //允许跨域
}

// 保存在线用户
var clients = make(map[uint]*websocket.Conn) //user_id -> websocket

// WebSocket 服务
func ServeWs(w http.ResponseWriter, r *http.Request) {
	//从URL获取token
	tokenString := r.URL.Query().Get("token")
	if tokenString == "" {
		http.Error(w, "缺少 token", http.StatusUnauthorized)
		return
	}
	//解析token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return config.JWTSecret, nil
	})
	if err != nil || !token.Valid {
		http.Error(w, "无效 token", http.StatusUnauthorized)
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		http.Error(w, "token claims 错误", http.StatusUnauthorized)
		return
	}
	userID := uint(claims["user_id"].(float64))
	username := claims["username"].(string)

	//升级http到websocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("升级 WebSocket 失败:", err)
		return
	}

	// 存储连接
	clients[userID] = conn
	fmt.Printf("用户 %s 已上线\n", username)

	// 接收消息循环
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Printf("用户 %s 已下线\n", username)
			delete(clients, userID)
			conn.Close()
			break
		}
		fmt.Printf("收到 %s 的消息: %s\n", username, string(msg))
		// TODO: 消息处理，广播给目标用户
	}
}
