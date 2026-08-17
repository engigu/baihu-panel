package utils

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"strings"
	"testing"
)

// legacyZlibCompress 以前的 zlib 压缩逻辑，用于构造测试样本
func legacyZlibCompress(data string) (string, error) {
	if data == "" {
		return "", nil
	}
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	if _, err := zw.Write([]byte(data)); err != nil {
		return "", err
	}
	if err := zw.Close(); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func TestCompressAndDecompressZstd(t *testing.T) {
	originalText := "Hello, this is a test log message for ZSTD compression in Baihu Panel! Repeat: Hello, this is a test log message for ZSTD compression in Baihu Panel!"

	// 1. 测试 ZSTD 压缩
	compressed, err := CompressToBase64(originalText)
	if err != nil {
		t.Fatalf("CompressToBase64 failed: %v", err)
	}

	// 验证前缀是否正确
	if !strings.HasPrefix(compressed, "zstd:") {
		t.Errorf("Expected compressed output to have 'zstd:' prefix, got: %s", compressed)
	}

	// 2. 测试 ZSTD 解密解压
	decompressed, err := DecompressFromBase64(compressed)
	if err != nil {
		t.Fatalf("DecompressFromBase64 failed: %v", err)
	}

	if decompressed != originalText {
		t.Errorf("Decompressed text mismatch.\nExpected: %s\nGot: %s", originalText, decompressed)
	}
}

func TestDecompressLegacyZlibCompatibility(t *testing.T) {
	originalText := "This is a legacy log message compressed using zlib. It should be decompressed successfully."

	// 1. 用老逻辑压缩生成旧数据
	legacyCompressed, err := legacyZlibCompress(originalText)
	if err != nil {
		t.Fatalf("legacyZlibCompress failed: %v", err)
	}

	// 验证没有 zstd 前缀
	if strings.HasPrefix(legacyCompressed, "zstd:") {
		t.Fatalf("Legacy compressed string shouldn't have 'zstd:' prefix")
	}

	// 2. 使用新版的 DecompressFromBase64 解压，验证其对旧格式的兼容性
	decompressed, err := DecompressFromBase64(legacyCompressed)
	if err != nil {
		t.Fatalf("DecompressFromBase64 failed to decompress legacy data: %v", err)
	}

	if decompressed != originalText {
		t.Errorf("Decompressed legacy text mismatch.\nExpected: %s\nGot: %s", originalText, decompressed)
	}
}

func TestEmptyString(t *testing.T) {
	compressed, err := CompressToBase64("")
	if err != nil {
		t.Fatalf("CompressToBase64 for empty string failed: %v", err)
	}
	if compressed != "" {
		t.Errorf("Expected empty string for empty input, got: %q", compressed)
	}

	decompressed, err := DecompressFromBase64("")
	if err != nil {
		t.Fatalf("DecompressFromBase64 for empty string failed: %v", err)
	}
	if decompressed != "" {
		t.Errorf("Expected empty string for empty input, got: %q", decompressed)
	}
}

func TestCompressShortTextRaw(t *testing.T) {
	shortText := "python secret.py" // 16 bytes

	// 1. 压缩（应转为 raw 前缀）
	output, err := CompressToBase64(shortText)
	if err != nil {
		t.Fatalf("CompressToBase64 short text failed: %v", err)
	}

	if !strings.HasPrefix(output, "raw:") {
		t.Errorf("Expected short text output to have 'raw:' prefix, got: %q", output)
	}
	if output != "raw:python secret.py" {
		t.Errorf("Expected output raw:python secret.py, got: %q", output)
	}

	// 2. 解码解密
	decompressed, err := DecompressFromBase64(output)
	if err != nil {
		t.Fatalf("DecompressFromBase64 short text failed: %v", err)
	}

	if decompressed != shortText {
		t.Errorf("Decompressed short text mismatch.\nExpected: %q\nGot: %q", shortText, decompressed)
	}
}

