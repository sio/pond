package audit

import (
	"bytes"
	"encoding/json"
	"time"
)

type Message struct {
	Timestamp time.Time
	Action    Verb
	Items     []Item
}

type Verb byte

const (
	Add Verb = '+'
	Del Verb = '-'
)

type Item struct {
	Name  string
	Value string
}

func (i *Item) MarshalJSON() ([]byte, error) {
	var name, value []byte
	var err error
	name, err = json.Marshal(i.Name)
	if err != nil {
		return nil, err
	}
	value, err = json.Marshal(i.Value)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	b.WriteRune('{')
	b.Write(name)
	b.WriteRune(':')
	b.Write(value)
	b.WriteRune('}')
	return b.Bytes(), nil
}
