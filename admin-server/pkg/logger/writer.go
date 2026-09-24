package logger

import (
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

// ensureDir 确保日志目录存在
func ensureDir(path string) error {
	if path == "" {
		return nil
	}
	return os.MkdirAll(path, 0755)
}

// ensureFileWritable checks the path before installing a file-backed logger.
func ensureFileWritable(filename string) error {
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	return f.Close()
}

// newRollingWriter 创建基于 lumberjack 的日志切割写入器
func newRollingWriter(filename string, cfg Config) *lumberjack.Logger {
	maxSize := cfg.MaxSize
	if maxSize <= 0 {
		maxSize = 100
	}
	maxAge := cfg.MaxAge
	if maxAge <= 0 {
		maxAge = 30
	}
	maxBackups := cfg.MaxBackups
	if maxBackups < 0 {
		maxBackups = 10
	}

	return &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    maxSize,
		MaxAge:     maxAge,
		MaxBackups: maxBackups,
		Compress:   cfg.Compress,
	}
}
