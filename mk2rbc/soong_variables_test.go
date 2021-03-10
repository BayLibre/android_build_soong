package mk2rbc

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
)

type dirResolverForTest struct {
	ScopeBase
}

func (t dirResolverForTest) Get(name string) string {
	if name != "BUILD_SYSTEM" {
		return fmt.Sprintf("$(%s)", name)
	}
	return getTestDirectory()
}

func TestSoongVariables(t *testing.T) {
	testFile := filepath.Join(getTestDirectory(), "soong_variables.mk.test")
	var actual testVariables
	if err := FindSoongVariables(testFile, dirResolverForTest{}, &actual); err != nil {
		t.Fatal(err)
	}
	expected := testVariables{[]testVar{
		{"BUILD_ID", VarClassSoong, starlarkTypeString},
		{"PLATFORM_SDK_VERSION", VarClassSoong, starlarkTypeInt},
		{"DEVICE_PACKAGE_OVERLAYS", VarClassSoong, starlarkTypeList},
		{"ENABLE_PREOPT", VarClassSoong, starlarkTypeBool},
	}}
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("\nExpected: %v\n  Actual: %v", expected, actual)
	}
}
