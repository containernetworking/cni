// Copyright 2026 CNI authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package convert_test

import (
	"errors"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/containernetworking/cni/pkg/types"
	convert "github.com/containernetworking/cni/pkg/types/internal"
)

// stubResult is a minimal types.Result used to observe registry behavior.
// Each spec registers factories and converters under unique fake versions,
// because the registries are package-global and reject duplicates.
type stubResult struct {
	version string
}

func (r *stubResult) Version() string                           { return r.version }
func (r *stubResult) GetAsVersion(string) (types.Result, error) { return r, nil }
func (r *stubResult) Print() error                              { return nil }
func (r *stubResult) PrintTo(io.Writer) error                   { return nil }

var _ = Describe("creator registry", func() {
	It("creates a result via the factory registered for the version", func() {
		var gotBytes []byte
		expected := &stubResult{version: "9.9.1"}
		convert.RegisterCreator([]string{"9.9.1"}, func(b []byte) (types.Result, error) {
			gotBytes = b
			return expected, nil
		})

		result, err := convert.Create("9.9.1", []byte(`{"k":"v"}`))
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(BeIdenticalTo(expected))
		Expect(gotBytes).To(Equal([]byte(`{"k":"v"}`)))
	})

	It("returns an error for an unregistered version", func() {
		_, err := convert.Create("8.8.8", nil)
		Expect(err).To(MatchError(`unsupported CNI result version "8.8.8"`))
	})

	It("propagates factory errors", func() {
		convert.RegisterCreator([]string{"9.9.2"}, func([]byte) (types.Result, error) {
			return nil, errors.New("factory exploded")
		})
		_, err := convert.Create("9.9.2", nil)
		Expect(err).To(MatchError("factory exploded"))
	})

	It("panics when a version is registered twice", func() {
		noop := func([]byte) (types.Result, error) { return nil, nil }
		convert.RegisterCreator([]string{"9.9.3"}, noop)
		Expect(func() {
			convert.RegisterCreator([]string{"9.9.3"}, noop)
		}).To(Panic())
	})
})

var _ = Describe("converter registry", func() {
	It("returns the input unchanged when versions match", func() {
		in := &stubResult{version: "9.9.4"}
		out, err := convert.Convert(in, "9.9.4")
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(BeIdenticalTo(in))
	})

	It("defaults an empty target version to 0.1.0", func() {
		converted := &stubResult{version: "0.1.0"}
		convert.RegisterConverter("9.9.5", []string{"0.1.0"}, func(types.Result, string) (types.Result, error) {
			return converted, nil
		})
		out, err := convert.Convert(&stubResult{version: "9.9.5"}, "")
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(BeIdenticalTo(converted))
	})

	It("converts via the converter registered for the version pair", func() {
		converted := &stubResult{version: "9.9.6"}
		var gotFrom types.Result
		var gotToVersion string
		convert.RegisterConverter("9.9.7", []string{"9.9.6"}, func(from types.Result, toVersion string) (types.Result, error) {
			gotFrom, gotToVersion = from, toVersion
			return converted, nil
		})
		in := &stubResult{version: "9.9.7"}
		out, err := convert.Convert(in, "9.9.6")
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(BeIdenticalTo(converted))
		Expect(gotFrom).To(BeIdenticalTo(in))
		Expect(gotToVersion).To(Equal("9.9.6"))
	})

	It("returns an error when no converter is registered for the version pair", func() {
		_, err := convert.Convert(&stubResult{version: "9.9.8"}, "7.7.7")
		Expect(err).To(MatchError("no converter for CNI result version 9.9.8 to 7.7.7"))
	})

	It("rejects a conversion when only one end of the version pair matches", func() {
		const from, to, other = "9.8.1", "9.8.2", "9.8.3"
		convert.RegisterConverter(from, []string{to}, func(types.Result, string) (types.Result, error) {
			return nil, nil
		})

		_, err := convert.Convert(&stubResult{version: from}, other)
		Expect(err).To(HaveOccurred())

		_, err = convert.Convert(&stubResult{version: other}, to)
		Expect(err).To(HaveOccurred())
	})

	It("panics when a version pair is registered twice", func() {
		noop := func(types.Result, string) (types.Result, error) { return nil, nil }
		convert.RegisterConverter("9.9.9", []string{"7.7.9"}, noop)
		Expect(func() {
			convert.RegisterConverter("9.9.9", []string{"7.7.9"}, noop)
		}).To(Panic())
	})
})
