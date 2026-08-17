package utils

import (
	"bufio"
	"io"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// ToUTF8 converts potentially non-UTF8 data (like GBK on Windows) to UTF-8
func ToUTF8(data []byte) string {
	if utf8.Valid(data) {
		return string(data)
	}
	// Try GBK (common on Windows)
	reader := transform.NewReader(
		bufio.NewReader(
			&byteReader{data: data},
		),
		simplifiedchinese.GBK.NewDecoder(),
	)
	result, err := io.ReadAll(reader)
	if err != nil {
		return string(data)
	}
	return string(result)
}

type byteReader struct {
	data []byte
	pos  int
}

func (r *byteReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// TrimLastRunes 从字符串尾部保留最多 maxRunes 个字符（不仅限于 ASCII，支持中英文混排的真实字符数量）
func TrimLastRunes(s string, maxRunes int) string {
	// 如果字符串的总字节数小于等于 maxRunes，那么它的字符数一定也小于等于 maxRunes
	if len(s) <= maxRunes {
		return s
	}

	count := 0
	for i := len(s); i > 0; {
		_, size := utf8.DecodeLastRuneInString(s[:i])
		i -= size
		count++
		if count == maxRunes {
			return s[i:]
		}
	}
	return s
}
