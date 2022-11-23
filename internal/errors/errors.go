package errors

import (
	"fmt"
	"runtime"
)

type Error struct {
	Cause    error
	Location string
}

func Wrap(err error, skip int) error {
	if err == nil {
		return nil
	}

	c := &Error{
		Cause:    err,
		Location: getLocation(skip),
	}

	return c
}

func (w *Error) Error() string {
	return w.Cause.Error()
}

func (f *Error) Unwrap() error {
	return f.Cause
}

func (f *Error) Format(s fmt.State, verb rune) {
	fmt.Fprintf(s, "%s\n", f.Cause.Error())
	fmt.Fprintf(s, "\t%s\n", f.Location)
}

func getLocation(skip int) string {
	_, file, line, _ := runtime.Caller(2 + skip)
	return fmt.Sprintf("%s:%d", file, line)
}
