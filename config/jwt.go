package config

import (
	"time"
)

// 密钥
var JWTSecret = []byte("chat-app-secret")

// token有效期
const TokenExpireTime = time.Hour * 24
