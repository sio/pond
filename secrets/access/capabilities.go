package access

type Capability string

const (
	ManageWriters Capability = "pond/secrets: manage write access"
	ManageReaders            = "pond/secrets: manage read access"
	Read                     = "pond/secrets: read secrets"
	Write                    = "pond/secrets: write secrets"
)

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
