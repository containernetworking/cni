// Copyright 2014-2016 CNI authors
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

package skel

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("direct plugin", func() {

	Context("when no func provided", func() {
		plugin := NewDirectPlugin()
		Describe("method Add", func() {
			It("returns error", func() {
				res, err := plugin.Add(nil)
				Expect(err).To(HaveOccurred())
				Expect(res).To(BeNil())
			})
		})
		Describe("method Check", func() {
			It("returns error", func() {
				res, err := plugin.Check(nil)
				Expect(err).To(HaveOccurred())
				Expect(res).To(BeNil())
			})
		})
		Describe("method Delete", func() {
			It("returns error", func() {
				err := plugin.Del(nil)
				Expect(err).To(HaveOccurred())
			})
		})
		Describe("method Version", func() {
			It("returns error", func() {
				res, err := plugin.Version()
				Expect(err).To(HaveOccurred())
				Expect(res).To(BeNil())
			})
		})
	})

})
