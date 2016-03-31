package utils

import (
	"crypto/sha512"
	"fmt"
)

// 29 - len('CNI') - 2*len('-')
const maxNameLen = 16

// Generates a chain name to be used with iptables.
// Ensures that the generated name is less than
// 29 chars in length
func FormatChainName(name string, id string) string {
	h := sha512.Sum512([]byte(id))
	if len(name) > maxNameLen {
		return fmt.Sprintf("CNI-%s-%x", name[:len(name)-maxNameLen], h[:8])
	}
	return fmt.Sprintf("CNI-%s-%x", name, h[:8])
}
