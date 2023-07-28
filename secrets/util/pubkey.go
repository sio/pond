package util

import (
	"strings"

	"golang.org/x/crypto/ssh"
)

// Uniformly represent SSH public keys as text
func KeyText(k ssh.PublicKey) string {
	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(k)))
}
