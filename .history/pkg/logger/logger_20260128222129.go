package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger

// InitLogger инициализирует глобальный logger
func InitLogger(environment string) error {
	var config zap.Config

	if environment == "production" {
		// Production: JSON формат, без stacktrace для info
		config = zap.NewProductionConfig()
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		// Development: красивый вывод с цветами
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	// Настройка уровня логирования из env
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel != "" {
		var level zapcore.Level
		if err := level.UnmarshalText([]byte(logLevel)); err == nil {
			config.Level = zap.NewAtomicLevelAt(level)
		}
	}

	// Создание logger
	logger, err := config.Build(
		zap.AddCallerSkip(1), // Пропустить обертку для корректного caller
		zap.AddStacktrace(zapcore.ErrorLevel), // Stacktrace только для errors
	)
	if err != nil {
		return err
	}

	Log = logger
	return nil
}

// Sync сбрасывает буферы
func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}

// Helper функции для удобства

func Debug(msg string, fields ...zap.Field) {
	Log.Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	Log.Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	Log.Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	Log.Error(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	Log.Fatal(msg, fields...)
}

func Panic(msg string, fields ...zap.Field) {
	Log.Panic(msg, fields...)
}

// With создает logger с предустановленными полями
func With(fields ...zap.Field) *zap.Logger {
	return Log.With(fields...)
}
