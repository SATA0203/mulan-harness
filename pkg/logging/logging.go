package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"runtime/debug"
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

func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "debug"
	case INFO:
		return "info"
	case WARN:
		return "warn"
	case ERROR:
		return "error"
	case FATAL:
		return "fatal"
	default:
		return "unknown"
	}
}

// ParseLogLevel 解析日志级别字符串
func ParseLogLevel(level string) LogLevel {
	switch level {
	case "debug", "DEBUG":
		return DEBUG
	case "info", "INFO":
		return INFO
	case "warn", "WARN", "warning", "WARNING":
		return WARN
	case "error", "ERROR":
		return ERROR
	case "fatal", "FATAL":
		return FATAL
	default:
		return INFO
	}
}

// LogFormat 日志格式
type LogFormat string

const (
	JSONFormat LogFormat = "json"
	TextFormat LogFormat = "text"
)

// AccessLogEntry 访问日志条目
type AccessLogEntry struct {
	Timestamp    string                 `json:"timestamp"`
	Method       string                 `json:"method"`
	Path         string                 `json:"path"`
	Host         string                 `json:"host"`
	RemoteAddr   string                 `json:"remote_addr"`
	UserAgent    string                 `json:"user_agent,omitempty"`
	StatusCode   int                    `json:"status_code"`
	ResponseBody int64                  `json:"response_size"`
	RequestBody  int64                  `json:"request_size"`
	Duration     int64                  `json:"duration_ms"`
	UpstreamAddr string                 `json:"upstream_addr,omitempty"`
	TraceID      string                 `json:"trace_id,omitempty"`
	SpanID       string                 `json:"span_id,omitempty"`
	Headers      map[string]string      `json:"headers,omitempty"`
	Error        string                 `json:"error,omitempty"`
	Extra        map[string]interface{} `json:"extra,omitempty"`
}

// Logger 日志记录器
type Logger struct {
	level      LogLevel
	format     LogFormat
	output     io.Writer
	errorOutput io.Writer
	sampleRate float64
	baseLogger *log.Logger
}

// NewLogger 创建日志记录器
func NewLogger(level LogLevel, format LogFormat, output io.Writer, sampleRate float64) *Logger {
	if output == nil {
		output = os.Stdout
	}

	logger := &Logger{
		level:       level,
		format:      format,
		output:      output,
		errorOutput: os.Stderr,
		sampleRate:  sampleRate,
	}

	if sampleRate < 1.0 {
		// 如果采样率小于 1，使用双输出：错误日志始终输出，其他日志采样
		logger.errorOutput = os.Stderr
	}

	return logger
}

// ShouldSample 判断是否应该采样这条日志
func (l *Logger) ShouldSample() bool {
	if l.sampleRate >= 1.0 {
		return true
	}
	// 简单随机采样
	return float64(time.Now().UnixNano()%1000)/1000.0 < l.sampleRate
}

// logInternal 内部日志方法
func (l *Logger) logInternal(level LogLevel, msg string, fields map[string]interface{}) {
	if level < l.level {
		return
	}

	// 对于非 ERROR/FATAL 级别，应用采样
	if level != ERROR && level != FATAL && !l.ShouldSample() {
		return
	}

	var output []byte

	if l.format == JSONFormat {
		entry := map[string]interface{}{
			"timestamp": time.Now().Format(time.RFC3339),
			"level":     level.String(),
			"message":   msg,
		}
		for k, v := range fields {
			entry[k] = v
		}
		output, _ = json.Marshal(entry)
	} else {
		// Text 格式
		line := fmt.Sprintf("[%s] %s %s", time.Now().Format(time.RFC3339), level.String(), msg)
		for k, v := range fields {
			line += fmt.Sprintf(" %s=%v", k, v)
		}
		output = []byte(line)
	}

	if level >= ERROR {
		l.errorOutput.Write(append(output, '\n'))
	} else {
		l.output.Write(append(output, '\n'))
	}
}

// Debug 打印调试日志
func (l *Logger) Debug(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.logInternal(DEBUG, msg, f)
}

// Info 打印信息日志
func (l *Logger) Info(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.logInternal(INFO, msg, f)
}

// Warn 打印警告日志
func (l *Logger) Warn(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.logInternal(WARN, msg, f)
}

// Error 打印错误日志
func (l *Logger) Error(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.logInternal(ERROR, msg, f)
}

// Fatal 打印致命日志并退出
func (l *Logger) Fatal(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.logInternal(FATAL, msg, f)
	os.Exit(1)
}

// Panic 记录 panic 并重新抛出
func (l *Logger) Panic(msg string, fields ...map[string]interface{}) {
	l.Error(msg+" "+string(debug.Stack()), fields...)
	panic(msg)
}

// AccessLog 记录访问日志
func (l *Logger) AccessLog(entry *AccessLogEntry) {
	if l.level > INFO {
		return
	}

	if !l.ShouldSample() {
		return
	}

	entry.Timestamp = time.Now().Format(time.RFC3339)

	if l.format == JSONFormat {
		data, _ := json.Marshal(entry)
		l.output.Write(append(data, '\n'))
	} else {
		line := fmt.Sprintf("[%s] %s %s %s %d %dms",
			entry.Timestamp,
			entry.RemoteAddr,
			entry.Method,
			entry.Path,
			entry.StatusCode,
			entry.Duration,
		)
		if entry.UpstreamAddr != "" {
			line += fmt.Sprintf(" upstream:%s", entry.UpstreamAddr)
		}
		l.output.Write([]byte(line + "\n"))
	}
}

// SetLevel 动态设置日志级别
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

// SetSampleRate 动态设置采样率
func (l *Logger) SetSampleRate(rate float64) {
	if rate < 0 {
		rate = 0
	}
	if rate > 1 {
		rate = 1
	}
	l.sampleRate = rate
}

// DefaultLogger 默认日志记录器
var DefaultLogger *Logger

func init() {
	DefaultLogger = NewLogger(INFO, JSONFormat, os.Stdout, 1.0)
}

// Package-level convenience functions

// Debug 打印调试日志
func Debug(msg string, fields ...map[string]interface{}) {
	DefaultLogger.Debug(msg, fields...)
}

// Info 打印信息日志
func Info(msg string, fields ...map[string]interface{}) {
	DefaultLogger.Info(msg, fields...)
}

// Warn 打印警告日志
func Warn(msg string, fields ...map[string]interface{}) {
	DefaultLogger.Warn(msg, fields...)
}

// Error 打印错误日志
func Error(msg string, fields ...map[string]interface{}) {
	DefaultLogger.Error(msg, fields...)
}

// Fatal 打印致命日志
func Fatal(msg string, fields ...map[string]interface{}) {
	DefaultLogger.Fatal(msg, fields...)
}

// AccessLog 记录访问日志
func AccessLog(entry *AccessLogEntry) {
	DefaultLogger.AccessLog(entry)
}
