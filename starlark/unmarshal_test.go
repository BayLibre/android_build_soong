package starlark

import (
	"reflect"
	"testing"

	"go.starlark.net/starlark"
)

func createStarlarkValue(t *testing.T, code string) starlark.Value {
	t.Helper()
	result, err := starlark.ExecFile(&starlark.Thread{}, "main.bzl", "x = "+code, builtins)
	if err != nil {
		t.Error(err)
		return nil
	}
	return result["x"]
}

func TestUnmarshallConcreteType(t *testing.T) {
	x, err := Unmarshal[string](createStarlarkValue(t, `"foo"`))
	if err != nil {
		t.Error(err)
		return
	}
	if x != "foo" {
		t.Errorf(`Expected "foo", got %q`, x)
	}
}

func TestUnmarshall(t *testing.T) {
	testCases := []struct {
		input    string
		expected interface{}
	}{
		{
			input:    `"foo"`,
			expected: "foo",
		},
		{
			input:    `5`,
			expected: 5,
		},
		{
			input:    `["foo", "bar"]`,
			expected: []string{"foo", "bar"},
		},
		{
			input:    `("foo", "bar")`,
			expected: []string{"foo", "bar"},
		},
		{
			input:    `{"foo": 5, "bar": 10}`,
			expected: map[string]int{"foo": 5, "bar": 10},
		},
		{
			input:    `{"foo": ["qux"], "bar": []}`,
			expected: map[string][]string{"foo": {"qux"}, "bar": nil},
		},
		{
			input: `struct(Foo="foo", Bar=5)`,
			expected: struct {
				Foo string
				Bar int
			}{Foo: "foo", Bar: 5},
		},
		{
			// Unexported fields version of the above
			input: `struct(foo="foo", bar=5)`,
			expected: struct {
				foo string
				bar int
			}{foo: "foo", bar: 5},
		},
	}

	for _, tc := range testCases {
		x, err := UnmarshalReflect(createStarlarkValue(t, tc.input), reflect.TypeOf(tc.expected))
		if err != nil {
			t.Error(err)
			continue
		}
		if !reflect.DeepEqual(x.Interface(), tc.expected) {
			t.Errorf(`Expected %#v, got %#v`, tc.expected, x.Interface())
		}
	}
}
