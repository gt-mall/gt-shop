package logger

import (
	"os"
	"strconv"
	"strings"
)

// Config 日志配置结构体
type Config struct {
	Level      string `json:"level" yaml:"level"`             // 日志级别：debug, info, warn, error
	Path       string `json:"path" yaml:"path"`               // 日志存储目录，默认 ./logs
	Console    bool   `json:"console" yaml:"console"`         // 是否输出到控制台
	File       bool   `json:"file" yaml:"file"`               // 是否输出到文件
	MaxSize    int    `json:"max_size" yaml:"max_size"`       // 单个日志文件最大大小 (MB)
	MaxAge     int    `json:"max_age" yaml:"max_age"`         // 日志保留最大天数
	MaxBackups int    `json:"max_backups" yaml:"max_backups"` // 日志最多备份份数；0 表示不限制份数，仍受 MaxAge 限制
	Compress   bool   `json:"compress" yaml:"compress"`       // 旧日志是否压缩
}

// DefaultConfig 返回默认日志配置
func DefaultConfig() Config {
	return Config{
		Level:      "info",
		Path:       "./logs",
		Console:    true,
		File:       true,
		MaxSize:    100,
		MaxAge:     30,
		MaxBackups: 10,
		Compress:   true,
	}
}

// LoadConfigFromEnv 从环境变量读取日志配置，未设置项保持默认值
func LoadConfigFromEnv() Config {
	cfg := DefaultConfig()
	if lvl := strings.TrimSpace(os.Getenv("LOG_LEVEL")); lvl != "" {
		cfg.Level = lvl
	}
	if p := strings.TrimSpace(os.Getenv("LOG_PATH")); p != "" {
		cfg.Path = p
	}
	if c := strings.TrimSpace(os.Getenv("LOG_CONSOLE")); c != "" {
		if val, err := strconv.ParseBool(c); err == nil {
			cfg.Console = val
		}
	}
	if f := strings.TrimSpace(os.Getenv("LOG_FILE")); f != "" {
		if val, err := strconv.ParseBool(f); err == nil {
			cfg.File = val
		}
	}
	if s := strings.TrimSpace(os.Getenv("LOG_MAX_SIZE")); s != "" {
		if val, err := strconv.Atoi(s); err == nil && val > 0 {
			cfg.MaxSize = val
		}
	}
	if a := strings.TrimSpace(os.Getenv("LOG_MAX_AGE")); a != "" {
		if val, err := strconv.Atoi(a); err == nil && val > 0 {
			cfg.MaxAge = val
		}
	}
	if b := strings.TrimSpace(os.Getenv("LOG_MAX_BACKUPS")); b != "" {
		if val, err := strconv.Atoi(b); err == nil && val >= 0 {
			cfg.MaxBackups = val
		}
	}
	if cmp := strings.TrimSpace(os.Getenv("LOG_COMPRESS")); cmp != "" {
		if val, err := strconv.ParseBool(cmp); err == nil {
			cfg.Compress = val
		}
	}
	return cfg
}
