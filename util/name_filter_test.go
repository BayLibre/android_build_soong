package util

import (
	"strings"
	"testing"
)

func TestNameMatcher_Match(t *testing.T) {
	type pair struct {
		in   string
		want bool
	}
	tests := []struct {
		filter string
		items  []pair
	}{
		{
			items: []pair{
				{"foo", false},
			},
		},
		{
			filter: "a.* .* -.* ",
			items: []pair{
				{"abcd", true},
				{"xyz", true},
			},
		},
		{
			filter: "-.* .*",
			items: []pair{
				{"abc", false},
			},
		},
		{
			filter: "a.* b.*",
			items: []pair{
				{"azx", true},
				{"bzx", true},
				{"ca", false},
			},
		},
		{
			filter: "a.*z -b[^z].*  b.*",
			items: []pair{
				{"aq", false},
				{"axz", true},
				{"ba", false},
				{"bza", true},
			},
		},
		{
			filter: "a.* b.* -c.* -d.* .*",
			items: []pair{
				{"ab", true},
				{"ba", true},
				{"cf", false},
				{"ea", true},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.filter, func(t *testing.T) {
			filter, err := NewNameFilter(strings.Split(tt.filter, " "))
			if err != nil {
				t.Errorf("bad filter %q: %s", tt.filter, err)
				return
			}
			for _, item := range tt.items {
				if got := filter.Match(item.in); got != item.want {
					t.Errorf("Match(%q) = %v, wanted %v", item.in, got, item.want)
				}
			}
		})
	}
}
