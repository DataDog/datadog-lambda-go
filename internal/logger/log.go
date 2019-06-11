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
		InnerError error  `json:"innerError,omitempty"`
	}

	finalMessage := logStructure{
		Message:    fmt.Sprintf("datadog: %s", message),
		InnerError: err,
	}
	result, _ := json.Marshal(finalMessage)

	log.Println(string(result))
}
