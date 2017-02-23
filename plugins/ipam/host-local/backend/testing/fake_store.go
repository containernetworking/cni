// Copyright 2015 CNI authors
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

package testing

import (
	"net"

	"github.com/containernetworking/cni/plugins/ipam/host-local/backend"
)

// FakeStore represents a fake store.
type FakeStore struct {
	ipMap          map[string]string
	lastReservedIP net.IP
}

// FakeStore implements the Store interface
var _ backend.Store = &FakeStore{}

// NewFakeStore returns a new fake store.
func NewFakeStore(ipmap map[string]string, lastIP net.IP) *FakeStore {
	return &FakeStore{ipmap, lastIP}
}

// Lock is a no-op.
func (s *FakeStore) Lock() error {
	return nil
}

// Unlock is a no-op.
func (s *FakeStore) Unlock() error {
	return nil
}

// Close is a no-op.
func (s *FakeStore) Close() error {
	return nil
}

// Reserve reserves an IP in the fake store.
func (s *FakeStore) Reserve(id string, ip net.IP) (bool, error) {
	key := ip.String()
	if _, ok := s.ipMap[key]; !ok {
		s.ipMap[key] = id
		s.lastReservedIP = ip
		return true, nil
	}
	return false, nil
}

// LastReservedIP returns last reserved IP in the fake store.
func (s *FakeStore) LastReservedIP() (net.IP, error) {
	return s.lastReservedIP, nil
}

// Release releases all IPs in the fake store.
func (s *FakeStore) Release(ip net.IP) error {
	delete(s.ipMap, ip.String())
	return nil
}

// ReleaseByID releases an IP identified by id, in the fake store.
func (s *FakeStore) ReleaseByID(id string) error {
	toDelete := []string{}
	for k, v := range s.ipMap {
		if v == id {
			toDelete = append(toDelete, k)
		}
	}
	for _, ip := range toDelete {
		delete(s.ipMap, ip)
	}
	return nil
}
