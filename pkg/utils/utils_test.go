package utils

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Utils", func() {
	It("must format a short name", func() {
		chain := FormatChainName("test", "1234")
		Expect(len(chain)).To(Equal(maxChainLength))
		Expect(chain).To(Equal("CNI-2bbe0c48b91a7d1b8a6753a8"))
	})

	It("must truncate a long name", func() {
		chain := FormatChainName("testalongnamethatdoesnotmakesense", "1234")
		Expect(len(chain)).To(Equal(maxChainLength))
		Expect(chain).To(Equal("CNI-374f33fe84ab0ed84dcdebe3"))
	})

	It("must be predictable", func() {
		chain1 := FormatChainName("testalongnamethatdoesnotmakesense", "1234")
		chain2 := FormatChainName("testalongnamethatdoesnotmakesense", "1234")
		Expect(len(chain1)).To(Equal(maxChainLength))
		Expect(len(chain2)).To(Equal(maxChainLength))
		Expect(chain1).To(Equal(chain2))
	})

	It("must change when a character changes", func() {
		chain1 := FormatChainName("testalongnamethatdoesnotmakesense", "1234")
		chain2 := FormatChainName("testalongnamethatdoesnotmakesense", "1235")
		Expect(len(chain1)).To(Equal(maxChainLength))
		Expect(len(chain2)).To(Equal(maxChainLength))
		Expect(chain1).To(Equal("CNI-374f33fe84ab0ed84dcdebe3"))
		Expect(chain1).NotTo(Equal(chain2))
	})
})
