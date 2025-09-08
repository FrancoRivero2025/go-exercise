package log

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	LevelDebug = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

type Logger interface {
	SetLevel(level int)
	SetOutput(w io.Writer)
	SetOutputToFile(filename string) error
	GetLevel() int
	Debug(message string, v ...interface{})
	Info(message string, v ...interface{})
	Warn(message string, v ...interface{})
	Error(message string, v ...interface{})
	Fatal(message string, v ...interface{})
}

type RealLogger struct {
	level    int
	output   io.Writer
	instance *log.Logger
	mu       sync.Mutex
	testing  bool
}

var (
	once     sync.Once
	instance Logger
)

func GetInstance() Logger {
	once.Do(func() {
		instance = &RealLogger{
			level:    LevelInfo,
			output:   os.Stdout,
			instance: log.New(os.Stdout, "", 0),
		}
	})
	return instance
}

func SetInstance(l Logger) {
	instance = l
}

func (l *RealLogger) SetTestingMode(testing bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.testing = testing
}

func (l *RealLogger) Fatal(message string, v ...interface{}) {
	l.logMessage(LevelFatal, message, v...)
	if !l.testing {
		os.Exit(1)
	}
}

func (l *RealLogger) SetLevel(level int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

func (l *RealLogger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = w
	l.instance.SetOutput(w)
}

func (l *RealLogger) SetOutputToFile(filename string) error {
    file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    if err != nil {
        return err
    }
    l.SetOutput(file)
    return nil
}

func (l *RealLogger) GetLevel() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.level
}

func (l *RealLogger) getLevelName(level int) string {
	switch level {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

func (l *RealLogger) getCallerInfo() string {
	_, file, line, ok := runtime.Caller(3)
	if !ok {
		return "unknown:0"
	}
	
	parts := strings.Split(file, "/")
	if len(parts) > 0 {
		file = parts[len(parts)-1]
	}
	
	return fmt.Sprintf("%s:%d", file, line)
}

func (l *RealLogger) logMessage(level int, message string, v ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	callerInfo := l.getCallerInfo()
	levelName := l.getLevelName(level)

	var formattedMessage string
	if len(v) > 0 {
		formattedMessage = fmt.Sprintf(message, v...)
	} else {
		formattedMessage = message
	}

	logLine := fmt.Sprintf("%s [%s] %s - %s", 
		timestamp, levelName, callerInfo, formattedMessage)

	l.instance.Println(logLine)
}

func (l *RealLogger) Debug(message string, v ...interface{}) {
	l.logMessage(LevelDebug, message, v...)
}

func (l *RealLogger) Info(message string, v ...interface{}) {
	l.logMessage(LevelInfo, message, v...)
}

func (l *RealLogger) Warn(message string, v ...interface{}) {
	l.logMessage(LevelWarn, message, v...)
}

func (l *RealLogger) Error(message string, v ...interface{}) {
	l.logMessage(LevelError, message, v...)
}
