package mk2rbc

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestProductsMakefile(t *testing.T) {
	testDir := getTestDirectory()
	abspath := func(relPath string) string { return filepath.Join(testDir, relPath) }
	actualProducts := make(map[string]string)
	if err := UpdateProductConfigMap(actualProducts, abspath("android_products.mk.test")); err != nil {
		t.Fatal(err)
	}
	expectedProducts := map[string]string{
		"aosp_cf_x86_tv": abspath("vsoc_x86/tv/device.mk"),
		"aosp_tv_arm":    abspath("aosp_tv_arm.mk"),
		"aosp_tv_arm64":  abspath("aosp_tv_arm64.mk"),
	}
	if !reflect.DeepEqual(actualProducts, expectedProducts) {
		t.Errorf("\nExpected: %v\n  Actual: %v", expectedProducts, actualProducts)
	}
}
