package main

import (
	"os"
	"path/filepath"

	"github.com/engigu/baihu-panel/internal/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// 日志实例
var loggerInstance *zap.Logger
var log *zap.SugaredLogger

func initLogger(logFile string, fileOnly bool) {
	logDir := filepath.Dir(logFile)
	if logDir != "" && logDir != "." {
		os.MkdirAll(logDir, 0755)
	}

	lumberjackLogger := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    5,
		MaxBackups: 3,
		MaxAge:     0,
		Compress:   false,
	}

	var output zapcore.WriteSyncer
	// fileOnly 模式下只输出到文件（daemon 模式或重启模式）
	if fileOnly {
		output = zapcore.AddSync(lumberjackLogger)
	} else {
		// 前台运行时同时输出到终端和文件
		output = zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(lumberjackLogger))
	}

	core := logger.NewCustomCore(zap.DebugLevel, output)
	loggerInstance = zap.New(core)
	log = loggerInstance.Sugar()
}
