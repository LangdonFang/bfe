// Copyright (c) 2019 Baidu, Inc.
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

package bal_slb

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"testing"
)

import (
	"github.com/baidu/bfe/bfe_balance/backend"
	"github.com/baidu/bfe/bfe_config/bfe_cluster_conf/cluster_table_conf"
)

func populateBackend(name, addr string, port int, avail bool) *backend.BfeBackend {
	b := backend.NewBfeBackend()
	b.Name = name
	b.Addr = addr
	b.Port = port
	b.AddrInfo = fmt.Sprintf("%s:%d", addr, port)
	b.SetAvail(avail)
	return b
}

func prepareBalanceRR() *BalanceRR {
	b1 := populateBackend("b1", "127.0.0.1", 80, true)
	b2 := populateBackend("b2", "127.0.0.1", 81, true)
	b3 := populateBackend("b3", "127.0.0.1", 82, true)

	rr := &BalanceRR{
		backends: []*BackendRR{
			&BackendRR{
				weight:  3,
				current: 3,
				backend: b1,
			},
			&BackendRR{
				weight:  2,
				current: 2,
				backend: b2,
			},
			&BackendRR{
				weight:  1,
				current: 1,
				backend: b3,
			},
		},
	}
	return rr
}

func processBalance(t *testing.T, label string, algor int, key []byte, rr *BalanceRR, result []string) {
	var l []string
	for i := 1; i < 10; i++ {
		r, err := rr.Balance(algor, key)
		if err != nil {
			t.Errorf("should not error")
		}
		r.AddConnNum()
		l = append(l, r.Name)
	}

	if !reflect.DeepEqual(l, result) {
		t.Errorf("balance error [%s] %v, expect %v", label, l, result)
	}
}

func TestBalance(t *testing.T) {
	// case 1
	rr := prepareBalanceRR()
	expectResult := []string{"b1", "b2", "b3", "b1", "b2", "b1", "b1", "b2", "b3"}
	processBalance(t, "case 1", WrrSimple, nil, rr, expectResult)

	// case 2
	rr = prepareBalanceRR()
	expectResult = []string{"b1", "b2", "b1", "b3", "b2", "b1", "b1", "b2", "b1"}
	processBalance(t, "case 2", WrrSmooth, nil, rr, expectResult)

	// case 3
	rr = prepareBalanceRR()
	rr.backends[0].backend.SetAvail(false)
	expectResult = []string{"b2", "b3", "b2", "b2", "b3", "b2", "b2", "b3", "b2"}
	processBalance(t, "case 3", WrrSimple, nil, rr, expectResult)

	// case 4
	rr = prepareBalanceRR()
	rr.backends[0].backend.SetAvail(false)
	expectResult = []string{"b2", "b3", "b2", "b2", "b3", "b2", "b2", "b3", "b2"}
	processBalance(t, "case 4", WrrSmooth, nil, rr, expectResult)

	// case 5
	rr = prepareBalanceRR()
	expectResult = []string{"b2", "b2", "b2", "b2", "b2", "b2", "b2", "b2", "b2"}
	processBalance(t, "case 5", WrrSticky, []byte{1}, rr, expectResult)

	rr.backends[0], rr.backends[2] = rr.backends[2], rr.backends[0]
	rr.sorted = false
	processBalance(t, "case 5", WrrSticky, []byte{1}, rr, expectResult)

	// case 6
	rr = prepareBalanceRR()
	rr.backends[0].backend.SetAvail(false)
	expectResult = []string{"b2", "b2", "b2", "b2", "b2", "b2", "b2", "b2", "b2"}
	processBalance(t, "case 6", WrrSticky, []byte{1}, rr, expectResult)

	// case 7, lcw balance
	rr = prepareBalanceRR()
	expectResult = []string{"b1", "b2", "b3", "b1", "b2", "b1", "b3", "b1", "b2"}
	processBalance(t, "case 7", WlcSmooth, []byte{1}, rr, expectResult)
}

func TestUpdate(t *testing.T) {
	b1 := populateBackend("b1", "127.0.0.1", 80, true)
	b2 := populateBackend("b2", "127.0.0.1", 81, true)
	b3 := populateBackend("b3", "127.0.0.1", 82, true)
	rr := &BalanceRR{
		backends: []*BackendRR{
			&BackendRR{
				weight:  3,
				current: 3,
				backend: b1,
			},
			&BackendRR{
				weight:  2,
				current: 2,
				backend: b2,
			},
			&BackendRR{
				weight:  1,
				current: 1,
				backend: b3,
			},
		},
	}

	var newConf cluster_table_conf.SubClusterBackend
	buf := []byte(`[{"name":"b1", "Addr":"12", "Port":10, "weight":10}, {"name":"b2", "Addr":"127.0.0.1", "Port":81, "weight":20}, {"name":"b4", "Addr":"13", "Port":90, "weight":10}]`)
	if err := json.Unmarshal(buf, &newConf); err != nil {
		t.Errorf("unmarshal error")
	}

	rr.Update(newConf)
	if len(rr.backends) != 3 {
		t.Errorf("backend len %d", len(rr.backends))
	}

	b, _ := rr.Balance(WlcSmooth, []byte{1})
	b.AddConnNum()
	b, _ = rr.Balance(WlcSmooth, []byte{1})
	b.AddConnNum()

	for i := 0; i < len(rr.backends); i++ {
		brr := rr.backends[i]
		switch brr.backend.Name {
		case "b1":
			checkBackend(t, rr.backends[i], "b1", "12", 10, 10, -1)
		case "b2":
			checkBackend(t, rr.backends[i], "b2", "127.0.0.1", 81, 20, 1)
		case "b4":
			checkBackend(t, rr.backends[i], "b4", "13", 90, 10, -1)
		default:
			t.Errorf("should not contail backend %v", brr)
		}
	}
}

func checkBackend(t *testing.T, brr *BackendRR, name string, addr string, port int, weight int, connNum int) {
	b := brr.backend
	if b.Name != name {
		t.Errorf("backend name wrong, expect %s, actual %s", name, b.Name)
	}
	if b.Addr != addr {
		t.Errorf("backend addr wrong, expect %s, actual %s", addr, b.Addr)
	}
	if b.Port != port {
		t.Errorf("backend port wrong, expect %d, actual %d", port, b.Port)
	}
	if brr.weight != weight {
		t.Errorf("backend weight wrong, expect %d, actual %d", weight, brr.weight)
	}
	if connNum != -1 && b.ConnNum() != connNum {
		t.Errorf("backend connNum wrong, expect %d, actual %d", connNum, b.ConnNum())
	}
}

func prepareBalanceRRForBench() *BalanceRR {
	rr := new(BalanceRR)
	rr.backends = make([]*BackendRR, 0)
	for i := 0; i < 100; i++ {
		addr := fmt.Sprintf("10.10.0.%d", i)
		backendRR := new(BackendRR)
		backendRR.weight = 1 + rand.Intn(5)
		backendRR.current = backendRR.weight
		backendRR.backend = populateBackend(addr, addr, 80, true)
		rr.backends = append(rr.backends, backendRR)
	}
	return rr
}

func BenchmarkSmoothBalance(b *testing.B) {
	rr := prepareBalanceRRForBench()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr.smoothBalance()
	}
}

func BenchmarkSimpleBalance(b *testing.B) {
	rr := prepareBalanceRRForBench()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr.simpleBalance()
	}
}

func BenchmarkStickyBalance(b *testing.B) {
	rr := prepareBalanceRRForBench()
	key := []byte{100}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr.stickyBalance(key)
	}
}
