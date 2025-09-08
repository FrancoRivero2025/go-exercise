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

type Logger struct {
	level    int
	output   io.Writer
	instance *log.Logger
	mu       sync.Mutex
}

var (
	once     sync.Once
	instance *Logger
)

func GetInstance() *Logger {
	once.Do(func() {
		instance = &Logger{
			level:    LevelInfo,
			output:   os.Stdout,
			instance: log.New(os.Stdout, "", 0),
		}
	})
	return instance
}

func (l *Logger) SetLevel(level int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = w
	l.instance.SetOutput(w)
}

func (l *Logger) SetOutputToFile(filename string) error {
    file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    if err != nil {
        return err
    }
    l.SetOutput(file)
    return nil
}

func (l *Logger) GetLevel() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.level
}

func (l *Logger) getLevelName(level int) string {
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

func (l *Logger) getCallerInfo() string {
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

func (l *Logger) logMessage(level int, message string, v ...interface{}) {
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

func (l *Logger) Debug(message string, v ...interface{}) {
	l.logMessage(LevelDebug, message, v...)
}

func (l *Logger) Info(message string, v ...interface{}) {
	l.logMessage(LevelInfo, message, v...)
}

func (l *Logger) Warn(message string, v ...interface{}) {
	l.logMessage(LevelWarn, message, v...)
}

func (l *Logger) Error(message string, v ...interface{}) {
	l.logMessage(LevelError, message, v...)
}

func (l *Logger) Fatal(message string, v ...interface{}) {
	l.logMessage(LevelFatal, message, v...)
	os.Exit(1)
}
