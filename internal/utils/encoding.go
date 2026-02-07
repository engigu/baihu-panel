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
