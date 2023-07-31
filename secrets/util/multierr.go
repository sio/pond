package util

import (
	"errors"
	"fmt"
	"io"
)

type MultiError []error

func (m *MultiError) Error(e error) {
	if e == nil {
		return
	}
	*m = append(*m, e)
}

func (m *MultiError) Sum() error {
	return errors.Join([]error(*m)...)
}

func (m *MultiError) Errorf(format string, args ...any) {
	var countNotNil uint
	for _, arg := range args {
		if err, ok := arg.(error); ok && errors.Is(err, io.EOF) {
			continue
		}
		if arg != nil {
			countNotNil++
		}
	}
	if len(args) > 0 && countNotNil == 0 {
		return
	}
	m.Error(fmt.Errorf(format, args...))
}
