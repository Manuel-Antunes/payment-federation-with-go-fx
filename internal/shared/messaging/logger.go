package messaging

import (
	"github.com/ThreeDotsLabs/watermill"
	"go.uber.org/zap"
)

// zapLogger adapta o *zap.Logger ao watermill.LoggerAdapter, para que os logs
// do barramento saiam no mesmo formato estruturado do resto do app.
type zapLogger struct {
	log *zap.Logger
}

var _ watermill.LoggerAdapter = (*zapLogger)(nil)

// NewWatermillLogger devolve um LoggerAdapter respaldado por zap.
func NewWatermillLogger(log *zap.Logger) watermill.LoggerAdapter {
	return &zapLogger{log: log.Named("watermill")}
}

func (l *zapLogger) Error(msg string, err error, fields watermill.LogFields) {
	l.log.Error(msg, append(toZapFields(fields), zap.Error(err))...)
}

func (l *zapLogger) Info(msg string, fields watermill.LogFields) {
	l.log.Info(msg, toZapFields(fields)...)
}

func (l *zapLogger) Debug(msg string, fields watermill.LogFields) {
	l.log.Debug(msg, toZapFields(fields)...)
}

// Trace é mapeado para Debug (zap não tem nível Trace).
func (l *zapLogger) Trace(msg string, fields watermill.LogFields) {
	l.log.Debug(msg, toZapFields(fields)...)
}

func (l *zapLogger) With(fields watermill.LogFields) watermill.LoggerAdapter {
	return &zapLogger{log: l.log.With(toZapFields(fields)...)}
}

func toZapFields(fields watermill.LogFields) []zap.Field {
	if len(fields) == 0 {
		return nil
	}
	zf := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zf = append(zf, zap.Any(k, v))
	}
	return zf
}
