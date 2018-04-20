// Copyright © 2017 Heptio
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

package contour

import (
	"reflect"
	"testing"

	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	"github.com/gogo/protobuf/proto"
)

func TestCacheInsert(t *testing.T) {
	var val, val2 v2.Cluster

	tests := map[string]struct {
		cache
		key   string
		value proto.Message
		want  map[string]proto.Message
	}{
		"empty, add new key": {
			key:   "alpha",
			value: &val,
			want: map[string]proto.Message{
				"alpha": &val,
			},
		},
		"one key, add second": {
			cache: cache{
				entries: map[string]proto.Message{
					"alpha": &val,
				},
			},
			key:   "beta",
			value: &val,
			want: map[string]proto.Message{
				"alpha": &val,
				"beta":  &val,
			},
		},
		"one key overwritten": {
			cache: cache{
				entries: map[string]proto.Message{
					"alpha": &val,
				},
			},
			key:   "alpha",
			value: &val2,
			want: map[string]proto.Message{
				"alpha": &val2,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tc.cache.insert(tc.key, tc.value)
			if !reflect.DeepEqual(tc.cache.entries, tc.want) {
				t.Fatalf("expected: %#v, got %#v", tc.want, tc.cache.entries)
			}
		})
	}
}

func TestCacheRemove(t *testing.T) {
	var val v2.Cluster

	tests := map[string]struct {
		cache
		key  string
		want map[string]proto.Message
	}{
		"one key, remove": {
			cache: cache{
				entries: map[string]proto.Message{
					"alpha": &val,
				},
			},
			key:  "alpha",
			want: map[string]proto.Message{},
		},
		"one key, remove unrelated": {
			cache: cache{
				entries: map[string]proto.Message{
					"alpha": &val,
				},
			},
			key: "beta",
			want: map[string]proto.Message{
				"alpha": &val,
			},
		},
		"empty, remove anything": {
			key:  "alpha",
			want: nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tc.cache.remove(tc.key)
			if !reflect.DeepEqual(tc.cache.entries, tc.want) {
				t.Fatalf("expected: %#v, got %#v", tc.want, tc.cache.entries)
			}
		})
	}
}

func TestVirtualHostCacheValuesReturnsACopyOfItsInternalSlice(t *testing.T) {
	var cc virtualHostCache
	c := &route.VirtualHost{
		Name:    "alpha",
		Domains: []string{"alpha"},
	}
	cc.Add(c)

	v1 := cc.Values()
	v2 := cc.Values()

	if &v1[0] == &v2[0] {
		// the address of the 0th element of the values slice should not be the same
		// if it is, then we don't have a copy.
		t.Fatalf("VirtualHostCache, consecutive calls to Values return the same backing slice: got: %v, want: %v", v1[0], v2[0])
	}
}

func TestVirtualHostCacheAddInsertsTwoElementsInSortOrder(t *testing.T) {
	var cc virtualHostCache
	c1 := &route.VirtualHost{
		Name:    "beta",
		Domains: []string{"beta"},
	}
	cc.Add(c1)
	c2 := &route.VirtualHost{
		Name:    "alpha",
		Domains: []string{"alpha"},
	}
	cc.Add(c2)
	got := cc.Values()
	want := []route.VirtualHost{{
		Name:    "alpha",
		Domains: []string{"alpha"},
	}, {
		Name:    "beta",
		Domains: []string{"beta"},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("VirtualHostCache.Add/Values returned elements missing or out of order, got: %v, want: %v", got, want)
	}
}

func TestVirtualHostCacheAddOverwritesElementsWithTheSameName(t *testing.T) {
	var cc virtualHostCache
	c1 := &route.VirtualHost{
		Name: "alpha",
		Domains: []string{
			"example.com",
		},
	}
	cc.Add(c1)
	c2 := &route.VirtualHost{
		Name: "alpha",
		Domains: []string{
			"heptio.com",
		},
	}
	cc.Add(c2)
	got := cc.Values()
	want := []route.VirtualHost{
		*c2,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("VirtualHostCache.Add/Values returned a stale element, got: %v, want: %v", got, want)
	}
}

func TestVirtualHostCacheIngnoresInvalidVirtualHosts(t *testing.T) {
	tests := map[string]struct {
		vh *route.VirtualHost
	}{
		"missing name": {
			&route.VirtualHost{
				Name: "",
			},
		},
		"missing domains": {
			&route.VirtualHost{
				Name:    "foo",
				Domains: nil,
			},
		},
		"empty domain": {
			&route.VirtualHost{
				Name:    "foo",
				Domains: []string{"foo.example.com", "", "bar.example.com"},
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var cc virtualHostCache
			cc.Add(tc.vh)
			got := cc.Values()
			want := []route.VirtualHost{}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("VirtualHostCache.Add/Values accepted invalid VirtualHost, got: %v, want: %v", got, want)
			}
		})
	}
}

func TestVirtualHostCacheIngnoreElementsWithBlankDomains(t *testing.T) {
	var cc virtualHostCache
	c1 := &route.VirtualHost{
		Name: "alpha",
	}
	cc.Add(c1)
	got := cc.Values()
	want := []route.VirtualHost{}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("VirtualHostCache.Add/Values accepted invalid VirtualHost, got: %v, want: %v", got, want)
	}
}

func TestVirtualHostCacheAddIsCopyOnWrite(t *testing.T) {
	var cc virtualHostCache
	c1 := &route.VirtualHost{
		Name:    "alpha",
		Domains: []string{"alpha"},
	}
	cc.Add(c1)
	v1 := cc.Values()

	c2 := &route.VirtualHost{
		Name:    "beta",
		Domains: []string{"beta"},
	}
	cc.Add(c2)
	v2 := cc.Values()

	if reflect.DeepEqual(v1, v2) {
		t.Fatalf("VirtualHostCache.Add affected the contents of a previous call to Values")
	}
}

func TestVirtualHostCacheRemove(t *testing.T) {
	var cc virtualHostCache
	c1 := &route.VirtualHost{
		Name:    "alpha",
		Domains: []string{"alpha"},
	}
	cc.Add(c1)
	cc.Remove("alpha")
	got := cc.Values()
	want := []route.VirtualHost{}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("VirtualHostCache.Remove: got: %v, want: %v", got, want)
	}
}
