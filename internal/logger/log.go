package logger

import (
	"encoding/json"
	"fmt"
	"log"
)

// LogError logs a structured error message to stdout
func LogError(message string, err error) {

	type logStructure struct {
		Message    string `json:"error"`
		InnerError string `json:"innerError,omitempty"`
	}

	var innerError string
	if err != nil {
		innerError = err.Error()
	}

	finalMessage := logStructure{
		Message:    fmt.Sprintf("datadog: %s", message),
		InnerError: innerError,
	}
	result, _ := json.Marshal(finalMessage)

	log.Println(string(result))
}
