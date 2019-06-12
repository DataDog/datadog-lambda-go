package logger

import (
	"encoding/json"
	"fmt"
	"log"
)

// LogLevel represents the level of logging that should be performed
type LogLevel int

const (
	// LevelDebug logs all information
	LevelDebug LogLevel = iota
	// LevelError only logs errors
	LevelError LogLevel = iota
)

var (
	logLevel = LevelError
)

// SetLogLevel set the level of logging for the ddlambda
func SetLogLevel(ll LogLevel) {
	logLevel = ll
}

// Error logs a structured error message to stdout
func Error(err error) {

	type logStructure struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}

	finalMessage := logStructure{
		Status:  "error",
		Message: fmt.Sprintf("datadog: %s", err.Error()),
	}
	result, _ := json.Marshal(finalMessage)

	log.Println(string(result))
}

// Debug logs a structured log message to stdout
func Debug(message string) {
	if logLevel > LevelDebug {
		return
	}
	type logStructure struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	finalMessage := logStructure{
		Status:  "debug",
		Message: fmt.Sprintf("datadog: %s", message),
	}

	result, _ := json.Marshal(finalMessage)

	log.Println(string(result))
}
