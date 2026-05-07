package service

import (
	"chat-app/config"
	"chat-app/model"
	"errors"
	"time"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func Register(username, password string) error {
	//检查用户是否存在
	var user model.User
	if err := config.DB.Where("username = ?", username).First(&user).Error; err == nil {
		return errors.New("用户名已存在")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	//密码加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	//保存用户
	newUser := model.User{
		Username: username,
		Password: string(hashedPassword),
	}
	if err := config.DB.Create(&newUser).Error; err != nil {
		return err
	}

	return nil
}

func Login(username, password string) (string, error) {
	//用户是否存在
	var user model.User
    if err := config.DB.Where("username = ?", username).First(&user).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return "", errors.New("用户不存在")
        }
        return "", err
    }
	//验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", errors.New("密码错误")
	}
	//生成jwt
	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"username": user.Username,
		"exp": time.Now().Add(config.TokenExpireTime).Unix(),
	})

	tokenString, err := token.SignedString(config.JWTSecret)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}
