package clibase

import (
	"bytes"
	"io"
	"regexp"
	"strings"
)

// AnsiRegex 匹配终端 ANSI 控制序列的通用正则表达式
var AnsiRegex = regexp.MustCompile("\x1b\\[[0-9;]*[a-zA-Z]")

// CleanWriter 过滤输出流中的终端回车符覆写及 ANSI 色彩代码
type CleanWriter struct {
	out io.Writer
	buf []byte
}

// NewCleanWriter 构造输出清洗器
func NewCleanWriter(out io.Writer) *CleanWriter {
	return &CleanWriter{out: out}
}

func (c *CleanWriter) Write(p []byte) (n int, err error) {
	c.buf = append(c.buf, p...)

	for {
		idx := bytes.IndexAny(c.buf, "\r\n")
		if idx == -1 {
			break
		}

		if c.buf[idx] == '\r' && idx == len(c.buf)-1 {
			// 跨块截断的回车，等待下一块
			break
		}

		char := c.buf[idx]
		line := string(c.buf[:idx])
		c.buf = c.buf[idx+1:]

		if char == '\r' && len(c.buf) > 0 && c.buf[0] == '\n' {
			c.buf = c.buf[1:]
			char = '\n'
		}

		s := AnsiRegex.ReplaceAllString(line, "")

		if char == '\r' {
			continue // 忽略终端进度条的同行覆盖
		}

		if s != "" {
			c.out.Write([]byte(s + "\n"))
		}
	}
	return len(p), nil
}

// Flush 输出末尾缓冲
func (c *CleanWriter) Flush() {
	if len(c.buf) > 0 {
		s := string(c.buf)
		s = strings.TrimSuffix(s, "\r")
		s = AnsiRegex.ReplaceAllString(s, "")
		if s != "" {
			c.out.Write([]byte(s + "\n"))
		}
		c.buf = nil
	}
}
