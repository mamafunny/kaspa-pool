package common

import (
	"github.com/mattn/go-colorable"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func ConfigureZap(level zapcore.Level) *zap.Logger {
	pe := zap.NewProductionEncoderConfig()
	pe.EncodeTime = zapcore.RFC3339TimeEncoder
	consoleEncoder := zapcore.NewConsoleEncoder(pe)

	core := zapcore.NewCore(consoleEncoder, zapcore.AddSync(colorable.NewColorableStdout()), level)
	return zap.New(core)
}
