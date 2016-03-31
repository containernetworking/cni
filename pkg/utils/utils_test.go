package utils

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Utils", func() {
	It("should format a short name", func() {
		chain := FormatChainName("test", "1234")
		Expect(chain).To(Equal("CNI-test-d404559f602eab6f"))
	})

	It("should truncate a long name", func() {
		chain := FormatChainName("testalongnamethatdoesnotmakesense", "1234")
		Expect(chain).To(Equal("CNI-testalongnamethat-d404559f602eab6f"))
	})
})
