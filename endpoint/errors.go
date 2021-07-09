package endpoint

import "fmt"

type BadRequestError struct {
	Err error
}

func (e *BadRequestError) Error() string {
	return fmt.Sprintf("bad request: %s", e.Err.Error())
}


func (e *BadRequestError) Unwrap() error { return e.Err }

func (e *BadRequestError) Is(target error) bool {
	_, ok := target.(*BadRequestError)
	return ok
}


type NotFoundError struct {
	Err error
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("not found: %s", e.Err.Error())
}


func (e *NotFoundError) Unwrap() error { return e.Err }

func (e *NotFoundError) Is(target error) bool {
	_, ok := target.(*NotFoundError)
	return ok
}


