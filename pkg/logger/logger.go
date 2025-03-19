package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger 封装zap日志库
type Logger struct {
	*zap.Logger
}

// NewLogger 创建一个新的日志记录器
func NewLogger(level string) *Logger {
	// 解析日志级别
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	// 配置日志
	cfg := zap.Config{
		Level:       zap.NewAtomicLevelAt(zapLevel),
		Development: false,
		Encoding:    "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	// 创建日志记录器
	logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}

	return &Logger{Logger: logger}
}

// Info 记录信息级别日志
func (l *Logger) Info(msg string, fields ...interface{}) {
	l.Logger.Info(msg, fieldsToZapFields(fields...)...)
}

// Debug 记录调试级别日志
func (l *Logger) Debug(msg string, fields ...interface{}) {
	l.Logger.Debug(msg, fieldsToZapFields(fields...)...)
}

// Warn 记录警告级别日志
func (l *Logger) Warn(msg string, fields ...interface{}) {
	l.Logger.Warn(msg, fieldsToZapFields(fields...)...)
}

// Error 记录错误级别日志
func (l *Logger) Error(msg string, fields ...interface{}) {
	l.Logger.Error(msg, fieldsToZapFields(fields...)...)
}

// Fatal 记录致命错误日志并退出程序
func (l *Logger) Fatal(msg string, fields ...interface{}) {
	l.Logger.Fatal(msg, fieldsToZapFields(fields...)...)
}

// 将通用接口转换为zap字段
func fieldsToZapFields(fields ...interface{}) []zap.Field {
	zapFields := make([]zap.Field, 0, len(fields))
	for i := 0; i < len(fields); i++ {
		switch f := fields[i].(type) {
		case error:
			zapFields = append(zapFields, zap.Error(f))
		case string:
			if i+1 < len(fields) {
				zapFields = append(zapFields, zap.Any(f, fields[i+1]))
				i++
			}
		default:
			zapFields = append(zapFields, zap.Any("field", f))
		}
	}
	return zapFields
}
