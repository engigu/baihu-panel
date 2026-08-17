package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/engigu/baihu-panel/internal/systime"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger
var Sugar *zap.SugaredLogger
var atomicLevel zap.AtomicLevel

// ANSI 颜色代码
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m" // error, fatal, panic
	colorYellow = "\033[33m" // warn
	colorBlue   = "\033[36m" // info
	colorGray   = "\033[37m" // debug
)

// CustomCore 实现 zapcore.Core 以提供统一日志格式
type CustomCore struct {
	Level  zapcore.LevelEnabler
	Writer zapcore.WriteSyncer
}

func (c *CustomCore) Enabled(l zapcore.Level) bool {
	return c.Level.Enabled(l)
}

func (c *CustomCore) With(fields []zapcore.Field) zapcore.Core {
	return c
}

func (c *CustomCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *CustomCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	// 统一使用东八区时间
	timestamp := systime.InCST(ent.Time).Format("2006-01-02 15:04:05")
	level := strings.ToUpper(ent.Level.String())

	var levelColor string
	switch ent.Level {
	case zapcore.DebugLevel:
		levelColor = colorGray
	case zapcore.InfoLevel:
		levelColor = colorBlue
	case zapcore.WarnLevel:
		levelColor = colorYellow
	case zapcore.ErrorLevel, zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		levelColor = colorRed
	default:
		levelColor = colorBlue
	}

	msg := fmt.Sprintf("[%s]%s[%s]%s %s\n", timestamp, levelColor, level, colorReset, ent.Message)
	_, err := c.Writer.Write([]byte(msg))
	return err
}

func (c *CustomCore) Sync() error {
	return c.Writer.Sync()
}

// NewCustomCore 创建通用导出的 Zap Core 实例
func NewCustomCore(level zapcore.LevelEnabler, writer zapcore.WriteSyncer) zapcore.Core {
	return &CustomCore{
		Level:  level,
		Writer: writer,
	}
}

func newLogger(output zapcore.WriteSyncer) *zap.Logger {
	return zap.New(NewCustomCore(atomicLevel, output))
}

func init() {
	// 强制设置全局时区为东八区
	time.Local = systime.CST
	atomicLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	Log = newLogger(zapcore.AddSync(os.Stdout))
	Sugar = Log.Sugar()
}

// SetupFileOutput 设置文件输出
func SetupFileOutput(logDir string) error {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	logFile := filepath.Join(logDir, systime.FormatDate(time.Now())+".log")
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	Log = newLogger(zapcore.AddSync(file))
	Sugar = Log.Sugar()
	return nil
}

// SetOutput 直接设置 Log 实例
func SetOutput(l *zap.Logger) {
	Log = l
	Sugar = l.Sugar()
}

// SetSugar 直接设置 Sugar 实例
func SetSugar(s *zap.SugaredLogger) {
	Sugar = s
}

// SetLevel 设置日志级别
func SetLevel(level string) {
	switch level {
	case "debug":
		atomicLevel.SetLevel(zap.DebugLevel)
	case "info":
		atomicLevel.SetLevel(zap.InfoLevel)
	case "warn":
		atomicLevel.SetLevel(zap.WarnLevel)
	case "error":
		atomicLevel.SetLevel(zap.ErrorLevel)
	default:
		atomicLevel.SetLevel(zap.InfoLevel)
	}
}

// 便捷方法
func Debug(args ...interface{}) { Sugar.Debug(args...) }
func Info(args ...interface{})  { Sugar.Info(args...) }
func Warn(args ...interface{})  { Sugar.Warn(args...) }
func Error(args ...interface{}) { Sugar.Error(args...) }
func Fatal(args ...interface{}) { Sugar.Fatal(args...) }

func Debugf(format string, args ...interface{}) { Sugar.Debugf(format, args...) }
func Infof(format string, args ...interface{})  { Sugar.Infof(format, args...) }
func Warnf(format string, args ...interface{})  { Sugar.Warnf(format, args...) }
func Errorf(format string, args ...interface{}) { Sugar.Errorf(format, args...) }
func Fatalf(format string, args ...interface{}) { Sugar.Fatalf(format, args...) }

// WithField 带字段的日志
func WithField(key string, value interface{}) *zap.SugaredLogger {
	return Sugar.With(key, value)
}

// WithFields 带多个字段的日志
func WithFields(fields map[string]interface{}) *zap.SugaredLogger {
	f := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		f = append(f, k, v)
	}
	return Sugar.With(f...)
}

// SchedulerLogger 兼容 internal/executor 的日志接口
type SchedulerLogger struct{}

func (s *SchedulerLogger) Infof(format string, args ...interface{}) {
	Sugar.Infof(format, args...)
}
func (s *SchedulerLogger) Warnf(format string, args ...interface{}) {
	Sugar.Warnf(format, args...)
}
func (s *SchedulerLogger) Errorf(format string, args ...interface{}) {
	Sugar.Errorf(format, args...)
}

// NewSchedulerLogger 创建一个兼容 executor.SchedulerLogger 的实例
func NewSchedulerLogger() *SchedulerLogger {
	return &SchedulerLogger{}
}
