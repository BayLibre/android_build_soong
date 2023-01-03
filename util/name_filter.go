package util

import (
	"fmt"
	"regexp"
	"strings"
)

// A filter has a single function, `match`.
// We use three filter kinds: `always` matches anything,
// `never` matches nothing, `rexFilter` matchers regex
type filter interface {
	match(string) bool
}

type always struct{}

func (_ always) match(_ string) bool {
	return true
}

type never struct{}

func (_ never) match(_ string) bool {
	return false
}

type rexFilter struct {
	expr *regexp.Regexp
}

func (r rexFilter) match(s string) bool {
	return r.expr.MatchString(s)
}

// plusMinus holds a positive and a negative filter.
type plusMinus struct {
	allow filter
	deny  filter
}

// NameFilter provides `Match` function.
type NameFilter struct {
	exprs []plusMinus
}

// Match checks given string against the sequence of regular expressions.
func (m NameFilter) Match(s string) bool {
	for _, ad := range m.exprs {
		if ad.allow.match(s) {
			return true
		}
		if ad.deny.match(s) {
			return false
		}
	}
	panic(fmt.Errorf("fell of filter %v", m))
}

// NewNameFilter builds NameFilter from the list of filter strings.
// Each filter string is a regular expression, optionally preceded
// by a '-' sign. A matcher returned by this function will _fully_ match
// a given string against consecutive filters. The matching process stops
// at the first successful  match and returns the polarity of this match
// (i.e., a positive matched filter returns true, a negative one false).
// If none of the filters match, match fails.
func NewNameFilter(items []string) (*NameFilter, error) {
	if len(items) == 0 {
		return &NameFilter{[]plusMinus{{&never{}, &always{}}}}, nil
	}
	nextIndex := 0
	next := func() string {
		if nextIndex >= len(items) {
			panic("bad next")
		}
		s := items[nextIndex]
		nextIndex++
		return s
	}
	back := func() {
		if nextIndex <= 0 {
			panic("bad push back")
		}
		nextIndex--
	}
	hasNext := func() bool {
		return nextIndex < len(items)
	}

	var rexBuf strings.Builder
	var rexSep string
	newRexBuf := func() {
		rexBuf = strings.Builder{}
		rexSep = "^"
	}
	rexBufAppend := func(item string) {
		rexBuf.WriteString(rexSep)
		rexSep = "|"
		rexBuf.WriteString(item)
	}
	rexBufToRexFilter := func() (filter, error) {
		if rexSep == "^" {
			return nil, nil
		}
		rexBuf.WriteString("$")
		rex, err := regexp.Compile(rexBuf.String())
		if err != nil {
			return nil, fmt.Errorf("bad rexexp %q: %s", rex.String(), err)
		}
		return rexFilter{rex}, nil
	}

	buildAllow := func() (filter, error) {
		newRexBuf()
		for hasNext() {
			item := next()
			if item == "" {
				continue
			}
			if item[0] == '-' {
				back()
				break
			}
			if item == ".*" {
				return always{}, nil
			}
			rexBufAppend(item)
		}
		return rexBufToRexFilter()
	}
	buildDeny := func() (filter, error) {
		newRexBuf()
		for hasNext() {
			item := next()
			if item == "" || item[0] != '-' {
				back()
				break
			}
			if item == "-.*" {
				return always{}, nil
			} else if item == "-" {
				continue
			}
			rexBufAppend(item[1:])
		}
		return rexBufToRexFilter()
	}

	var allow, deny filter
	var err error
	m := &NameFilter{}
	for hasNext() {
		if allow, err = buildAllow(); err != nil {
			return nil, err
		}
		if allow == nil {
			allow = never{}
		}
		if deny, err = buildDeny(); err != nil {
			return nil, err
		}
		if deny == nil {
			deny = always{}
		}
		m.exprs = append(m.exprs, struct{ allow, deny filter }{allow, deny})
	}

	return m, nil
}
