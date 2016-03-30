package utils

import (
	"crypto/sha512"
	"fmt"
)

const ChainCNIPrefix = ("CNI-")
const MaxChainLength = 29 - len(ChainCNIPrefix)

// Generates a chain name to be used with iptables.
// Ensures that the generated name is less than
// 29 chars in length
func FormatChainName(name string, id string) string {
	chain := fmt.Sprintf("%x", sha512.Sum512([]byte(name+id)))
	if len(chain) > MaxChainLength {
		chain = chain[:MaxChainLength]
	}
	return fmt.Sprintf("%s%s", ChainCNIPrefix, chain)
}
