package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// 日志实例
var log = logrus.New()

// CustomFormatter 自定义日志格式
type CustomFormatter struct{}

// ANSI 颜色代码
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[36m"
	colorGray   = "\033[37m"
)

func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	timestamp := entry.Time.Format("2006-01-02 15:04:05")
	level := strings.ToUpper(entry.Level.String())

	var levelColor string
	switch entry.Level {
	case logrus.DebugLevel, logrus.TraceLevel:
		levelColor = colorGray
	case logrus.InfoLevel:
		levelColor = colorBlue
	case logrus.WarnLevel:
		levelColor = colorYellow
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		levelColor = colorRed
	default:
		levelColor = colorBlue
	}

	msg := fmt.Sprintf("[%s]%s[%s]%s %s\n", timestamp, levelColor, level, colorReset, entry.Message)
	return []byte(msg), nil
}

func initLogger(logFile string) {
	logDir := filepath.Dir(logFile)
	if logDir != "" && logDir != "." {
		os.MkdirAll(logDir, 0755)
	}

	log.SetFormatter(&CustomFormatter{})
	log.SetLevel(logrus.InfoLevel)

	lumberjackLogger := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    5,
		MaxBackups: 3,
		MaxAge:     0,
		Compress:   false,
	}

	log.SetOutput(io.MultiWriter(os.Stdout, lumberjackLogger))
}
