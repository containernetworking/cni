package invoke_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestInvoke(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Invoke Suite")
}
