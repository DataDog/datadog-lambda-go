package internal

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateHandlerNotFunction(t *testing.T) {
	nonFunction := 1

	err := validateHandler(nonFunction)
	assert.EqualError(t, err, "handler is not a function")
}
func TestValidateHandlerToManyArguments(t *testing.T) {
	tooManyArgs := func(a, b, c int) {
	}

	err := validateHandler(tooManyArgs)
	assert.EqualError(t, err, "handler takes too many arguments")
}

func TestValidateHandlerContextIsNotFirstArgument(t *testing.T) {
	firstArgNotContext := func(arg1, arg2 int) {
	}

	err := validateHandler(firstArgNotContext)
	assert.EqualError(t, err, "handler should take context as first argument")
}

func TestValidateHandlerTwoArguments(t *testing.T) {
	twoArguments := func(arg1 context.Context, arg2 int) {
	}

	err := validateHandler(twoArguments)
	assert.NoError(t, err)
}

func TestValidateHandlerOneArgument(t *testing.T) {
	oneArgument := func(arg1 int) {
	}

	err := validateHandler(oneArgument)
	assert.NoError(t, err)
}

func TestValidateHandlerTooManyReturnValues(t *testing.T) {
	tooManyReturns := func() (int, int, error) {
		return 0, 0, nil
	}

	err := validateHandler(tooManyReturns)
	assert.EqualError(t, err, "handler returns more than two values")
}
func TestValidateHandlerLastReturnValueNotError(t *testing.T) {
	lastNotError := func() (int, int) {
		return 0, 0
	}

	err := validateHandler(lastNotError)
	assert.EqualError(t, err, "handler doesn't return error as it's last value")
}
func TestValidateHandlerCorrectFormat(t *testing.T) {
	correct := func(context context.Context) (int, error) {
		return 0, nil
	}

	err := validateHandler(correct)
	assert.NoError(t, err)
}
