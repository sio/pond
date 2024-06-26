package access

type Capability string

const (
	ManageWriters Capability = "pond/secrets: manage write access"
	ManageReaders Capability = "pond/secrets: manage read access"
	Read          Capability = "pond/secrets: read secrets"
	Write         Capability = "pond/secrets: write secrets"
)

var short = map[Capability]string{
	ManageWriters: "wA",
	ManageReaders: "rA",
	Read:          "r",
	Write:         "w",
}

var Required = map[Capability]Capability{
	Read:  ManageReaders,
	Write: ManageWriters,
}

var caps = map[Capability]uint8{
	ManageReaders: 1,
	ManageWriters: 2,
	Read:          3,
	Write:         4,
}

func (c Capability) Admin() bool {
	switch c {
	case ManageWriters, ManageReaders:
		return true
	}
	return false
}

func (c Capability) User() bool {
	switch c {
	case Read, Write:
		return true
	}
	return false
}

func (c Capability) Valid() bool {
	switch c {
	case ManageWriters, ManageReaders, Read, Write:
		return true
	}
	return false
}

func (c Capability) Short() string {
	s, ok := short[c]
	if !ok {
		s = string(c)
	}
	return s
}
