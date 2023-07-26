package util

import (
	"errors"
	"fmt"
	"io"
)

type MultiError []error

func (m *MultiError) Add(e error) {
	if e == nil {
		return
	}
	*m = append(*m, e)
}

func (m *MultiError) Err() error {
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
	m.Add(fmt.Errorf(format, args...))
}
