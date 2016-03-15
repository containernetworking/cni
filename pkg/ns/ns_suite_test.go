package ns_test

import (
	"math/rand"
	"runtime"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"

	"testing"
)

func TestNs(t *testing.T) {
	rand.Seed(config.GinkgoConfig.RandomSeed)
	runtime.LockOSThread()

	RegisterFailHandler(Fail)
	RunSpecs(t, "pkg/ns Suite")
}
