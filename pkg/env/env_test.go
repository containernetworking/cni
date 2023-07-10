// Copyright 2016 CNI authors
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

package env_test

import (
	"os"
	"strings"

	"github.com/containernetworking/cni/pkg/env"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetValue", func() {
	BeforeEach(func() {
		os.Setenv("FOO", "bar")
		os.Unsetenv("BAC")
	})

	Context("when environment has variable", func() {
		It("returns the variable value", func() {
			val := env.GetValue("FOO", "gaff")
			Expect(val).To(Equal("bar"))
		})
	})

	Context("when environment does not have variable", func() {
		It("returns the fallback value", func() {
			val := env.GetValue("BAC", "gaff")
			Expect(val).To(Equal("gaff"))
		})
	})
})

var _ = Describe("ParseCNIPath", func() {
	BeforeEach(func() {
		os.Unsetenv("CNI_PATH")
	})

	AfterEach(func() {
		os.Unsetenv("CNI_PATH")
	})

	Context("when no directories are specified", func() {
		It("returns an empty list", func() {
			val := env.ParseCNIPath()
			Expect(val).To(BeEmpty())
		})
	})

	Context("when multiple directories are specified", func() {
		It("returns the directories as a list", func() {
			mockCNIPath("/test/bin", "/test/libexec")

			val := env.ParseCNIPath()
			Expect(val).To(Equal([]string{"/test/bin", "/test/libexec"}))
		})
	})
})

var _ = Describe("ParseCNIArgs", func() {
	BeforeEach(func() {
		os.Unsetenv("CNI_ARGS")
	})

	AfterEach(func() {
		os.Unsetenv("CNI_ARGS")
	})

	Context("when no arguments are specified", func() {
		It("returns an empty list", func() {
			val, err := env.ParseCNIArgs()
			Expect(err).To(BeNil())
			Expect(val).To(BeEmpty())
		})
	})

	Context("when a single argument is specified", func() {
		It("returns the argument tuple as a list", func() {
			mockCNIArgs("KEY=value")

			val, err := env.ParseCNIArgs()
			Expect(err).To(BeNil())
			Expect(val).To(HaveLen(1))
			Expect(val).To(ContainElement([2]string{"KEY", "value"}))
		})
	})

	Context("when a multiple arguments are specified", func() {
		It("returns the argument tuples as a list", func() {
			mockCNIArgs("KEY=value", "GAFF=bac")

			val, err := env.ParseCNIArgs()
			Expect(err).To(BeNil())
			Expect(val).To(HaveLen(2))
			Expect(val).To(ContainElement([2]string{"KEY", "value"}))
			Expect(val).To(ContainElement([2]string{"GAFF", "bac"}))
		})
	})

	Context("when a the argument is malformed", func() {
		It("returns an error", func() {
			mockCNIArgs("KEY=value=error")

			_, err := env.ParseCNIArgs()
			Expect(err).To(MatchError("invalid CNI_ARGS pair \"KEY=value=error\""))
		})
	})

	Context("when a the argument has no value", func() {
		It("returns an error", func() {
			mockCNIArgs("KEY")

			_, err := env.ParseCNIArgs()
			Expect(err).To(MatchError("invalid CNI_ARGS pair \"KEY\""))
		})
	})

	Context("when a the argument has no key", func() {
		It("returns an error", func() {
			mockCNIArgs("=value")

			_, err := env.ParseCNIArgs()
			Expect(err).To(MatchError("invalid CNI_ARGS pair \"=value\""))
		})
	})
})

func mockCNIPath(dir ...string) {
	mock := strings.Join(dir, string(os.PathListSeparator))

	os.Setenv(env.VarCNIPath, mock)
}

func mockCNIArgs(arg ...string) {
	mock := strings.Join(arg, ";")

	os.Setenv(env.VarCNIArgs, mock)
}
