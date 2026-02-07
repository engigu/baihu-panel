package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

var Log *logrus.Logger

// CustomFormatter 自定义日志格式
type CustomFormatter struct{}

// ANSI 颜色代码 (参考 logrus 默认配色)
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m" // error, fatal, panic
	colorYellow = "\033[33m" // warn
	colorBlue   = "\033[36m" // info (logrus 默认用 cyan)
	colorGray   = "\033[37m" // debug (logrus 默认用 white)
)

func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	timestamp := entry.Time.Format("2006-01-02 15:04:05")
	level := strings.ToUpper(entry.Level.String())

	// 根据日志级别设置颜色 (参考 logrus TextFormatter 默认配色)
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

func init() {
	Log = logrus.New()

	// 设置自定义日志格式
	Log.SetFormatter(&CustomFormatter{})

	// 设置日志级别
	Log.SetLevel(logrus.InfoLevel)

	// 输出到标准输出
	Log.SetOutput(os.Stdout)
}

// SetupFileOutput 设置文件输出
func SetupFileOutput(logDir string) error {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	logFile := filepath.Join(logDir, time.Now().Format("2006-01-02")+".log")
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	Log.SetOutput(file)
	return nil
}

// SetLevel 设置日志级别
func SetLevel(level string) {
	switch level {
	case "debug":
		Log.SetLevel(logrus.DebugLevel)
	case "info":
		Log.SetLevel(logrus.InfoLevel)
	case "warn":
		Log.SetLevel(logrus.WarnLevel)
	case "error":
		Log.SetLevel(logrus.ErrorLevel)
	default:
		Log.SetLevel(logrus.InfoLevel)
	}
}

// 便捷方法
func Debug(args ...interface{}) { Log.Debug(args...) }
func Info(args ...interface{})  { Log.Info(args...) }
func Warn(args ...interface{})  { Log.Warn(args...) }
func Error(args ...interface{}) { Log.Error(args...) }
func Fatal(args ...interface{}) { Log.Fatal(args...) }

func Debugf(format string, args ...interface{}) { Log.Debugf(format, args...) }
func Infof(format string, args ...interface{})  { Log.Infof(format, args...) }
func Warnf(format string, args ...interface{})  { Log.Warnf(format, args...) }
func Errorf(format string, args ...interface{}) { Log.Errorf(format, args...) }
func Fatalf(format string, args ...interface{}) { Log.Fatalf(format, args...) }

// WithField 带字段的日志
func WithField(key string, value interface{}) *logrus.Entry {
	return Log.WithField(key, value)
}

// WithFields 带多个字段的日志
func WithFields(fields logrus.Fields) *logrus.Entry {
	return Log.WithFields(fields)
}

// SchedulerLogger 兼容 internal/executor 的日志接口
type SchedulerLogger struct{}

func (s *SchedulerLogger) Infof(format string, args ...interface{}) {
	Log.Infof(format, args...)
}
func (s *SchedulerLogger) Warnf(format string, args ...interface{}) {
	Log.Warnf(format, args...)
}
func (s *SchedulerLogger) Errorf(format string, args ...interface{}) {
	Log.Errorf(format, args...)
}

// NewSchedulerLogger 创建一个兼容 executor.SchedulerLogger 的实例
func NewSchedulerLogger() *SchedulerLogger {
	return &SchedulerLogger{}
}
