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

package invoke_test

import (
	"github.com/containernetworking/cni/pkg/invoke"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Parsing Delegate Network Configuration", func() {
	It("extracts an inner plugin configuration", func() {
		conf := `{
  "cniVersion": "0.2.0",
  "name": "chaining-example",
  "type": "traffic-shaping",
  "ingressbw": "20M",
  "egressbw": "10M",
  "inner": {
    "type": "bridge",
    "bridge": "cni0",
    "ipam": {
      "type": "host-local",
      "subnet": "10.1.0.0/16",
      "gateway": "10.1.0.1"
    }
  },
  "dns": {
    "nameservers": [ "10.1.0.1" ]
  }
}`
		delegatePlugin, delegateConf, err := invoke.GetNextPlugin([]byte(conf))
		Expect(err).NotTo(HaveOccurred())
		Expect(delegatePlugin).To(Equal("bridge"))
		Expect(delegateConf).To(Equal([]byte(`{
    "bridge": "cni0",
    "cniVersion": "0.2.0",
    "ipam": {
        "gateway": "10.1.0.1",
        "subnet": "10.1.0.0/16",
        "type": "host-local"
    },
    "name": "chaining-example",
    "type": "bridge"
}`)))
	})

	It("extracts an IPAM plugin configuration", func() {
		conf := `{
  "cniVersion": "0.2.0",
  "name": "chaining-example",
  "type": "traffic-shaping",
  "ingressbw": "20M",
  "egressbw": "10M",
  "ipam": {
    "type": "host-local",
    "subnet": "10.1.0.0/16",
    "gateway": "10.1.0.1"
  },
  "dns": {
    "nameservers": [ "10.1.0.1" ]
  }
}`
		delegatePlugin, delegateConf, err := invoke.GetNextPlugin([]byte(conf))
		Expect(err).NotTo(HaveOccurred())
		Expect(delegatePlugin).To(Equal("host-local"))
		Expect(delegateConf).To(Equal([]byte(conf)))
	})

	It("extracts no delegate plugin configuration when none present", func() {
		conf := `{
  "cniVersion": "0.2.0",
  "name": "chaining-example",
  "type": "traffic-shaping",
  "ingressbw": "20M",
  "egressbw": "10M"
}`
		delegatePlugin, delegateConf, err := invoke.GetNextPlugin([]byte(conf))
		Expect(err).NotTo(HaveOccurred())
		Expect(delegatePlugin).To(Equal(""))
		Expect(len(delegateConf)).To(Equal(0))
	})
})
