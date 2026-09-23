package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"fmt"
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

	cipherBytes, err := AesGcmEncrypt(masterSecretKey, []byte(plaintext))
	if err != nil {
		return "", err
	}
	if cipherBytes == nil {
		return "", nil
	}

	return base64.StdEncoding.EncodeToString(cipherBytes), nil
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

// AesGcmEncrypt AES-GCM 加密
func AesGcmEncrypt(keyBytes []byte, plainBytes []byte) ([]byte, error) {
	if len(plainBytes) <= 0 {
		return nil, nil
	}
	if len(keyBytes) <= 0 {
		return nil, errors.New("keyBytes is empty")
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	cipherBytes := aesGCM.Seal(nonce, nonce, plainBytes, nil)
	return cipherBytes, nil
}

// AesGcmEncryptString AES-GCM 加密
func AesGcmEncryptString(key string, plainText string) (string, error) {
	if plainText == "" {
		return "", nil
	}
	if key == "" {
		return "", errors.New("key is empty")
	}

	cipherBytes, err := AesGcmEncrypt([]byte(key), []byte(plainText))
	if err != nil {
		return "", err
	}
	if cipherBytes == nil {
		return "", nil
	}

	return base64.StdEncoding.EncodeToString(cipherBytes), nil
}

// RsaEncryptByPublicKey RSA 根据公钥加密
func RsaEncryptByPublicKey(publicKeyBytes []byte, plainBytes []byte) ([]byte, error) {
	if len(plainBytes) <= 0 {
		return nil, nil
	}
	if len(publicKeyBytes) <= 0 {
		return nil, errors.New("publicKeyBytes is empty")
	}

	pub, err := x509.ParsePKIXPublicKey(publicKeyBytes)
	if err != nil {
		return nil, err
	}

	publicKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("publicKey is error")
	}

	cipherBytes, err := rsa.EncryptOAEP(
		nil,
		rand.Reader,
		publicKey,
		plainBytes,
		nil,
	)
	if err != nil {
		return nil, err
	}

	return cipherBytes, nil
}

// RsaEncryptByPublicKeyString RSA 根据公钥加密
func RsaEncryptByPublicKeyString(publicKey string, plainText string) (string, error) {
	if plainText == "" {
		return "", nil
	}
	if publicKey == "" {
		return "", errors.New("publicKey is empty")
	}

	cipherBytes, err := RsaEncryptByPublicKey([]byte(publicKey), []byte(plainText))
	if err != nil {
		return "", err
	}
	if cipherBytes == nil {
		return "", nil
	}

	return base64.StdEncoding.EncodeToString(cipherBytes), nil
}
