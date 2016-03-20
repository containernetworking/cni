package skel

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestSkel(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Skel Suite")
}
