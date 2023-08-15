package shield

type ShieldedValue []byte

func (v *ShieldedValue) Bytes() []byte {
	return []byte(*v)
}

func (v *ShieldedValue) Close() {
	cleanup(*v)
}
