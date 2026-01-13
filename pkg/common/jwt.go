package common

import (
	"BackendTemplate/pkg/logger"
	"crypto/rand"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var JwtKey []byte

func init() {
	var err error
	JwtKey, err = generateSecureKey(32)
	if err != nil {
		logger.Error(err.Error())
	}
}

// Claims 结构体定义 JWT 的负载
type Claims struct {
	Username string `json:"username"`
	UID      string `json:"uid,omitempty"`     // 新增：用户ID
	Purpose  string `json:"purpose,omitempty"` // 新增：token用途
	jwt.RegisteredClaims
}

// 生成普通JWT
func GenerateJWT(username string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JwtKey)
}

// 生成带额外信息的JWT（用于WebSocket等特殊场景）
func GenerateJWTWithExtras(username, uid, purpose string, expiresIn time.Duration) (string, error) {
	expirationTime := time.Now().Add(expiresIn)
	claims := &Claims{
		Username: username,
		UID:      uid,
		Purpose:  purpose,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JwtKey)
}

func generateSecureKey(length int) ([]byte, error) {
	key := make([]byte, length)
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// 验证 JWT
func ValidateJWT(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return JwtKey, nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	return claims, nil
}
