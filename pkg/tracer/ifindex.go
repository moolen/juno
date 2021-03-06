// Copyright 2018 Authors of Cilium
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

// see: https://github.com/cilium/cilium/blob/7cc8a43ec837f06c6cdda484b41e647520ea4140/pkg/monitor/ifindex.go

package tracer

import (
	"fmt"
	"sync"
	"time"

	"github.com/vishvananda/netlink"
)

type linkMap map[int]string

var (
	ifindexMap = linkMap{}
	mutex      sync.RWMutex
)

func ifname(ifindex int) string {
	mutex.RLock()
	defer mutex.RUnlock()

	if name, ok := ifindexMap[ifindex]; ok {
		return name
	}

	return fmt.Sprintf("%d", ifindex)
}

func init() {
	go func() {
		for {
			newMap := linkMap{}

			links, err := netlink.LinkList()
			if err != nil {
				goto sleep
			}

			for _, link := range links {
				newMap[link.Attrs().Index] = link.Attrs().Name
			}

			mutex.Lock()
			ifindexMap = newMap
			mutex.Unlock()

		sleep:
			time.Sleep(time.Duration(15) * time.Second)
		}

	}()
}
