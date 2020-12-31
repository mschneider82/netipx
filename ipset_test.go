// Copyright 2020 The Inet.Af AUTHORS. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package netaddr

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"reflect"
	"testing"
)

func TestIPSet(t *testing.T) {
	tests := []struct {
		name         string
		f            func(s *IPSet)
		wantRanges   []IPRange
		wantPrefixes []IPPrefix      // non-nil to test
		wantContains map[string]bool // optional non-exhaustive IPs to test for in resulting set
	}{
		{
			name: "mix_family",
			f: func(s *IPSet) {
				s.AddPrefix(mustIPPrefix("10.0.0.0/8"))
				s.AddPrefix(mustIPPrefix("::/0"))
				s.RemovePrefix(mustIPPrefix("10.2.0.0/16"))
			},
			wantRanges: []IPRange{
				{mustIP("10.0.0.0"), mustIP("10.1.255.255")},
				{mustIP("10.3.0.0"), mustIP("10.255.255.255")},
				{mustIP("::"), mustIP("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff")},
			},
		},
		{
			name: "merge_adjacent",
			f: func(s *IPSet) {
				s.AddPrefix(mustIPPrefix("10.0.0.0/8"))
				s.AddPrefix(mustIPPrefix("11.0.0.0/8"))
			},
			wantRanges: []IPRange{
				{mustIP("10.0.0.0"), mustIP("11.255.255.255")},
			},
			wantPrefixes: pxv("10.0.0.0/7"),
		},
		{
			name: "remove_32",
			f: func(s *IPSet) {
				s.AddPrefix(mustIPPrefix("10.0.0.0/8"))
				s.RemovePrefix(mustIPPrefix("10.1.2.3/32"))
			},
			wantRanges: []IPRange{
				{mustIP("10.0.0.0"), mustIP("10.1.2.2")},
				{mustIP("10.1.2.4"), mustIP("10.255.255.255")},
			},
			wantPrefixes: pxv(
				"10.0.0.0/16",
				"10.1.0.0/23",
				"10.1.2.0/31",
				"10.1.2.2/32",
				"10.1.2.4/30",
				"10.1.2.8/29",
				"10.1.2.16/28",
				"10.1.2.32/27",
				"10.1.2.64/26",
				"10.1.2.128/25",
				"10.1.3.0/24",
				"10.1.4.0/22",
				"10.1.8.0/21",
				"10.1.16.0/20",
				"10.1.32.0/19",
				"10.1.64.0/18",
				"10.1.128.0/17",
				"10.2.0.0/15",
				"10.4.0.0/14",
				"10.8.0.0/13",
				"10.16.0.0/12",
				"10.32.0.0/11",
				"10.64.0.0/10",
				"10.128.0.0/9",
			),
		},
		{
			name: "remove_32_and_first_16",
			f: func(s *IPSet) {
				s.AddPrefix(mustIPPrefix("10.0.0.0/8"))
				s.RemovePrefix(mustIPPrefix("10.1.2.3/32"))
				s.RemovePrefix(mustIPPrefix("10.0.0.0/16"))
			},
			wantRanges: []IPRange{
				{mustIP("10.1.0.0"), mustIP("10.1.2.2")},
				{mustIP("10.1.2.4"), mustIP("10.255.255.255")},
			},
			wantPrefixes: pxv(
				"10.1.0.0/23",
				"10.1.2.0/31",
				"10.1.2.2/32",
				"10.1.2.4/30",
				"10.1.2.8/29",
				"10.1.2.16/28",
				"10.1.2.32/27",
				"10.1.2.64/26",
				"10.1.2.128/25",
				"10.1.3.0/24",
				"10.1.4.0/22",
				"10.1.8.0/21",
				"10.1.16.0/20",
				"10.1.32.0/19",
				"10.1.64.0/18",
				"10.1.128.0/17",
				"10.2.0.0/15",
				"10.4.0.0/14",
				"10.8.0.0/13",
				"10.16.0.0/12",
				"10.32.0.0/11",
				"10.64.0.0/10",
				"10.128.0.0/9",
			),
		},
		{
			name: "add_dup",
			f: func(s *IPSet) {
				s.AddPrefix(mustIPPrefix("10.0.0.0/8"))
				s.AddPrefix(mustIPPrefix("10.0.0.0/8"))
			},
			wantRanges: []IPRange{
				{mustIP("10.0.0.0"), mustIP("10.255.255.255")},
			},
		},
		{
			name: "add_dup_subet",
			f: func(s *IPSet) {
				s.AddPrefix(mustIPPrefix("10.0.0.0/8"))
				s.AddPrefix(mustIPPrefix("10.0.0.0/16"))
			},
			wantRanges: []IPRange{
				{mustIP("10.0.0.0"), mustIP("10.255.255.255")},
			},
		},
		{
			name: "add_remove_add",
			f: func(s *IPSet) {
				s.AddPrefix(mustIPPrefix("10.0.0.0/8"))
				s.RemovePrefix(mustIPPrefix("10.1.2.3/32"))
				s.AddPrefix(mustIPPrefix("10.1.0.0/16")) // undoes prior line
			},
			wantRanges: []IPRange{
				{mustIP("10.0.0.0"), mustIP("10.255.255.255")},
			},
		},
		{
			name: "remove_then_add",
			f: func(s *IPSet) {
				s.RemovePrefix(mustIPPrefix("1.2.3.4/32")) // no-op
				s.AddPrefix(mustIPPrefix("1.2.3.4/32"))
			},
			wantRanges: []IPRange{
				{mustIP("1.2.3.4"), mustIP("1.2.3.4")},
			},
		},
		{
			name: "remove_end_on_add_start",
			f: func(s *IPSet) {
				s.AddRange(IPRange{mustIP("0.0.0.38"), mustIP("0.0.0.177")})
				s.RemoveRange(IPRange{mustIP("0.0.0.18"), mustIP("0.0.0.38")})
			},
			wantRanges: []IPRange{
				{mustIP("0.0.0.39"), mustIP("0.0.0.177")},
			},
		},
		{
			name: "fuzz_fail_2",
			f: func(s *IPSet) {
				s.AddRange(IPRange{mustIP("0.0.0.143"), mustIP("0.0.0.185")})
				s.AddRange(IPRange{mustIP("0.0.0.84"), mustIP("0.0.0.174")})
				s.AddRange(IPRange{mustIP("0.0.0.51"), mustIP("0.0.0.61")})
				s.RemoveRange(IPRange{mustIP("0.0.0.66"), mustIP("0.0.0.146")})
				s.AddRange(IPRange{mustIP("0.0.0.22"), mustIP("0.0.0.207")})
				s.RemoveRange(IPRange{mustIP("0.0.0.198"), mustIP("0.0.0.203")})
				s.RemoveRange(IPRange{mustIP("0.0.0.23"), mustIP("0.0.0.69")})
				s.AddRange(IPRange{mustIP("0.0.0.64"), mustIP("0.0.0.105")})
				s.AddRange(IPRange{mustIP("0.0.0.151"), mustIP("0.0.0.203")})
				s.AddRange(IPRange{mustIP("0.0.0.138"), mustIP("0.0.0.160")})
				s.RemoveRange(IPRange{mustIP("0.0.0.64"), mustIP("0.0.0.161")})
			},
			wantRanges: []IPRange{
				{mustIP("0.0.0.22"), mustIP("0.0.0.22")},
				{mustIP("0.0.0.162"), mustIP("0.0.0.207")},
			},
			wantContains: map[string]bool{
				"0.0.0.22": true,
			},
		},
		{
			name: "single_ips",
			f: func(s *IPSet) {
				s.Add(mustIP("10.0.0.0"))
				s.Add(mustIP("10.0.0.1"))
				s.Add(mustIP("10.0.0.2"))
				s.Add(mustIP("10.0.0.3"))
				s.Add(mustIP("10.0.0.4"))
				s.Remove(mustIP("10.0.0.4"))
				s.Add(mustIP("10.0.0.255"))
			},
			wantRanges: []IPRange{
				{mustIP("10.0.0.0"), mustIP("10.0.0.3")},
				{mustIP("10.0.0.255"), mustIP("10.0.0.255")},
			},
			wantPrefixes: pxv("10.0.0.0/30", "10.0.0.255/32"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			debugf = t.Logf
			defer func() { debugf = discardf }()
			var s IPSet
			tt.f(&s)
			got := s.Ranges()
			t.Run("ranges", func(t *testing.T) {
				if reflect.DeepEqual(got, tt.wantRanges) {
					return
				}
				t.Error("failed. got:\n")
				for _, v := range got {
					t.Errorf("  %s -> %s", v.From, v.To)
				}
				t.Error("want:\n")
				for _, v := range tt.wantRanges {
					t.Errorf("  %s -> %s", v.From, v.To)
				}
			})
			if tt.wantPrefixes != nil {
				t.Run("prefixes", func(t *testing.T) {
					got := s.Prefixes()
					if got == nil {
						got = []IPPrefix{}
					}
					if reflect.DeepEqual(got, tt.wantPrefixes) {
						return
					}
					t.Error("failed. got:\n")
					for _, v := range got {
						t.Errorf("  %v", v)
					}
					t.Error("want:\n")
					for _, v := range tt.wantPrefixes {
						t.Errorf("  %v", v)
					}
				})
			}
			if len(tt.wantContains) > 0 {
				contains := s.ContainsFunc()
				for s, want := range tt.wantContains {
					got := contains(mustIP(s))
					if got != want {
						t.Errorf("Contains(%q) = %v; want %v", s, got, want)
					}
				}
			}
		})
	}
}

func TestIPSetRemoveFreePrefix(t *testing.T) {
	pfx := mustIPPrefix
	tests := []struct {
		name         string
		f            func(s *IPSet)
		b            uint8
		wantPrefix   IPPrefix
		wantPrefixes []IPPrefix
		wantOK       bool
	}{
		{
			name: "cut in half",
			f: func(s *IPSet) {
				s.AddPrefix(pfx("10.0.0.0/8"))
			},
			b:            9,
			wantPrefix:   pfx("10.0.0.0/9"),
			wantPrefixes: pxv("10.128.0.0/9"),
			wantOK:       true,
		},
		{
			name: "on prefix left",
			f: func(s *IPSet) {
				s.AddPrefix(pfx("10.0.0.0/8"))
				s.RemovePrefix(pfx("10.0.0.0/9"))
			},
			b:            9,
			wantPrefix:   pfx("10.128.0.0/9"),
			wantPrefixes: nil,
			wantOK:       true,
		},
	}
	for _, tt := range tests {
		var s IPSet
		tt.f(&s)
		got, ok := s.RemoveFreePrefix(tt.b)
		if ok != tt.wantOK {
			t.Errorf("extractPrefix() ok = %t, wantOK %t", ok, tt.wantOK)
			return
		}
		if !reflect.DeepEqual(got, tt.wantPrefix) {
			t.Errorf("extractPrefix() = %v, want %v", got, tt.wantPrefix)
		}
		if !reflect.DeepEqual(s.Prefixes(), tt.wantPrefixes) {
			t.Errorf("extractPrefix() = %v, want %v", s.Prefixes(), tt.wantPrefixes)
		}
	}
}

func TestIPSetContainsFunc(t *testing.T) {
	var s IPSet
	s.AddPrefix(mustIPPrefix("10.0.0.0/8"))
	s.AddPrefix(mustIPPrefix("1.2.3.4/32"))
	s.AddPrefix(mustIPPrefix("fc00::/7"))
	contains := s.ContainsFunc()

	tests := []struct {
		ip   string
		want bool
	}{
		{"0.0.0.0", false},
		{"::", false},

		{"1.2.3.3", false},
		{"1.2.3.4", true},
		{"1.2.3.5", false},

		{"9.255.255.255", false},
		{"10.0.0.0", true},
		{"10.1.2.3", true},
		{"10.255.255.255", true},
		{"11.0.0.0", false},

		{"::", false},
		{"fc00::", true},
		{"fc00::1", true},
		{"fd00::1", true},
		{"ff00::1", false},
	}
	for _, tt := range tests {
		got := contains(mustIP(tt.ip))
		if got != tt.want {
			t.Errorf("contains(%q) = %v; want %v", tt.ip, got, tt.want)
		}
	}
}

func TestIPSetFuzz(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		doIPSetFuzz(t, 100)
	} else {
		doIPSetFuzz(t, 5000)
	}
}

func BenchmarkIPSetFuzz(b *testing.B) {
	b.ReportAllocs()
	doIPSetFuzz(b, b.N)
}

func doIPSetFuzz(t testing.TB, iters int) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)
	debugf = logger.Printf
	defer func() { debugf = discardf }()
	for i := 0; i < iters; i++ {
		buf.Reset()
		steps, set, wantContains := newRandomIPSet()
		contains := set.ContainsFunc()
		for b, want := range wantContains {
			ip := IPv4(0, 0, 0, uint8(b))
			got := contains(ip)
			if got != want {
				t.Fatalf("for steps %q, contains(%v) = %v; want %v\n%s", steps, ip, got, want, buf.Bytes())
			}
		}
	}
}

func newRandomIPSet() (steps []string, s *IPSet, wantContains [256]bool) {
	s = new(IPSet)
	nstep := 2 + rand.Intn(10)
	for i := 0; i < nstep; i++ {
		op := rand.Intn(2)
		ip1 := uint8(rand.Intn(256))
		ip2 := uint8(rand.Intn(256))
		if ip2 < ip1 {
			ip1, ip2 = ip2, ip1
		}
		var v bool
		switch op {
		case 0:
			steps = append(steps, fmt.Sprintf("add 0.0.0.%d-0.0.0.%d", ip1, ip2))
			s.AddRange(IPRange{From: IPv4(0, 0, 0, ip1), To: IPv4(0, 0, 0, ip2)})
			v = true
		case 1:
			steps = append(steps, fmt.Sprintf("remove 0.0.0.%d-0.0.0.%d", ip1, ip2))
			s.RemoveRange(IPRange{From: IPv4(0, 0, 0, ip1), To: IPv4(0, 0, 0, ip2)})
		}
		for i := ip1; i <= ip2; i++ {
			wantContains[i] = v
			if i == ip2 {
				break
			}
		}
	}
	return
}

// TestIPSetRanges tests IPSet.Ranges against 64k
// patterns of sets of ranges, checking the real implementation
// against the test's separate implementation.
//
// For each of uint16 pattern, each set bit is treated as an IP that
// should be in the set's resultant Ranges.
func TestIPSetRanges(t *testing.T) {
	t.Parallel()
	upper := 0x0fff
	if *long {
		upper = 0xffff
	}
	for pat := 0; pat <= upper; pat++ {
		var s IPSet
		var from, to IP
		ranges := make([]IPRange, 0)
		flush := func() {
			r := IPRange{From: from, To: to}
			s.AddRange(r)
			ranges = append(ranges, r)
			from, to = IP{}, IP{}
		}
		for b := uint16(0); b < 16; b++ {
			if uint16(pat)&(1<<b) != 0 {
				ip := IPv4(1, 0, 0, uint8(b))
				to = ip
				if from.IsZero() {
					from = ip
				}
				continue
			}
			if !from.IsZero() {
				flush()
			}
		}
		if !from.IsZero() {
			flush()
		}
		got := s.Ranges()
		if !reflect.DeepEqual(got, ranges) {
			t.Errorf("for %016b: got %v; want %v", pat, got, ranges)
		}
	}
}

func TestIPSetRangesStress(t *testing.T) {
	t.Parallel()
	n := 50
	if testing.Short() {
		n /= 10
	} else if *long {
		n = 500
	}
	randRange := func() (a, b int, r IPRange) {
		a, b = rand.Intn(0x10000), rand.Intn(0x10000)
		if a > b {
			a, b = b, a
		}
		return a, b, IPRange{
			From: IPv4(0, 0, uint8(a>>8), uint8(a)),
			To:   IPv4(0, 0, uint8(b>>8), uint8(b)),
		}
	}
	for i := 0; i < n; i++ {
		var s IPSet
		var want [0xffff]bool
		// Add some ranges
		const maxAdd = 10
		for i := 0; i < 1+rand.Intn(2); i++ {
			a, b, r := randRange()
			for i := a; i <= b; i++ {
				want[i] = true
			}
			s.AddRange(r)
		}
		// Remove some ranges
		for i := 0; i < rand.Intn(3); i++ {
			a, b, r := randRange()
			for i := a; i <= b; i++ {
				want[i] = false
			}
			s.RemoveRange(r)
		}
		ranges := s.Ranges()

		// Make sure no ranges are adjacent or overlapping
		for i, r := range ranges {
			if i == 0 {
				continue
			}
			if ranges[i-1].To.Compare(r.From) != -1 {
				t.Fatalf("overlapping ranges: %v", ranges)
			}
		}

		// Copy the ranges back to a new set before using
		// ContainsFunc, in case the ContainsFunc implementation
		// changes in the future to not use Ranges itself:
		var s2 IPSet
		for _, r := range ranges {
			s2.AddRange(r)
		}
		contains := s2.ContainsFunc()
		for i, want := range want {
			if got := contains(IPv4(0, 0, uint8(i>>8), uint8(i))); got != want {
				t.Fatal("failed")
			}
		}
	}
}

func TestPointLess(t *testing.T) {
	tests := []struct {
		a, b point
		want bool
	}{
		// IPs sort first.
		{
			point{ip: mustIP("1.2.3.4"), want: false, start: true},
			point{ip: mustIP("1.2.3.5"), want: false, start: true},
			true,
		},

		// Starts.
		{
			point{ip: mustIP("1.1.1.1"), want: false, start: true},
			point{ip: mustIP("1.1.1.1"), want: true, start: true},
			true,
		},
		{
			point{ip: mustIP("2.2.2.2"), want: true, start: true},
			point{ip: mustIP("2.2.2.2"), want: false, start: true},
			false,
		},

		// Ends.
		{
			point{ip: mustIP("3.3.3.3"), want: true, start: false},
			point{ip: mustIP("3.3.3.3"), want: false, start: false},
			false,
		},
		{
			point{ip: mustIP("4.4.4.4"), want: false, start: false},
			point{ip: mustIP("4.4.4.4"), want: true, start: false},
			true,
		},

		// End & start at same IP.
		{
			point{ip: mustIP("5.5.5.5"), want: true, start: true},
			point{ip: mustIP("5.5.5.5"), want: true, start: false},
			true,
		},
		{
			point{ip: mustIP("6.6.6.6"), want: true, start: false},
			point{ip: mustIP("6.6.6.6"), want: true, start: true},
			false,
		},

		// For same IP & both start, unwanted comes first.
		{
			point{ip: mustIP("7.7.7.7"), want: false, start: true},
			point{ip: mustIP("7.7.7.7"), want: true, start: true},
			true,
		},
		{
			point{ip: mustIP("8.8.8.8"), want: true, start: true},
			point{ip: mustIP("8.8.8.8"), want: false, start: true},
			false,
		},

		// And not-want-end should come after a do-want-start.
		{
			point{ip: mustIP("10.0.0.30"), want: false, start: false},
			point{ip: mustIP("10.0.0.30"), want: true, start: true},
			false,
		},
		{
			point{ip: mustIP("10.0.0.40"), want: true, start: true},
			point{ip: mustIP("10.0.0.40"), want: false, start: false},
			true,
		},

		// A not-want start should come before a not-want want.
		{
			point{ip: mustIP("10.0.0.9"), want: false, start: true},
			point{ip: mustIP("10.0.0.9"), want: false, start: false},
			true,
		},
		{
			point{ip: mustIP("10.0.0.9"), want: false, start: false},
			point{ip: mustIP("10.0.0.9"), want: false, start: true},
			false,
		},
	}
	for _, tt := range tests {
		got := tt.a.Less(tt.b)
		if got != tt.want {
			t.Errorf("Less(%+v, %+v) = %v; want %v", tt.a, tt.b, got, tt.want)
			continue
		}
		got2 := tt.b.Less(tt.a)
		if got && got2 {
			t.Errorf("Less(%+v, %+v) = properly true; but is also true in reverse", tt.a, tt.b)
		}
		if !got && !got2 && tt.a != tt.b {
			t.Errorf("Less(%+v, %+v) both false but unequal", tt.a, tt.b)
		}
	}

}