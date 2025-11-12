package logger

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

// LogLevel 日志级别
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// Logger 结构化日志记录器
type Logger struct {
	level  LogLevel
	logger *log.Logger
}

var (
	defaultLogger *Logger
	levelNames    = map[LogLevel]string{
		DEBUG: "DEBUG",
		INFO:  "INFO",
		WARN:  "WARN",
		ERROR: "ERROR",
		FATAL: "FATAL",
	}
	levelColors = map[LogLevel]string{
		DEBUG: "\033[36m", // 青色
		INFO:  "\033[32m", // 绿色
		WARN:  "\033[33m", // 黄色
		ERROR: "\033[31m", // 红色
		FATAL: "\033[35m", // 紫色
	}
	resetColor = "\033[0m"
)

func init() {
	defaultLogger = NewLogger(INFO)
}

// NewLogger 创建新的日志记录器
func NewLogger(level LogLevel) *Logger {
	return &Logger{
		level:  level,
		logger: log.New(os.Stdout, "", 0),
	}
}

// SetLevel 设置日志级别
func SetLevel(level LogLevel) {
	defaultLogger.level = level
}

// formatMessage 格式化日志消息
func (l *Logger) formatMessage(level LogLevel, msg string, fields map[string]interface{}) string {
	// 获取调用者信息
	_, file, line, ok := runtime.Caller(3)
	caller := "unknown"
	if ok {
		// 只显示文件名，不显示完整路径
		parts := strings.Split(file, "/")
		if len(parts) > 0 {
			file = parts[len(parts)-1]
		}
		caller = fmt.Sprintf("%s:%d", file, line)
	}

	// 时间戳
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")

	// 构建基础消息
	levelName := levelNames[level]
	color := levelColors[level]

	var parts []string
	parts = append(parts, fmt.Sprintf("%s[%s]%s", color, levelName, resetColor))
	parts = append(parts, timestamp)
	parts = append(parts, caller)
	parts = append(parts, msg)

	// 添加字段
	if len(fields) > 0 {
		var fieldParts []string
		for k, v := range fields {
			fieldParts = append(fieldParts, fmt.Sprintf("%s=%v", k, v))
		}
		parts = append(parts, strings.Join(fieldParts, " "))
	}

	return strings.Join(parts, " | ")
}

// log 内部日志方法
func (l *Logger) log(level LogLevel, msg string, fields map[string]interface{}) {
	if level < l.level {
		return
	}

	message := l.formatMessage(level, msg, fields)
	l.logger.Println(message)

	// FATAL 级别退出程序
	if level == FATAL {
		os.Exit(1)
	}
}

// Debug 调试级别日志
func (l *Logger) Debug(msg string, fields ...map[string]interface{}) {
	f := make(map[string]interface{})
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(DEBUG, msg, f)
}

// Info 信息级别日志
func (l *Logger) Info(msg string, fields ...map[string]interface{}) {
	f := make(map[string]interface{})
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(INFO, msg, f)
}

// Warn 警告级别日志
func (l *Logger) Warn(msg string, fields ...map[string]interface{}) {
	f := make(map[string]interface{})
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(WARN, msg, f)
}

// Error 错误级别日志
func (l *Logger) Error(msg string, fields ...map[string]interface{}) {
	f := make(map[string]interface{})
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(ERROR, msg, f)
}

// Fatal 致命错误级别日志（会退出程序）
func (l *Logger) Fatal(msg string, fields ...map[string]interface{}) {
	f := make(map[string]interface{})
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(FATAL, msg, f)
}

// 全局便捷方法
func Debug(msg string, fields ...map[string]interface{}) {
	defaultLogger.Debug(msg, fields...)
}

func Info(msg string, fields ...map[string]interface{}) {
	defaultLogger.Info(msg, fields...)
}

func Warn(msg string, fields ...map[string]interface{}) {
	defaultLogger.Warn(msg, fields...)
}

func Error(msg string, fields ...map[string]interface{}) {
	defaultLogger.Error(msg, fields...)
}

func Fatal(msg string, fields ...map[string]interface{}) {
	defaultLogger.Fatal(msg, fields...)
}

// Debugf 格式化调试日志
func Debugf(format string, args ...interface{}) {
	defaultLogger.Debug(fmt.Sprintf(format, args...))
}

// Infof 格式化信息日志
func Infof(format string, args ...interface{}) {
	defaultLogger.Info(fmt.Sprintf(format, args...))
}

// Warnf 格式化警告日志
func Warnf(format string, args ...interface{}) {
	defaultLogger.Warn(fmt.Sprintf(format, args...))
}

// Errorf 格式化错误日志
func Errorf(format string, args ...interface{}) {
	defaultLogger.Error(fmt.Sprintf(format, args...))
}

// Fatalf 格式化致命错误日志
func Fatalf(format string, args ...interface{}) {
	defaultLogger.Fatal(fmt.Sprintf(format, args...))
}
