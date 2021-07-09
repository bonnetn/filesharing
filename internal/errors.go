package internal

import "fmt"

// BadRequestError represents an user input error.
type BadRequestError struct {
	Err error
}

// Error implements the error interface.
func (e *BadRequestError) Error() string {
	return fmt.Sprintf("bad request: %s", e.Err.Error())
}


func (e *BadRequestError) Unwrap() error { return e.Err }

func (e *BadRequestError) Is(target error) bool {
	_, ok := target.(*BadRequestError)
	return ok
}


// NotFoundError represents a resource not found error.
type NotFoundError struct {
	Err error
}

// Error implements the error interface.
func (e *NotFoundError) Error() string {
	return fmt.Sprintf("not found: %s", e.Err.Error())
}


func (e *NotFoundError) Unwrap() error { return e.Err }

func (e *NotFoundError) Is(target error) bool {
	_, ok := target.(*NotFoundError)
	return ok
}


