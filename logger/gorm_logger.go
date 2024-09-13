package logger

import (
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm/logger"
)

type GormZapLogger struct {
	ZapLogger *zap.Logger
}

func NewGormZapLogger(zapLogger *zap.Logger) *GormZapLogger {
	return &GormZapLogger{ZapLogger: zapLogger}
}

// LogMode sets the log level
func (l *GormZapLogger) LogMode(level logger.LogLevel) logger.Interface {
	// Handle log levels mapping between GORM and Zap if needed
	return l
}

// Info logs info messages
func (l *GormZapLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	l.ZapLogger.Sugar().Infof(msg, data...)
}

// Warn logs warning messages
func (l *GormZapLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	l.ZapLogger.Sugar().Warnf(msg, data...)
}

// Error logs error messages
func (l *GormZapLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	l.ZapLogger.Sugar().Errorf(msg, data...)
}

// Trace logs SQL queries and execution times
func (l *GormZapLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()

	switch {
	case err != nil:
		l.ZapLogger.Sugar().Errorf("[%.3fms] [rows:%v] %s %s", float64(elapsed.Milliseconds()), rows, sql, err.Error())
	case elapsed > 200*time.Millisecond:
		l.ZapLogger.Sugar().Warnf("[%.3fms] [rows:%v] %s", float64(elapsed.Milliseconds()), rows, sql)
	default:
		l.ZapLogger.Sugar().Infof("[%.3fms] [rows:%v] %s", float64(elapsed.Milliseconds()), rows, sql)
	}
}
