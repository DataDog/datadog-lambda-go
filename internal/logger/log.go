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
func Error(message string, err error) {

	type logStructure struct {
		Level      string `json:"error"`
		Message    string `json:"message"`
		InnerError string `json:"innerError,omitempty"`
	}

	var innerError string
	if err != nil {
		innerError = err.Error()
	}

	finalMessage := logStructure{
		Level:      "error",
		Message:    fmt.Sprintf("datadog: %s", message),
		InnerError: innerError,
	}
	result, _ := json.Marshal(finalMessage)

	log.Println(string(result))
}

// Debug logs a structured lgo message to stdout
func Debug(message string) {
	if logLevel > LevelDebug {
		return
	}
	type logStructure struct {
		Level   string `json:"level"`
		Message string `json:"message"`
	}
	finalMessage := logStructure{
		Level:   "debug",
		Message: fmt.Sprintf("datadog: %s", message),
	}

	result, _ := json.Marshal(finalMessage)

	log.Println(string(result))
}
