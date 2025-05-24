package util

import (
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/hashicorp/go-hclog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ZapHCLogAdapter struct {
	zapLogger *zap.Logger
	name      string
}

func NewZapAdapter(zapLogger *zap.Logger, name string) *ZapHCLogAdapter {
	return &ZapHCLogAdapter{
		zapLogger: zapLogger.WithOptions(zap.AddCallerSkip(2)),
		name:      name,
	}
}

func (z *ZapHCLogAdapter) Log(level hclog.Level, msg string, args ...interface{}) {
	fields := make([]zap.Field, 0, len(args)/2)
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			key, ok := args[i].(string)
			if !ok {
				key = fmt.Sprintf("%v", args[i])
			}
			fields = append(fields, zap.Any(key, args[i+1]))
		}
	}

	switch level {
	case hclog.Trace:
		z.zapLogger.Debug(msg, fields...)
	case hclog.Debug:
		z.zapLogger.Debug(msg, fields...)
	case hclog.Info:
		z.zapLogger.Info(msg, fields...)
	case hclog.Warn:
		z.zapLogger.Warn(msg, fields...)
	case hclog.Error:
		z.zapLogger.Error(msg, fields...)
	}
}

func (z *ZapHCLogAdapter) Trace(msg string, args ...interface{}) {
	z.Log(hclog.Trace, msg, args...)
}

func (z *ZapHCLogAdapter) Debug(msg string, args ...interface{}) {
	z.Log(hclog.Debug, msg, args...)
}

func (z *ZapHCLogAdapter) Info(msg string, args ...interface{}) {
	z.Log(hclog.Info, msg, args...)
}

func (z *ZapHCLogAdapter) Warn(msg string, args ...interface{}) {
	z.Log(hclog.Warn, msg, args...)
}

func (z *ZapHCLogAdapter) Error(msg string, args ...interface{}) {
	z.Log(hclog.Error, msg, args...)
}

func (z *ZapHCLogAdapter) IsTrace() bool {
	return z.zapLogger.Core().Enabled(zapcore.DebugLevel)
}

func (z *ZapHCLogAdapter) IsDebug() bool {
	return z.zapLogger.Core().Enabled(zapcore.DebugLevel)
}

func (z *ZapHCLogAdapter) IsInfo() bool {
	return z.zapLogger.Core().Enabled(zapcore.InfoLevel)
}

func (z *ZapHCLogAdapter) IsWarn() bool {
	return z.zapLogger.Core().Enabled(zapcore.WarnLevel)
}

func (z *ZapHCLogAdapter) IsError() bool {
	return z.zapLogger.Core().Enabled(zapcore.ErrorLevel)
}

func (z *ZapHCLogAdapter) ImpliedArgs() []interface{} {
	return nil
}

func (z *ZapHCLogAdapter) With(args ...interface{}) hclog.Logger {
	if len(args) == 0 {
		return z
	}

	fields := make([]zap.Field, 0, len(args)/2)
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			key, ok := args[i].(string)
			if !ok {
				key = fmt.Sprintf("%v", args[i])
			}
			fields = append(fields, zap.Any(key, args[i+1]))
		}
	}

	newLogger := z.zapLogger.With(fields...)
	return &ZapHCLogAdapter{
		zapLogger: newLogger,
	}
}

func (z *ZapHCLogAdapter) Name() string {
	return z.name
}

func (z *ZapHCLogAdapter) Named(name string) hclog.Logger {
	return &ZapHCLogAdapter{zapLogger: z.zapLogger.Named(name), name: name}
}

func (z *ZapHCLogAdapter) ResetNamed(name string) hclog.Logger {
	return &ZapHCLogAdapter{
		zapLogger: z.zapLogger,
		name:      name,
	}
}

func (z *ZapHCLogAdapter) SetLevel(level hclog.Level) {
}

func (z *ZapHCLogAdapter) GetLevel() hclog.Level {
	return hclog.LevelFromString(z.zapLogger.Level().String())
}

func (z *ZapHCLogAdapter) StandardLogger(opts *hclog.StandardLoggerOptions) *log.Logger {
	return log.New(z, "", log.LstdFlags)
}

func (z *ZapHCLogAdapter) StandardWriter(opts *hclog.StandardLoggerOptions) io.Writer {
	return z
}

func (z *ZapHCLogAdapter) Write(p []byte) (n int, err error) {
	s := strings.TrimSpace(string(p))
	z.Info(s)
	return len(p), nil
}
