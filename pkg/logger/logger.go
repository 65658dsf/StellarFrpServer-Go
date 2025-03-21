package logger

import (
	"os"
	"path/filepath"
	"time"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"stellarfrp/config"
)

// Logger 封装zap日志库
type Logger struct {
	*zap.Logger
	lumberJackLogger *lumberjack.Logger
}

// NewLogger 创建一个新的日志记录器
func NewLogger(level string) *Logger {
	return NewLoggerWithConfig(level, config.LogFileConfig{})
}

// NewLoggerWithConfig 使用配置创建一个新的日志记录器
func NewLoggerWithConfig(level string, logFileConfig config.LogFileConfig) *Logger {
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

	// 获取当前日期作为日志文件名
	currentTime := time.Now()
	logFileName := filepath.Join(filepath.Dir(logFileConfig.Path), currentTime.Format("2006-01-02")+".log")

	// 创建编码器配置
	encoderConfig := zapcore.EncoderConfig{
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
	}

	// 创建控制台输出
	consoleEncoder := zapcore.NewJSONEncoder(encoderConfig)
	consoleOutput := zapcore.Lock(os.Stdout)
	consoleLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapLevel
	})

	// 创建核心
	var cores []zapcore.Core
	cores = append(cores, zapcore.NewCore(consoleEncoder, consoleOutput, consoleLevel))

	// 声明lumberJackLogger变量
	var lumberJackLogger *lumberjack.Logger

	// 如果启用了文件日志，添加文件输出
	if logFileConfig.Enabled && logFileConfig.Path != "" {
		// 确保日志目录存在
		logDir := filepath.Dir(logFileConfig.Path)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			panic(err)
		}

		// 配置lumberjack进行日志轮转
		lumberJackLogger = &lumberjack.Logger{
			Filename:   logFileName,
			MaxSize:    logFileConfig.MaxSize,    // 单个文件最大大小，单位MB
			MaxBackups: logFileConfig.MaxBackups, // 最大保留旧文件数量
			MaxAge:     logFileConfig.MaxAge,     // 最大保留天数
			Compress:   logFileConfig.Compress,   // 是否压缩
			LocalTime:  true,                     // 使用本地时间
		}

		// 启动定时任务，每天零点进行日志轮转
		go func() {
			for {
				now := time.Now()
				next := now.Add(time.Hour * 24)
				next = time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, next.Location())
				duration := next.Sub(now)
				timer := time.NewTimer(duration)
				<-timer.C

				// 更新日志文件名
				newLogFileName := filepath.Join(filepath.Dir(logFileConfig.Path), time.Now().Format("2006-01-02")+".log")
				// 先更新文件名，再进行轮转
				lumberJackLogger.Filename = newLogFileName
				lumberJackLogger.Rotate()
			}
		}()

		// 创建文件输出
		fileEncoder := zapcore.NewJSONEncoder(encoderConfig)
		fileOutput := zapcore.AddSync(lumberJackLogger)
		fileLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl >= zapLevel
		})

		cores = append(cores, zapcore.NewCore(fileEncoder, fileOutput, fileLevel))
	}

	// 创建日志记录器
	core := zapcore.NewTee(cores...)
	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	return &Logger{Logger: logger, lumberJackLogger: lumberJackLogger}
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
