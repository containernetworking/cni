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

package consul

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/plugins/ipam/allocator/sequential"
	"github.com/hashicorp/consul/api"
)

type Store struct {
	Consul *api.Client
	Key    string
}

type IP_Settings struct {
	Gw     net.IP        `json:"gw"`
	Net    types.IPNet   `json:"net"`
	Start  net.IP        `json:"start"`
	End    net.IP        `json:"end"`
	Routes []types.Route `json:"routes"`
}

type Lease struct {
	IP        net.IP `json:"ip"`
	MAC       string `json:"mac"`
	Id        string `json:"id"`
	Timestamp int64  `json:"timestamp"`
}

func ConnectStore(Addr string, Port string, DC string) (consul *api.Client, err error) {

	config := api.DefaultConfig()

	config.Address = fmt.Sprintf("%s:%s", Addr, Port)
	config.Datacenter = fmt.Sprintf("%s", DC)
	//config.Scheme = fmt.Sprintf("%s", Scheme)

	consul, err = api.NewClient(config)
	if err != nil {
		panic(err)
	}
	return consul, err
}

func New(n *sequential.IPAMConfig) (*Store, error) {
	// creating new consul connection

	addr := fmt.Sprintf("%s", n.Args.StoreAddr)
	port := fmt.Sprintf("%s", n.Args.StorePort)
	dc := fmt.Sprintf("%s", n.Args.StoreNS)

	consul, err := ConnectStore(addr, port, dc)
	if err != nil {
		panic(err)
	}
	network, err := NetConfigJson(n)
	key, err := InitStore(n.Name, network, consul)
	// write values in Store object
	store := &Store{
		Consul: consul,
		Key:    key,
	}
	return store, nil
}

func InitStore(k string, network []byte, consul *api.Client) (store string, err error) {
	kv := consul.KV()
	list, err := GetKV(k, kv)
	// if store doesn't exist we create
	if len(list) == 0 {
		store, err := PutKV(k, network, kv)
		if err != nil {
			panic(err)
		}
		return store, nil
		// otherwise just use existing store as is
	} else {
		store := k
		return store, nil
	}
}

func NetConfigJson(n *sequential.IPAMConfig) (conf []byte, err error) {
	ip_set := IP_Settings{
		Gw:     n.Gateway,
		Net:    n.Subnet,
		Start:  n.RangeStart,
		End:    n.RangeEnd,
		Routes: n.Routes,
	}
	conf, err = json.Marshal(ip_set)
	return conf, err
}

func (s *Store) Unlock() error {
	ses := s.Consul.Session()
	kv := s.Consul.KV()
	pair, _, err := kv.Get(s.Key, nil)
	kv.Release(pair, nil)

	// TODO currently destroying all sessions
	sessions, _, _ := ses.List(nil)
	for _, session := range sessions {
		ses.Destroy(session.ID, nil)
	}
	if err != nil {
		panic(err)
	}
	return nil
}

func (s *Store) Lock() error {

	Session := s.Consul.Session()
	kv := s.Consul.KV()
	var entry *api.SessionEntry

	// create session
	id, _, err := Session.Create(entry, nil)
	if err != nil {
		panic(err)
	}
	// get pair object from consul
	pair, _, err := kv.Get(s.Key, nil)
	pair.Session = id
	if err != nil {
		panic(err)
	}
	// acquire is false
	acq := false
	attempts := 0
	// will try 10 times to get the lock - 10 seconds
	for acq != true {
		if attempts == 10 {
			panic("Wasn't able to acquire the lock in 10 seconds")
		}
		acq, _, err = kv.Acquire(pair, nil)
		if err != nil {
			panic(err)
		}
		attempts += 1
		time.Sleep(1000 * time.Millisecond)
	}
	return err
}

func PutKV(k string, val []byte, kv *api.KV) (k_store string, err error) {
	// put key-value pair
	d := &api.KVPair{Key: k, Value: val}
	_, err = kv.Put(d, nil)
	if err != nil {
		return k, err
	} else {
		k_store = k
		return k, nil
	}
}

func GetKV(k string, kv *api.KV) (list api.KVPairs, err error) {
	// get key list
	list, _, err = kv.List(k, nil)
	if err != nil {
		panic(err)
	}
	return list, err
}

func LeaseJson(ip net.IP, id string) (conf []byte, err error) {
	// TODO mac
	mac := "00:00:00:00:00:00"
	// create lease object
	ip_set := Lease{
		IP:        ip,
		MAC:       mac,
		Id:        id,
		Timestamp: time.Now().Unix(),
	}
	// marshal and return
	conf, err = json.Marshal(ip_set)
	return conf, err
}

func (s *Store) Reserve(id string, ip net.IP) (bool, error) {
	// get consul KV
	kv := s.Consul.KV()
	// create path
	path := s.Key + "/" + fmt.Sprintf("%s", ip)
	pair, _ := GetKV(path, kv)
	// if key exists return false
	if len(pair) != 0 {
		return false, nil
	}
	// otherwise create a byte object and put
	b, _ := LeaseJson(ip, id)
	PutKV(path, b, kv)
	return true, nil
}

// LastReservedIP returns the last reserved IP if exists
func (s *Store) LastReservedIP() (net.IP, error) {
	kv := s.Consul.KV()
	pairs, _ := GetKV(s.Key, kv)

	var lease Lease
	var latest_ip string
	var prev_TS int64

	prev_TS = 0

	for _, pair := range pairs {
		if err := json.Unmarshal(pair.Value, &lease); err != nil {
			return nil, err
		}
		if lease.Timestamp > prev_TS {
			latest_ip = pair.Key
		}
	}
	return net.ParseIP(latest_ip), nil
}

func (s *Store) Release(ip net.IP) error {
	kv := s.Consul.KV()
	path := s.Key + "/" + fmt.Sprintf("%s", ip)
	_, err := kv.Delete(path, nil)
	return err
}

func (s *Store) ReleaseByID(id string) error {
	kv := s.Consul.KV()
	pairs, _ := GetKV(s.Key, kv)

	var lease Lease

	for _, pair := range pairs {
		if err := json.Unmarshal(pair.Value, &lease); err != nil {
			return err
		}
		if lease.Id == id {
			_, err := kv.Delete(pair.Key, nil)
			return err
		}
	}
	return nil
}

func (s *Store) Close() error {
	// stub we don't need close anything
	return nil
}
