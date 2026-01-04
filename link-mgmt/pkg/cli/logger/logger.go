package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	logger  *log.Logger
	logFile *os.File
)

func init() {
	// Create log directory if it doesn't exist
	logDir := "tmp"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		// If we can't create log dir, just use stderr
		logger = log.New(os.Stderr, "[CLI] ", log.LstdFlags|log.Lshortfile)
		return
	}

	// Create log file with timestamp
	logFileName := filepath.Join(logDir, fmt.Sprintf("cli-%s.log", time.Now().Format("20060102-150405")))

	var err error
	logFile, err = os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		// If we can't open log file, use stderr
		logger = log.New(os.Stderr, "[cli] ", log.LstdFlags|log.Lshortfile)
		return
	}

	// Create logger that writes only to file
	logger = log.New(logFile, "[cli] ", log.LstdFlags|log.Lshortfile)
}

// Log writes a log message
func Log(format string, v ...interface{}) {
	if logger != nil {
		logger.Printf(format, v...)
	}
}

// LogError writes an error log message
func LogError(err error, format string, v ...interface{}) {
	if logger != nil {
		msg := fmt.Sprintf(format, v...)
		logger.Printf("ERROR: %s: %v", msg, err)
	}
}

// CloseLog closes the log file
func CloseLog() {
	if logFile != nil {
		logFile.Close()
	}
}
