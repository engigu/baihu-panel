package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"strings"
	"testing"
)

func TestEcdhEncryptDecrypt(t *testing.T) {
	curve := ecdh.P256()

	// 1. 模拟客户端生成 P-256 密钥对
	clientPriv, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("生成客户端密钥对失败: %v", err)
	}

	clientPubBytes := clientPriv.PublicKey().Bytes()
	clientPubBase64 := base64.StdEncoding.EncodeToString(clientPubBytes)

	// 2. 测试简单短文本机密
	plainText := "ghp_xxxxxxxxxxxx1234567890abcdef"
	payload, err := EcdhEncrypt(clientPubBase64, plainText)
	if err != nil {
		t.Fatalf("EcdhEncrypt 失败: %v", err)
	}

	// 3. 模拟客户端接收到 payload 后进行 ECDH 协商解密
	serverPubBytes, err := base64.StdEncoding.DecodeString(payload.ServerPublicKey)
	if err != nil {
		t.Fatalf("解析 ServerPublicKey 失败: %v", err)
	}

	serverPub, err := curve.NewPublicKey(serverPubBytes)
	if err != nil {
		t.Fatalf("解析服务端公钥失败: %v", err)
	}

	// 客户端执行 ECDH 派生共享密钥
	sharedSecret, err := clientPriv.ECDH(serverPub)
	if err != nil {
		t.Fatalf("客户端 ECDH 协商失败: %v", err)
	}

	// 客户端使用 sharedSecret 进行 AES-GCM 解密
	block, err := aes.NewCipher(sharedSecret)
	if err != nil {
		t.Fatalf("NewCipher 失败: %v", err)
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("NewGCM 失败: %v", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(payload.Nonce)
	if err != nil {
		t.Fatalf("解码 Nonce 失败: %v", err)
	}
	cipherBytes, err := base64.StdEncoding.DecodeString(payload.Ciphertext)
	if err != nil {
		t.Fatalf("解码 Ciphertext 失败: %v", err)
	}

	decryptedBytes, err := aesGCM.Open(nil, nonce, cipherBytes, nil)
	if err != nil {
		t.Fatalf("解密失败: %v", err)
	}

	if string(decryptedBytes) != plainText {
		t.Fatalf("解密内容不匹配! 期望: %s, 实际: %s", plainText, string(decryptedBytes))
	}

	// 4. 测试超长机密文本（模拟长证书/长多行凭据，50KB）
	largePlainText := strings.Repeat("-----BEGIN CERTIFICATE-----\nMIIDdzCCAl+gAwIBAgIU\n-----END CERTIFICATE-----\n", 1000)
	largePayload, err := EcdhEncrypt(clientPubBase64, largePlainText)
	if err != nil {
		t.Fatalf("大文本 EcdhEncrypt 失败: %v", err)
	}

	largeServerPubBytes, err := base64.StdEncoding.DecodeString(largePayload.ServerPublicKey)
	if err != nil {
		t.Fatalf("解析大文本 ServerPublicKey 失败: %v", err)
	}
	largeServerPub, err := curve.NewPublicKey(largeServerPubBytes)
	if err != nil {
		t.Fatalf("解析大文本服务端公钥失败: %v", err)
	}

	largeSharedSecret, err := clientPriv.ECDH(largeServerPub)
	if err != nil {
		t.Fatalf("大文本 ECDH 协商失败: %v", err)
	}

	largeBlock, _ := aes.NewCipher(largeSharedSecret)
	largeAesGCM, _ := cipher.NewGCM(largeBlock)

	largeCipherBytes, _ := base64.StdEncoding.DecodeString(largePayload.Ciphertext)
	largeNonce, _ := base64.StdEncoding.DecodeString(largePayload.Nonce)
	largeDecryptedBytes, err := largeAesGCM.Open(nil, largeNonce, largeCipherBytes, nil)
	if err != nil {
		t.Fatalf("大文本解密失败: %v", err)
	}
	if string(largeDecryptedBytes) != largePlainText {
		t.Fatalf("大文本解密内容不匹配")
	}
}
