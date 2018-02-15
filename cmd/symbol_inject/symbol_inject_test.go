package main

import (
	"bytes"
	"strconv"
	"testing"
)

func TestCopyAndInject(t *testing.T) {
	s := "abcdefghijklmnopqrstuvwxyz"
	testCases := []struct {
		offset, size uint64
		value        string
		expected     string
	}{
		{
			offset:   0,
			size:     1,
			value:    "A",
			expected: "Abcdefghijklmnopqrstuvwxyz",
		},
		{
			offset:   1,
			size:     1,
			value:    "B",
			expected: "aBcdefghijklmnopqrstuvwxyz",
		},
		{
			offset:   1,
			size:     1,
			value:    "BCD",
			expected: "aBcdefghijklmnopqrstuvwxyz",
		},
		{
			offset:   25,
			size:     1,
			value:    "Z",
			expected: "abcdefghijklmnopqrstuvwxyZ",
		},
	}

	for i, testCase := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			in := bytes.NewReader([]byte(s))
			out := &bytes.Buffer{}
			copyAndInject(in, out, testCase.offset, testCase.size, testCase.value)

			if out.String() != testCase.expected {
				t.Errorf("expected %s, got %s", testCase.expected, out.String())
			}
		})
	}
}
