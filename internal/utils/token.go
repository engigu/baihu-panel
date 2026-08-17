package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
	TokenVersion int    `json:"version"`
	jwt.RegisteredClaims
}

// GenerateToken 生成 JWT token
func GenerateToken(userID string, username string, version int, expireDays int, secret string) (string, error) {
	claims := Claims{
		UserID:       userID,
		Username:     username,
		TokenVersion: version,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expireDays) * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseToken 解析 JWT token
func ParseToken(tokenString string, secret string) (string, string, int, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		// 校验算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})

	if err != nil {
		return "", "", 0, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims.UserID, claims.Username, claims.TokenVersion, nil
	}

	return "", "", 0, errors.New("invalid token")
}

// OtpPendingClaims 用于两步验证第一阶段和第二阶段之间传递的临时状态
type OtpPendingClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// GenerateOtpPendingToken 生成有效期为 5 分钟的临时登录 Token
func GenerateOtpPendingToken(userID string, secret string) (string, error) {
	claims := OtpPendingClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseOtpPendingToken 解析临时登录 Token 并提取 UserID
func ParseOtpPendingToken(tokenString string, secret string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &OtpPendingClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return "", err
	}
	if claims, ok := token.Claims.(*OtpPendingClaims); ok && token.Valid {
		return claims.UserID, nil
	}
	return "", errors.New("invalid otp pending token")
}
