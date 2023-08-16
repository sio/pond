package shield

// Unprotected sensitive data.
// Needs to be handled with care and never copied.
//
// Call Close() to clean up as soon as done processing.
type UnshieldedValue []byte

// Represent unshielded value as a slice of bytes without copying data
func (v *UnshieldedValue) Bytes() []byte {
	return []byte(*v)
}

// Clear sensitive data from memory
func (v *UnshieldedValue) Close() {
	cleanup(*v)
}
