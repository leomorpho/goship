package logging

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/labstack/gommon/log"
)

type EchoLogger struct {
	slog *slog.Logger
}

func NewEchoLogger(s *slog.Logger) *EchoLogger {
	return &EchoLogger{slog: s}
}

func (l *EchoLogger) Output() io.Writer { return io.Discard }
func (l *EchoLogger) SetOutput(w io.Writer) {}
func (l *EchoLogger) Prefix() string { return "" }
func (l *EchoLogger) SetPrefix(p string) {}
func (l *EchoLogger) Level() log.Lvl { return log.INFO }
func (l *EchoLogger) SetLevel(v log.Lvl) {}
func (l *EchoLogger) SetHeader(h string) {}

func (l *EchoLogger) Print(i ...interface{}) { l.slog.Info(fmt.Sprint(i...)) }
func (l *EchoLogger) Printf(format string, args ...interface{}) { l.slog.Info(fmt.Sprintf(format, args...)) }
func (l *EchoLogger) Printj(j log.JSON) { l.slog.Info("json", "data", j) }

func (l *EchoLogger) Debug(i ...interface{}) { l.slog.Debug(fmt.Sprint(i...)) }
func (l *EchoLogger) Debugf(format string, args ...interface{}) { l.slog.Debug(fmt.Sprintf(format, args...)) }
func (l *EchoLogger) Debugj(j log.JSON) { l.slog.Debug("json", "data", j) }

func (l *EchoLogger) Info(i ...interface{}) { l.slog.Info(fmt.Sprint(i...)) }
func (l *EchoLogger) Infof(format string, args ...interface{}) { l.slog.Info(fmt.Sprintf(format, args...)) }
func (l *EchoLogger) Infoj(j log.JSON) { l.slog.Info("json", "data", j) }

func (l *EchoLogger) Warn(i ...interface{}) { l.slog.Warn(fmt.Sprint(i...)) }
func (l *EchoLogger) Warnf(format string, args ...interface{}) { l.slog.Warn(fmt.Sprintf(format, args...)) }
func (l *EchoLogger) Warnj(j log.JSON) { l.slog.Warn("json", "data", j) }

func (l *EchoLogger) Error(i ...interface{}) { l.slog.Error(fmt.Sprint(i...)) }
func (l *EchoLogger) Errorf(format string, args ...interface{}) { l.slog.Error(fmt.Sprintf(format, args...)) }
func (l *EchoLogger) Errorj(j log.JSON) { l.slog.Error("json", "data", j) }

func (l *EchoLogger) Fatal(i ...interface{}) { l.slog.Error(fmt.Sprint(i...)); panic("fatal") }
func (l *EchoLogger) Fatalf(format string, args ...interface{}) { l.slog.Error(fmt.Sprintf(format, args...)); panic("fatal") }
func (l *EchoLogger) Fatalj(j log.JSON) { l.slog.Error("json", "data", j); panic("fatal") }

func (l *EchoLogger) Panic(i ...interface{}) { l.slog.Error(fmt.Sprint(i...)); panic(fmt.Sprint(i...)) }
func (l *EchoLogger) Panicf(format string, args ...interface{}) { l.slog.Error(fmt.Sprintf(format, args...)); panic(fmt.Sprintf(format, args...)) }
func (l *EchoLogger) Panicj(j log.JSON) { l.slog.Error("json", "data", j); panic("panic") }
