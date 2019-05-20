package internal

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
)

// Handler wraps an existing handler
func Handler(handler interface{}) interface{} {

	err := validateHandler(handler)
	if err != nil {
		// TODO: Log error
		return handler
	}

	return func(ctx context.Context, msg json.RawMessage) (interface{}, error) {
		return nil, nil
	}
}

func validateHandler(handler interface{}) error {
	handlerType := reflect.TypeOf(handler)
	if handlerType.Kind() != reflect.Func {
		return errors.New("handler is not a function")
	}

	if handlerType.NumIn() == 2 {
		contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
		firstArgType := handlerType.In(0)
		if !firstArgType.Implements(contextType) {
			return errors.New("handler should take context as first argument")
		}
	}
	if handlerType.NumIn() > 2 {
		return errors.New("handler takes too many arguments")
	}

	errorType := reflect.TypeOf((*error)(nil)).Elem()

	if handlerType.NumOut() > 2 {
		return errors.New("handler returns more than two values")
	}
	if handlerType.NumOut() > 0 {
		rt := handlerType.Out(handlerType.NumOut() - 1) // Last returned value
		if !rt.Implements(errorType) {
			return errors.New("handler doesn't return error as it's last value")
		}
	}
	return nil
}
