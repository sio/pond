package crypto

//
// Export private methods for tests.
//
// Decryption method does not need exporting as it must be selected
// automatically by SecretValue.Decrypt() based on the first byte
//

func V1encrypt(sign SignerFunc, value string, keywords ...string) (s SecretValue, e error) {
	e = s.v1encrypt(sign, value, keywords...)
	if e != nil {
		return nil, e
	}
	return s, nil
}
