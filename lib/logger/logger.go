package logger

import (
	"fmt"
	"hades/settings"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// 流式日志

type logLevel int

// log levels
const (
	DEBUG logLevel = iota
	INFO
	WARNING
	ERROR
	FATAL
)

const (
	flags              = log.LstdFlags // 日志前缀的标志 2009/01/23 01:23:23
	defaultCallerDepth = 2             // 默认调用深度2
	bufferSize         = 1e5           // 缓冲区大小
)

var (
	levelFlags = []string{"DEBUG", "INFO", "WARNING", "ERROR", "FATAL"} // 日志级别
)

type logEntry struct {
	msg   string
	level logLevel
}

type Logger struct {
	logFile   *os.File
	logger    *log.Logger
	entryChan chan *logEntry
	entryPool *sync.Pool
}

var DefaultLogger = NewStdoutLogger()

// NewStdoutLogger creates a logger which print msg to stdout
func NewStdoutLogger() *Logger {
	logger := &Logger{
		logFile:   nil,
		logger:    log.New(os.Stdout, "", flags),
		entryChan: make(chan *logEntry, bufferSize),
		entryPool: &sync.Pool{
			New: func() interface{} {
				return &logEntry{}
			},
		},
	}
	go func() {
		for e := range logger.entryChan {
			_ = logger.logger.Output(0, e.msg) // msg includes call stack, no need for calldepth
			logger.entryPool.Put(e)
		}
	}()
	return logger
}

// NewFileLogger creates a logger which print msg to stdout and log file
func NewFileLogger(settings *settings.LogConfig) (*Logger, error) {
	fileName := fmt.Sprintf("%s-%s.%s",
		settings.Name,
		time.Now().Format(settings.TimeFormat),
		settings.Ext)
	logFile, err := mustOpen(fileName, settings.Path)
	if err != nil {
		return nil, fmt.Errorf("logging.Join err: %s", err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	logger := &Logger{
		logFile:   logFile,
		logger:    log.New(mw, "", flags),
		entryChan: make(chan *logEntry, bufferSize),
		entryPool: &sync.Pool{
			New: func() interface{} {
				return &logEntry{}
			},
		},
	}
	go func() {
		for e := range logger.entryChan {
			logFilename := fmt.Sprintf("%s-%s.%s",
				settings.Name,
				time.Now().Format(settings.TimeFormat),
				settings.Ext)
			if path.Join(settings.Path, logFilename) != logger.logFile.Name() {
				logFile, err := mustOpen(logFilename, settings.Path)
				if err != nil {
					panic("open log " + logFilename + " failed: " + err.Error())
				}
				logger.logFile = logFile
				logger.logger = log.New(io.MultiWriter(os.Stdout, logFile), "", flags)
			}
			_ = logger.logger.Output(0, e.msg) // msg includes call stack, no need for calldepth
			logger.entryPool.Put(e)
		}
	}()
	return logger, nil
}

// Setup initializes DefaultLogger
func Setup(settings *settings.LogConfig) {
	logger, err := NewFileLogger(settings)
	if err != nil {
		panic(err)
	}
	DefaultLogger = logger
}

// Output sends a msg to logger
func (logger *Logger) Output(level logLevel, callerDepth int, msg string) {
	var formattedMsg string
	_, file, line, ok := runtime.Caller(callerDepth)
	if ok {
		formattedMsg = fmt.Sprintf("[%s][%s:%d] %s", levelFlags[level], filepath.Base(file), line, msg)
	} else {
		formattedMsg = fmt.Sprintf("[%s] %s", levelFlags[level], msg)
	}
	entry := logger.entryPool.Get().(*logEntry)
	entry.msg = formattedMsg
	entry.level = level
	logger.entryChan <- entry
}

// Debug logs debug message through DefaultLogger
func Debug(v ...interface{}) {
	msg := fmt.Sprintln(v...)
	DefaultLogger.Output(DEBUG, defaultCallerDepth, msg)
}

// Debugf logs debug message through DefaultLogger
func Debugf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	DefaultLogger.Output(DEBUG, defaultCallerDepth, msg)
}

// Info logs message through DefaultLogger
func Info(v ...interface{}) {
	msg := fmt.Sprintln(v...)
	DefaultLogger.Output(INFO, defaultCallerDepth, msg)
}

// Infof logs message through DefaultLogger
func Infof(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	DefaultLogger.Output(INFO, defaultCallerDepth, msg)
}

// Warn logs warning message through DefaultLogger
func Warn(v ...interface{}) {
	msg := fmt.Sprintln(v...)
	DefaultLogger.Output(WARNING, defaultCallerDepth, msg)
}

// Error logs error message through DefaultLogger
func Error(v ...interface{}) {
	msg := fmt.Sprintln(v...)
	DefaultLogger.Output(ERROR, defaultCallerDepth, msg)
}

// Errorf logs error message through DefaultLogger
func Errorf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	DefaultLogger.Output(ERROR, defaultCallerDepth, msg)
}

// Fatal prints error message then stop the program
func Fatal(v ...interface{}) {
	msg := fmt.Sprintln(v...)
	DefaultLogger.Output(FATAL, defaultCallerDepth, msg)
}

func mustOpen(fileName, dir string) (*os.File, error) {
	_, err := os.Stat(dir)
	if os.IsPermission(err) {
		return nil, fmt.Errorf("permission denied dir: %s", dir)
	}
	if os.IsNotExist(err) {
		if err = os.MkdirAll(dir, os.ModePerm); err != nil {
			return nil, fmt.Errorf("error during mkdir %s: %s", dir, err)
		}
	}
	f, err := os.OpenFile(path.Join(dir, fileName), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("error opening file %s: %s", fileName, err)
	}
	return f, nil
}
