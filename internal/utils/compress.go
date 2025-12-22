package utils

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
)

// CompressToBase64 compresses data using zlib and encodes to base64
func CompressToBase64(data string) (string, error) {
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

// // DecompressFromBase64 decodes base64 and decompresses zlib data
// func DecompressFromBase64(data string) (string, error) {
// 	if data == "" {
// 		return "", nil
// 	}
// 	decoded, err := base64.StdEncoding.DecodeString(data)
// 	if err != nil {
// 		return "", err
// 	}
// 	zr, err := zlib.NewReader(bytes.NewReader(decoded))
// 	if err != nil {
// 		return "", err
// 	}
// 	defer zr.Close()
// 	result, err := io.ReadAll(zr)
// 	if err != nil {
// 		return "", err
// 	}
// 	return string(result), nil
// }
