package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

type multiError []error

func (m *multiError) Error(e error) {
	if e == nil {
		return
	}
	*m = append(*m, e)
}

func (m *multiError) Sum() error {
	return errors.Join([]error(*m)...)
}

func (m *multiError) Errorf(format string, args ...any) {
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

func (m multiError) String() string {
	messages := make([]string, len(m))
	for i, err := range m {
		messages[i] = err.Error()
	}
	return strings.Join(messages, "; ")
}

func (m *multiError) MarshalJSON() ([]byte, error) {
	messages := make([]string, len(*m))
	for i, err := range *m {
		messages[i] = err.Error()
	}
	return json.Marshal(messages)
}
