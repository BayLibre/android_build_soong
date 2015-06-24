package main

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	bpparser "github.com/google/blueprint/parser"
)

var valueTestCases = []struct {
	blueprint string
	expected  string
}{
	{
		blueprint: `test = false`,
		expected:  `false`,
	},
	{
		blueprint: `test = Variable`,
		expected:  `$(Variable)`,
	},
	{
		blueprint: `test = "string"`,
		expected:  `string`,
	},
	{
		blueprint: `test = ["a", "b"]`,
		expected: `\
    a \
    b`,
	},
	{
		blueprint: `test = Var + "b"`,
		expected:  `$(Var)b`,
	},
	{
		blueprint: `test = ["a"] + ["b"]`,
		expected: `\
    a\
    b`,
	},
}

func TestValueToString(t *testing.T) {
	for _, testCase := range valueTestCases {
		blueprint, errs := bpparser.Parse("", strings.NewReader(testCase.blueprint), nil)
		if len(errs) > 0 {
			t.Errorf("Failed to read blueprint: %q", errs)
		}

		str := valueToString(blueprint.Defs[0].(*bpparser.Assignment).Value)
		expect(t, testCase.blueprint, testCase.expected, str)
	}
}

var moduleTestCases = []struct {
	blueprint string
	androidmk string
}{
	// Target-only
	{
		blueprint: `cc_library_shared { name: "test", }`,
		androidmk: `include $(CLEAR_VARS)
LOCAL_MODULE := test
include $(BUILD_SHARED_LIBRARY)

`,
	},
	// Host-only
	{
		blueprint: `cc_library_host_shared { name: "test", }`,
		androidmk: `include $(CLEAR_VARS)
LOCAL_MODULE := test
include $(BUILD_HOST_SHARED_LIBRARY)

`,
	},
	// Target and Host
	{
		blueprint: `cc_library_shared { name: "test", host_supported: true, }`,
		androidmk: `include $(CLEAR_VARS)
LOCAL_MODULE := test
include $(BUILD_SHARED_LIBRARY)

include $(CLEAR_VARS)
LOCAL_MODULE := test
include $(BUILD_HOST_SHARED_LIBRARY)

`,
	},
	// Static and Shared
	{
		blueprint: `cc_library { name: "test", }`,
		androidmk: `include $(CLEAR_VARS)
LOCAL_MODULE := test
include $(BUILD_SHARED_LIBRARY)

include $(CLEAR_VARS)
LOCAL_MODULE := test
include $(BUILD_STATIC_LIBRARY)

`,
	},
}

func TestModules(t *testing.T) {
	for _, testCase := range moduleTestCases {
		blueprint, errs := bpparser.Parse("", strings.NewReader(testCase.blueprint), nil)
		if len(errs) > 0 {
			t.Errorf("Failed to read blueprint: %q", errs)
		}

		buf := &bytes.Buffer{}
		writer := &androidMkWriter{
			blueprint: blueprint,
			path:      "",
			mapScope:  make(map[string][]*bpparser.Property),
			Writer:    bufio.NewWriter(buf),
		}

		module := blueprint.Defs[0].(*bpparser.Module)
		writer.handleModule(module)
		writer.Flush()

		expect(t, testCase.blueprint, testCase.androidmk, buf.String())
	}
}

func expect(t *testing.T, testCase string, expected string, out string) {
	if expected != out {
		t.Errorf("test case: %s", testCase)
		t.Errorf("unexpected difference:")
		t.Errorf("  expected: \"\"\"%s\"\"\"", expected)
		t.Errorf("       got: \"\"\"%s\"\"\"", out)
	}
}
