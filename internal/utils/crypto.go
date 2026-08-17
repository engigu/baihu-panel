package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"strings"
)

var (
	masterSecretKey []byte
	ErrKeyNotSet    = errors.New("加密秘钥未配置，请按照文档使用 BAIHU_SECRET_KEY 环境变量启动服务配置秘钥")
)

// InitSecretKey initialized the master secret key from the environment and unsets it
func InitSecretKey() {
	key := os.Getenv("BAIHU_SECRET_KEY")
	if key != "" {
		hash := sha256.Sum256([]byte(key))
		masterSecretKey = hash[:]
		// Ensure it's only in memory by unsetting the environment variable
		os.Unsetenv("BAIHU_SECRET_KEY")
	}
}

// IsSecretKeySet returns true if the master secret key is configured
func IsSecretKeySet() bool {
	return len(masterSecretKey) > 0
}

// Encrypt encrypts a plaintext string using AES-GCM
func Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	if !IsSecretKeySet() {
		return "", ErrKeyNotSet
	}

	block, err := aes.NewCipher(masterSecretKey)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a ciphertext string using AES-GCM
// Returns the original string if decryption fails or if it wasn't encrypted
func Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	if !IsSecretKeySet() {
		return ciphertext, ErrKeyNotSet
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return ciphertext, err
	}

	block, err := aes.NewCipher(masterSecretKey)
	if err != nil {
		return ciphertext, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return ciphertext, err
	}

	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return ciphertext, errors.New("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return ciphertext, err
	}

	return string(plaintext), nil
}

// MaskSecrets 将文本中的所有敏感机密值替换为脱敏字符串 "********"
func MaskSecrets(text string, secrets []string) string {
	if len(secrets) == 0 || text == "" {
		return text
	}
	for _, mask := range secrets {
		if mask != "" {
			text = strings.ReplaceAll(text, mask, "********")
		}
	}
	return text
}

// MaskString 对字符串进行脱敏处理，保留首尾，中间用星号遮掩
func MaskString(s string) string {
	if s == "" {
		return ""
	}
	n := len(s)
	if n <= 3 {
		return "***"
	}
	if n <= 6 {
		return s[:1] + "***" + s[n-1:]
	}
	return s[:2] + "****" + s[n-2:]
}
