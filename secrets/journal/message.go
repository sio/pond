package journal

import (
	"encoding/binary"
	"fmt"
	"time"

	"secrets/pack"
)

type Message struct {
	Timestamp time.Time
	Action    Verb
	Elements  []string
}

type Verb byte // TODO: new Verbs for "stop logging" and "truncate file"

const (
	Add Verb = '+'
	Del Verb = '-'
)

// Serialize message to binary representation
func (m *Message) Encode(epoch time.Time) ([]byte, error) {
	elements, err := pack.Encode(m.Elements)
	if err != nil {
		return nil, fmt.Errorf("encoding message elements: %w", err)
	}
	offset := m.Timestamp.Sub(epoch).Seconds()
	if offset < 0 {
		return nil, fmt.Errorf("message timestamp [%v] before the start of the epoch [%v]", m.Timestamp, epoch)
	}
	var encoded []byte
	encoded = binary.BigEndian.AppendUint32(nil, uint32(offset))
	encoded = append(encoded, byte(m.Action))
	encoded = append(encoded, elements...)
	return encoded, nil
}

// Deserialize message from binary representation
func (m *Message) Decode(b []byte, epoch time.Time) error {
	if len(b) < 4+1+1 {
		return fmt.Errorf("message too short to start decoding: %d bytes", len(b))
	}
	timeOffset := int64(binary.BigEndian.Uint32(b[:4]))
	action := Verb(b[4])
	switch action {
	case Add, Del:
	default:
		return fmt.Errorf("invalid message action: %x (%c)", b[4], b[4])
	}
	elements, err := pack.Decode(b[5:])
	if err != nil {
		return fmt.Errorf("decoding message elements: %w", err)
	}
	m.Timestamp = time.Unix(timeOffset+epoch.Unix(), 0)
	m.Action = action
	m.Elements = elements
	return nil
}
