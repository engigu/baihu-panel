package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
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

// EncryptIfSecret 如果变量类型为机密 (secret)，则对其执行加密处理
func EncryptIfSecret(envType string, value string) (string, error) {
	if envType == "secret" {
		return Encrypt(value)
	}
	return value, nil
}

// DecryptIfSecret 如果变量类型为机密 (secret)，则对其执行解密处理
func DecryptIfSecret(envType string, value string) (string, error) {
	if envType == "secret" {
		return Decrypt(value)
	}
	return value, nil
}

// IsSecretEncrypted 判断给定的字符串是否看起来像是已加密的密文
func IsSecretEncrypted(value string) bool {
	if value == "" {
		return false
	}
	data, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return false
	}
	return len(data) > 12 // AES-GCM 的标准 Nonce 大小通常为 12 字节
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

// EcdhCipherPayload 椭圆曲线 ECDH (P-256) + AES-256-GCM 传输密文载荷
type EcdhCipherPayload struct {
	ServerPublicKey string `json:"server_public_key"` // 服务端临时 P-256 公钥 (Base64)
	Ciphertext      string `json:"ciphertext"`        // AES-256-GCM 加密后的密文 (Base64, 包含 16 字节 Auth Tag)
	Nonce           string `json:"nonce"`             // AES-256-GCM Nonce/IV (Base64, 12 字节)
}

// ParseEcdhP256PublicKey 解析客户端公钥，支持 65 字节 Raw 格式或 PKIX/SPKI 格式
func ParseEcdhP256PublicKey(pubKeyStr string) (*ecdh.PublicKey, error) {
	pubKeyStr = strings.TrimSpace(pubKeyStr)
	if pubKeyStr == "" {
		return nil, errors.New("客户端公钥为空")
	}

	var rawBytes []byte
	if strings.Contains(pubKeyStr, "-----BEGIN") {
		block, _ := pem.Decode([]byte(pubKeyStr))
		if block == nil {
			return nil, errors.New("无效的 PEM 格式公钥")
		}
		rawBytes = block.Bytes
	} else {
		cleaned := strings.ReplaceAll(pubKeyStr, "\n", "")
		cleaned = strings.ReplaceAll(cleaned, "\r", "")
		cleaned = strings.ReplaceAll(cleaned, " ", "")
		var err error
		rawBytes, err = base64.StdEncoding.DecodeString(cleaned)
		if err != nil {
			return nil, fmt.Errorf("客户端公钥 Base64 解码失败: %w", err)
		}
	}

	curve := ecdh.P256()

	// 1. 如果长度为 65 字节且首字节为 0x04 (未压缩格式点)，直接按 ecdh Raw 公钥解析
	if len(rawBytes) == 65 && rawBytes[0] == 0x04 {
		pubKey, err := curve.NewPublicKey(rawBytes)
		if err != nil {
			return nil, fmt.Errorf("无效的 P-256 原始公钥点: %w", err)
		}
		return pubKey, nil
	}

	// 2. 尝试作为 PKIX/SPKI 解析
	parsedKey, err := x509.ParsePKIXPublicKey(rawBytes)
	if err == nil {
		if ecdsaKey, ok := parsedKey.(*ecdsa.PublicKey); ok {
			if ecdsaKey.Curve == elliptic.P256() {
				return ecdsaKey.ECDH()
			}
			return nil, errors.New("公钥曲线类型不匹配，必须为 P-256 (secp256r1)")
		}
	}

	return nil, errors.New("无法解析客户端公钥为有效的 ECDH P-256 公钥")
}

// EcdhEncrypt 使用 ECDH (P-256) 协商 32 字节共享秘钥并使用 AES-256-GCM 加密明文，支持任意长度数据且具备前向保密性
func EcdhEncrypt(clientPubKeyStr string, plainText string) (*EcdhCipherPayload, error) {
	clientPub, err := ParseEcdhP256PublicKey(clientPubKeyStr)
	if err != nil {
		return nil, err
	}

	curve := ecdh.P256()

	// 1. 服务端生成一次性临时 P-256 密钥对 (保证完全前向安全性 PFS)
	serverPriv, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("生成服务端临时密钥对失败: %w", err)
	}

	// 2. 执行 ECDH 协商，派生 32 字节共享秘钥 (与 Web Crypto deriveKey 规范对齐)
	sharedSecret, err := serverPriv.ECDH(clientPub)
	if err != nil {
		return nil, fmt.Errorf("ECDH 密钥协商失败: %w", err)
	}

	// 3. 使用协商好的共享秘钥进行 AES-256-GCM 对称加密
	block, err := aes.NewCipher(sharedSecret)
	if err != nil {
		return nil, fmt.Errorf("初始化 AES Cipher 失败: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("初始化 AES-GCM 模式失败: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("生成随机 Nonce 失败: %w", err)
	}

	cipherBytes := aesGCM.Seal(nil, nonce, []byte(plainText), nil)

	// 4. 将服务端临时公钥导出为 65 字节 Raw 格式 Base64
	serverPubBytes := serverPriv.PublicKey().Bytes()

	return &EcdhCipherPayload{
		ServerPublicKey: base64.StdEncoding.EncodeToString(serverPubBytes),
		Ciphertext:      base64.StdEncoding.EncodeToString(cipherBytes),
		Nonce:           base64.StdEncoding.EncodeToString(nonce),
	}, nil
}
