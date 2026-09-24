package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Field 日志字段类型别名
type Field = zap.Field

var (
	loggerPtr    *zap.Logger
	sugaredPtr   *zap.SugaredLogger
	activeCore   *switchableCore
	fallbackCore zapcore.Core
)

func init() {
	// 默认初始化一个简易控制台 logger，防止未调用 Init 时 panic
	fallbackCore = zapcore.NewCore(
		zapcore.NewConsoleEncoder(defaultConsoleEncoderConfig()),
		zapcore.AddSync(os.Stdout),
		zapcore.InfoLevel,
	)
	activeCore = newSwitchableCore(fallbackCore)
	loggerPtr = zap.New(activeCore, zap.AddCaller(), zap.AddCallerSkip(1))
	sugaredPtr = loggerPtr.Sugar()
}

// Init 根据配置初始化全局日志实例
func Init(cfg Config) error {
	level := parseLevel(cfg.Level)
	var cores []zapcore.Core
	var newWriters []*lumberjack.Logger

	// 1. 控制台输出
	if cfg.Console {
		consoleEncoder := zapcore.NewConsoleEncoder(defaultConsoleEncoderConfig())
		consoleCore := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level)
		cores = append(cores, levelFilterCore{Core: consoleCore})
	}

	// 2. 文件输出
	if cfg.File {
		logPath := strings.TrimSpace(cfg.Path)
		if logPath == "" {
			logPath = "./logs"
		}
		if err := ensureDir(logPath); err != nil {
			return fmt.Errorf("创建日志目录 %s 失败: %w", logPath, err)
		}

		jsonEncoder := zapcore.NewJSONEncoder(defaultJSONEncoderConfig())

		// app.log 保存所有满足最小 level 的日志
		appLogFile := filepath.Join(logPath, "app.log")
		if err := ensureFileWritable(appLogFile); err != nil {
			return fmt.Errorf("日志文件 %s 不可写: %w", appLogFile, err)
		}
		appWriter := newRollingWriter(appLogFile, cfg)
		newWriters = append(newWriters, appWriter)
		appCore := zapcore.NewCore(jsonEncoder, zapcore.AddSync(appWriter), level)
		cores = append(cores, levelFilterCore{Core: appCore})

		// error.log 只保存 ERROR 及以上级别
		errLogFile := filepath.Join(logPath, "error.log")
		if err := ensureFileWritable(errLogFile); err != nil {
			return fmt.Errorf("日志文件 %s 不可写: %w", errLogFile, err)
		}
		errWriter := newRollingWriter(errLogFile, cfg)
		newWriters = append(newWriters, errWriter)
		errorLevelEnabler := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl >= zapcore.ErrorLevel && lvl >= level
		})
		errCore := zapcore.NewCore(jsonEncoder, zapcore.AddSync(errWriter), errorLevelEnabler)
		cores = append(cores, levelFilterCore{Core: errCore})
	}

	// 若既未开启控制台也未开启文件，则使用 NopCore
	var finalCore zapcore.Core
	if len(cores) == 0 {
		finalCore = zapcore.NewNopCore()
	} else if len(cores) == 1 {
		finalCore = cores[0]
	} else {
		finalCore = zapcore.NewTee(cores...)
	}

	activeCore.swap(finalCore, newWriters)

	return nil
}

// defaultJSONEncoderConfig 返回统一的 JSON 编码配置
func defaultJSONEncoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		TimeKey:       "time",
		LevelKey:      "level",
		NameKey:       "logger",
		CallerKey:     "caller",
		MessageKey:    "message",
		StacktraceKey: "stacktrace",
		LineEnding:    zapcore.DefaultLineEnding,
		EncodeLevel:   zapcore.LowercaseLevelEncoder,
		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format(time.RFC3339))
		},
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}

// defaultConsoleEncoderConfig 返回开发友好的控制台编码配置
func defaultConsoleEncoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000"),
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}

// parseLevel 解析日志级别字符串
func parseLevel(lvl string) zapcore.Level {
	switch strings.ToLower(strings.TrimSpace(lvl)) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "dpanic":
		return zapcore.DPanicLevel
	case "panic":
		return zapcore.PanicLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

// L 获取当前底层 zap.Logger 实例
func L() *zap.Logger {
	return loggerPtr
}

// S 获取当前底层 zap.SugaredLogger 实例
func S() *zap.SugaredLogger {
	return sugaredPtr
}

// Sync 同步刷新日志缓冲区
func Sync() error {
	err := loggerPtr.Sync()
	if isIgnoredSyncErr(err) {
		return nil
	}
	return err
}

// Close 关闭所有文件写入器并刷新缓冲
func Close() error {
	err := Sync()
	activeCore.swap(fallbackCore, nil)
	return err
}

func isIgnoredSyncErr(err error) bool {
	if err == nil {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "invalid argument") ||
		strings.Contains(msg, "inappropriate ioctl for device") ||
		strings.Contains(msg, "bad file descriptor")
}

// ================= 日志输出接口 =================

// Debug 输出 Debug 级别结构化日志
func Debug(msg string, fields ...Field) {
	loggerPtr.Debug(msg, fields...)
}

// Info 输出 Info 级别结构化日志
func Info(msg string, fields ...Field) {
	loggerPtr.Info(msg, fields...)
}

// Warn 输出 Warn 级别结构化日志
func Warn(msg string, fields ...Field) {
	loggerPtr.Warn(msg, fields...)
}

// Error 输出 Error 级别结构化日志
func Error(msg string, fields ...Field) {
	loggerPtr.Error(msg, fields...)
}

// DPanic 输出 DPanic 级别结构化日志
func DPanic(msg string, fields ...Field) {
	loggerPtr.DPanic(msg, fields...)
}

// Panic 输出 Panic 级别结构化日志并触发 panic
func Panic(msg string, fields ...Field) {
	loggerPtr.Panic(msg, fields...)
}

// Fatal 输出 Fatal 级别结构化日志并调用 os.Exit(1)
func Fatal(msg string, fields ...Field) {
	loggerPtr.Fatal(msg, fields...)
}

// ================= 格式化输出接口 =================

// Debugf 输出 Debug 级别格式化日志
func Debugf(template string, args ...interface{}) {
	sugaredPtr.Debugf(template, args...)
}

// Infof 输出 Info 级别格式化日志
func Infof(template string, args ...interface{}) {
	sugaredPtr.Infof(template, args...)
}

// Warnf 输出 Warn 级别格式化日志
func Warnf(template string, args ...interface{}) {
	sugaredPtr.Warnf(template, args...)
}

// Errorf 输出 Error 级别格式化日志
func Errorf(template string, args ...interface{}) {
	sugaredPtr.Errorf(template, args...)
}

// ================= 常用字段封装 =================

// String 字符串字段
func String(key string, val string) Field {
	return zap.String(key, val)
}

// Int 整型字段
func Int(key string, val int) Field {
	return zap.Int(key, val)
}

// Int64 64位整型字段
func Int64(key string, val int64) Field {
	return zap.Int64(key, val)
}

// Uint64 64位无符号整型字段
func Uint64(key string, val uint64) Field {
	return zap.Uint64(key, val)
}

// Float64 浮点数字段
func Float64(key string, val float64) Field {
	return zap.Float64(key, val)
}

// Bool 布尔字段
func Bool(key string, val bool) Field {
	return zap.Bool(key, val)
}

// Any 任意类型字段
func Any(key string, val interface{}) Field {
	return zap.Any(key, val)
}

// Err 错误字段
func Err(err error) Field {
	return zap.Error(err)
}

// Strings 字符串切片字段
func Strings(key string, val []string) Field {
	return zap.Strings(key, val)
}

// Int8 8位整型字段
func Int8(key string, val int8) Field {
	return zap.Int8(key, val)
}

// Time 时间字段
func Time(key string, val time.Time) Field {
	return zap.Time(key, val)
}

// Duration 时间间隔字段
func Duration(key string, val time.Duration) Field {
	return zap.Duration(key, val)
}
