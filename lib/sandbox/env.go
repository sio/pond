package sandbox

import (
	"fmt"
	"os"
)

// Set environment variable for sandboxed processes.
//
// Sandbox inherits parent process environment by default.
func (s *Sandbox) Setenv(key, value string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.env == nil {
		s.env = os.Environ()
	}
	s.env = append(s.env, fmt.Sprintf("%s=%s", key, value))
}
