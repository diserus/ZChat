package logger

import (
	"os"
	"zchat/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(cfg config.LogConfig) *zap.Logger {
	var (
		level  zapcore.Level
		encode zapcore.Encoder
	)

	if cfg.LogLevel == "debug" {
		level = zapcore.DebugLevel
		encode = zapcore.NewConsoleEncoder(developmentEncoderConfig())
	} else {
		level = zapcore.InfoLevel
		encode = zapcore.NewJSONEncoder(productionEncoderConfig())
	}
	core := zapcore.NewCore(encode, zapcore.AddSync(os.Stdout), level)

	return zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
}

func productionEncoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}

func developmentEncoderConfig() zapcore.EncoderConfig {
	cfg := zap.NewDevelopmentEncoderConfig()
	cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	cfg.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05")
	return cfg
}
