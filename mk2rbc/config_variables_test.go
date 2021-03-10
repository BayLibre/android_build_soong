package mk2rbc

import (
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

type testVar struct {
	name string
	cl   varClass
	ty   starlarkType
}

type vars struct {
	v []testVar
}

func (v *vars) NewVariable(name string, varClass varClass, valueType starlarkType) {
	v.v = append(v.v, testVar{name, varClass, valueType})
}

func Test(t *testing.T) {
	_, myfile, _, _ := runtime.Caller(0)
	testFile := filepath.Join(filepath.Dir(myfile), "test", "config_variables.mk.test")
	var actual vars
	if err := FindConfigVariables(testFile, &actual); err != nil {
		t.Fatal(err)
	}
	expected := vars{[]testVar{
		{"PRODUCT_NAME", VarClassConfig, starlarkTypeUnknown},
		{"PRODUCT_MODEL", VarClassConfig, starlarkTypeUnknown},
		{"PRODUCT_LOCALES", VarClassConfig, starlarkTypeList},
		{"PRODUCT_AAPT_CONFIG", VarClassConfig, starlarkTypeList},
		{"PRODUCT_AAPT_PREF_CONFIG", VarClassConfig, starlarkTypeUnknown},
	}}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("\nExpected: %v\n  Actual: %v", expected, actual)
	}
}
