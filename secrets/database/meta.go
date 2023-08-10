package database

import (
	"time"
)

type Metadata struct {
	Created  time.Time
	Modified time.Time
	Expires  time.Time
}

func (m *Metadata) Expired() bool {
	return time.Now().After(m.Expires)
}
