// Copyright 2019 Google Inc. All rights reserved.
//
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

package java

import (
	"reflect"
	"testing"

	"android/soong/android"
)

func TestRequired(t *testing.T) {
	ctx, config := testJava(t, `
		java_library {
			name: "foo",
			srcs: ["a.java"],
			required: ["libfoo"],
		}
	`)

	mod := ctx.ModuleForTests("foo", "android_common").Module()
	entries := android.AndroidMkEntriesForTest(t, config, "", mod)[0]

	expected := []string{"libfoo"}
	actual := entries.EntryMap["LOCAL_REQUIRED_MODULES"]
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Unexpected required modules - expected: %q, actual: %q", expected, actual)
	}
}

func TestHostdex(t *testing.T) {
	ctx, config := testJava(t, `
		java_library {
			name: "foo",
			srcs: ["a.java"],
			hostdex: true,
		}
	`)

	mod := ctx.ModuleForTests("foo", "android_common").Module()
	entriesList := android.AndroidMkEntriesForTest(t, config, "", mod)
	if len(entriesList) != 2 {
		t.Errorf("two entries are expected, but got %d", len(entriesList))
	}

	mainEntries := &entriesList[0]
	expected := []string{"foo"}
	actual := mainEntries.EntryMap["LOCAL_MODULE"]
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Unexpected module name - expected: %q, actual: %q", expected, actual)
	}

	subEntries := &entriesList[1]
	expected = []string{"foo-hostdex"}
	actual = subEntries.EntryMap["LOCAL_MODULE"]
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Unexpected module name - expected: %q, actual: %q", expected, actual)
	}
}

func TestHostdexRequired(t *testing.T) {
	ctx, config := testJava(t, `
		java_library {
			name: "foo",
			srcs: ["a.java"],
			hostdex: true,
			required: ["libfoo"],
		}
	`)

	mod := ctx.ModuleForTests("foo", "android_common").Module()
	entriesList := android.AndroidMkEntriesForTest(t, config, "", mod)
	if len(entriesList) != 2 {
		t.Errorf("two entries are expected, but got %d", len(entriesList))
	}

	mainEntries := &entriesList[0]
	expected := []string{"libfoo"}
	actual := mainEntries.EntryMap["LOCAL_REQUIRED_MODULES"]
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Unexpected required modules - expected: %q, actual: %q", expected, actual)
	}

	subEntries := &entriesList[1]
	expected = []string{"libfoo"}
	actual = subEntries.EntryMap["LOCAL_REQUIRED_MODULES"]
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Unexpected required modules - expected: %q, actual: %q", expected, actual)
	}
}

func TestHostdexSpecificRequired(t *testing.T) {
	ctx, config := testJava(t, `
		java_library {
			name: "foo",
			srcs: ["a.java"],
			hostdex: true,
			target: {
				hostdex: {
					required: ["libfoo"],
				},
			},
		}
	`)

	mod := ctx.ModuleForTests("foo", "android_common").Module()
	entriesList := android.AndroidMkEntriesForTest(t, config, "", mod)
	if len(entriesList) != 2 {
		t.Errorf("two entries are expected, but got %d", len(entriesList))
	}

	mainEntries := &entriesList[0]
	if r, ok := mainEntries.EntryMap["LOCAL_REQUIRED_MODULES"]; ok {
		t.Errorf("Unexpected required modules: %q", r)
	}

	subEntries := &entriesList[1]
	expected := []string{"libfoo"}
	actual := subEntries.EntryMap["LOCAL_REQUIRED_MODULES"]
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Unexpected required modules - expected: %q, actual: %q", expected, actual)
	}
}

func TestJavaSdkLibrary_RequireXmlPermissionFile(t *testing.T) {
	ctx, config := testJava(t, `
		java_sdk_library {
			name: "foo-shared_library",
			srcs: ["a.java"],
		}
		java_sdk_library {
			name: "foo-no_shared_library",
			srcs: ["a.java"],
			shared_library: false,
		}
		`)

	// Verify the existence of internal modules
	ctx.ModuleForTests("foo-shared_library.xml", "android_common")

	testCases := []struct {
		moduleName string
		expected   []string
	}{
		{"foo-shared_library", []string{"foo-shared_library.xml"}},
		{"foo-no_shared_library", nil},
	}
	for _, tc := range testCases {
		mod := ctx.ModuleForTests(tc.moduleName, "android_common").Module()
		entries := android.AndroidMkEntriesForTest(t, config, "", mod)[0]
		actual := entries.EntryMap["LOCAL_REQUIRED_MODULES"]
		if !reflect.DeepEqual(tc.expected, actual) {
			t.Errorf("Unexpected required modules - expected: %q, actual: %q", tc.expected, actual)
		}
	}
}

func TestAndroidAppImportExportPackageResources(t *testing.T) {
	ctx, config := testJava(t, `
		android_app_import {
			name: "framework-res",
			certificate: "platform",
			apk: "package-res.apk",
			prefer: true,
			export_package_resources: true,
		}
	`)

	mod := ctx.ModuleForTests("prebuilt_framework-res", "android_common").Module()
	entries := android.AndroidMkEntriesForTest(t, config, "", mod)[0]

	expectedPath := "."
	expectedPrebuiltModuleFile := buildDir + "/.intermediates/prebuilt_framework-res/android_common/verify_uses_libraries/package-res.apk"
	// Verify installation to /system/framework.
	expectedSoongBuiltInstalledOdex := buildDir + "/.intermediates/prebuilt_framework-res/android_common/dexpreopt/oat/arm64/package.odex:/system/framework/oat/arm64/framework-res.odex"
	expectedSoongBuiltInstalledVdex := buildDir + "/.intermediates/prebuilt_framework-res/android_common/dexpreopt/oat/arm64/package.vdex:/system/framework/oat/arm64/framework-res.vdex"
	expectedSoongBuiltInstalled := expectedSoongBuiltInstalledOdex + " " + expectedSoongBuiltInstalledVdex
	// Verify that the apk is preprocessed: The export package is the same
	// as the prebuilt.
	expectedSoongResourceExportPackage := expectedPrebuiltModuleFile

	actualPath := entries.EntryMap["LOCAL_PATH"]
	actualPrebuiltModuleFile := entries.EntryMap["LOCAL_PREBUILT_MODULE_FILE"]
	actualSoongBuiltInstalled := entries.EntryMap["LOCAL_SOONG_BUILT_INSTALLED"]
	actualSoongResourceExportPackage := entries.EntryMap["LOCAL_SOONG_RESOURCE_EXPORT_PACKAGE"]

	if len(actualPath) != 1 {
		t.Errorf("LOCAL_PATH incorrect len %d", len(actualPath))
	} else {
		if actualPath[0] != expectedPath {
			t.Errorf("LOCAL_PATH mismatch, actual: %s, expected: %s", actualPath[0], expectedPath)
		}
	}

	if len(actualPrebuiltModuleFile) != 1 {
		t.Errorf("LOCAL_PREBUILT_MODULE_FILE incorrect len %d", len(actualPrebuiltModuleFile))
	} else {
		if actualPrebuiltModuleFile[0] != expectedPrebuiltModuleFile {
			t.Errorf("LOCAL_PREBUILT_MODULE_FILE mismatch, actual: %s, expected: %s", actualPrebuiltModuleFile[0], expectedPrebuiltModuleFile)
		}
	}

	if len(actualSoongBuiltInstalled) != 1 {
		t.Errorf("LOCAL_SOONG_BUILT_INSTALLED incorrect len %d", len(actualSoongBuiltInstalled))
	} else {
		if actualSoongBuiltInstalled[0] != expectedSoongBuiltInstalled {
			t.Errorf("LOCAL_SOONG_BUILT_INSTALLED mismatch, actual: %s, expected: %s", actualSoongBuiltInstalled[0], expectedSoongBuiltInstalled)
		}
	}

	if len(actualSoongResourceExportPackage) != 1 {
		t.Errorf("LOCAL_SOONG_RESOURCE_EXPORT_PACKAGE incorrect len %d", len(actualSoongResourceExportPackage))
	} else {
		if actualSoongResourceExportPackage[0] != expectedSoongResourceExportPackage {
			t.Errorf("LOCAL_SOONG_RESOURCE_EXPORT_PACKAGE mismatch, actual: %s, expected: %s", actualSoongResourceExportPackage[0], expectedSoongResourceExportPackage)
		}
	}
}

func TestImportSoongDexJar(t *testing.T) {
	ctx, config := testJava(t, `
		java_import {
			name: "my-java-import",
			jars: ["a.jar"],
			prefer: true,
			compile_dex: true,
		}
	`)

	mod := ctx.ModuleForTests("my-java-import", "android_common").Module()
	entries := android.AndroidMkEntriesForTest(t, config, "", mod)[0]
	expectedSoongDexJar := buildDir + "/.intermediates/my-java-import/android_common/dex/my-java-import.jar"
	actualSoongDexJar := entries.EntryMap["LOCAL_SOONG_DEX_JAR"]

	if len(actualSoongDexJar) != 1 {
		t.Errorf("LOCAL_SOONG_DEX_JAR incorrect len %d", len(actualSoongDexJar))
	} else if actualSoongDexJar[0] != expectedSoongDexJar {
		t.Errorf("LOCAL_SOONG_DEX_JAR mismatch, actual: %s, expected: %s", actualSoongDexJar[0], expectedSoongDexJar)
	}
}
