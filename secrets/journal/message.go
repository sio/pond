package journal

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"
)

type Message struct { // TODO: add sequential message id!
	Timestamp time.Time
	Action    Verb
	Items     []Item
}

type Verb byte // TODO: new Verbs for "stop logging" and "truncate file"

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

// Serialize message to binary representation
func (m *Message) Encode(epoch time.Time) ([]byte, error) {
	items, err := json.Marshal(m.Items)
	if err != nil {
		return nil, fmt.Errorf("json: %w", err)
	}
	offset := m.Timestamp.Sub(epoch).Seconds()
	if offset < 0 {
		return nil, fmt.Errorf("message timestamp [%v] before the start of the epoch [%v]", m.Timestamp, epoch)
	}
	var encoded []byte
	encoded = binary.BigEndian.AppendUint32(nil, uint32(offset))
	encoded = append(encoded, byte(m.Action))
	encoded = append(encoded, items...)
	return encoded, nil
}

// Deserialize message from binary representation
func (m *Message) Decode(b []byte, epoch time.Time) error {
	return nil
}
