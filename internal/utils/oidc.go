package utils

import (
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"strings"
)

// RandomStringSecure 生成指定长度的高熵随机十六进制字符串（用于自动生成口令等）
func RandomStringSecure(n int) (string, error) {
	b := make([]byte, (n+1)/2)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	s := hex.EncodeToString(b)
	if len(s) > n {
		s = s[:n]
	}
	return s, nil
}

// ParsePositiveInt 解析正整数，非法或非正数时返回 0
func ParsePositiveInt(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

// DecryptSecret 解密数据库中存储的敏感配置；未配置主密钥或解密失败时按明文处理
func DecryptSecret(value string) string {
	if value == "" {
		return ""
	}
	// 未设置 BAIHU_SECRET_KEY 时无法解密，按原值返回，避免把可用配置读成空
	if !IsSecretKeySet() {
		return value
	}
	plain, err := Decrypt(value)
	if err != nil {
		return value
	}
	return plain
}
