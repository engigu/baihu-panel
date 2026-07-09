package utils

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"io"
	"strings"
	"sync"

	"github.com/klauspost/compress/zstd"
)

const (
	zstdPrefix = "zstd:"
	rawPrefix  = "raw:"
)

var zlibWriterPool = sync.Pool{
	New: func() interface{} {
		return zlib.NewWriter(io.Discard)
	},
}

// GetZlibWriter 从对象池中获取 zlib 写入器
func GetZlibWriter(w io.Writer) *zlib.Writer {
	zw := zlibWriterPool.Get().(*zlib.Writer)
	zw.Reset(w)
	return zw
}

// PutZlibWriter 将 zlib 写入器还回对象池
func PutZlibWriter(zw *zlib.Writer) {
	zlibWriterPool.Put(zw)
}

var zstdEncoderPool = sync.Pool{
	New: func() interface{} {
		zw, _ := zstd.NewWriter(nil)
		return zw
	},
}

// GetZstdWriter 从对象池中获取 zstd 写入器并定向到 w
func GetZstdWriter(w io.Writer) *zstd.Encoder {
	zw := zstdEncoderPool.Get().(*zstd.Encoder)
	zw.Reset(w)
	return zw
}

// PutZstdWriter 将 zstd 写入器还回对象池
func PutZstdWriter(zw *zstd.Encoder) {
	zstdEncoderPool.Put(zw)
}

var (
	zstdEncoder *zstd.Encoder
	zstdDecoder *zstd.Decoder
	zstdOnce    sync.Once
)

func initZstd() {
	zstdOnce.Do(func() {
		var err error
		// 默认级别适合常规压缩
		zstdEncoder, err = zstd.NewWriter(nil)
		if err != nil {
			panic(err)
		}
		zstdDecoder, err = zstd.NewReader(nil)
		if err != nil {
			panic(err)
		}
	})
}

// CompressToBase64 compresses data using zstd and encodes to base64 with a prefix (uses raw for data under 128 bytes)
func CompressToBase64(data string) (string, error) {
	if data == "" {
		return "", nil
	}

	// 小于等于 128 字节，不需要压缩，直接前缀明文保存
	if len(data) <= 128 {
		return rawPrefix + data, nil
	}

	initZstd()
	compressed := zstdEncoder.EncodeAll([]byte(data), nil)
	return zstdPrefix + base64.StdEncoding.EncodeToString(compressed), nil
}

// DecompressFromBase64 decodes base64 and decompresses data (supports raw/zstd prefix and falls back to zlib)
func DecompressFromBase64(data string) (string, error) {
	if data == "" {
		return "", nil
	}

	if strings.HasPrefix(data, rawPrefix) {
		return data[len(rawPrefix):], nil
	}

	if strings.HasPrefix(data, zstdPrefix) {
		initZstd()
		encoded := data[len(zstdPrefix):]
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return "", err
		}
		decompressed, err := zstdDecoder.DecodeAll(decoded, nil)
		if err != nil {
			return "", err
		}
		return string(decompressed), nil
	}

	// Fallback to legacy zlib
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", err
	}
	zr, err := zlib.NewReader(bytes.NewReader(decoded))
	if err != nil {
		return "", err
	}
	defer zr.Close()
	result, err := io.ReadAll(zr)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

