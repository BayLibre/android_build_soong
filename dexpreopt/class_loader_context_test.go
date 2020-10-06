// Copyright 2020 Google Inc. All rights reserved.
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

package dexpreopt

// This file contains unit tests for class loader context structure.
// For class loader context tests involving .bp files, see TestUsesLibraries in java package.

import (
	"reflect"
	"strings"
	"testing"

	"android/soong/android"
)

func TestCLC(t *testing.T) {
	ctx := testContext()

	shared := true     // dependencies are not added to uses libs
	nonshared := false // dependencies are added to uses libs

	// A few class loader contexts to be used as nested subcontexts, nothing special about them.
	m1 := make(ClassLoaderContextMap)
	m1.AddContext(ctx, "m1_a", nonshared, buildPath(ctx, "m1_a"), installPath(ctx, "m1_a"))
	m1.AddContext(ctx, "m1_b", shared, buildPath(ctx, "m1_b"), installPath(ctx, "m1_b"))
	m2 := make(ClassLoaderContextMap)
	m2.AddContext(ctx, "m2_a", nonshared, buildPath(ctx, "m2_a"), installPath(ctx, "m2_a"))
	m2.AddContext(ctx, "m2_b", shared, buildPath(ctx, "m2_b"), installPath(ctx, "m2_b"))
	m2.AddContextForSdk(ctx, AnySdkVersion, "m2_c", shared, buildPath(ctx, "m2_c"), installPath(ctx, "m2_c"), m1)
	m4 := make(ClassLoaderContextMap)
	m4.AddContext(ctx, "m4_a", nonshared, buildPath(ctx, "m4_a"), installPath(ctx, "m4_a"))
	m4.AddContext(ctx, "m4_b", shared, buildPath(ctx, "m4_b"), installPath(ctx, "m4_b"))

	m := make(ClassLoaderContextMap)

	// Simple cases.
	m.AddContext(ctx, "m_a", nonshared, buildPath(ctx, "m_a"), installPath(ctx, "m_a"))
	m.AddContext(ctx, "m_b", shared, buildPath(ctx, "m_b"), installPath(ctx, "m_b"))

	// Test that an unexpected unknown build path causes immediate error.
	t.Run("build", func(t *testing.T) {
		defer testFail(t, "unknown build path to <uses-library> \"m_a\"")
		m.AddContext(ctx, "m_a", nonshared, nil, nil)
	})

	// Test that an unexpected unknown install path causes immediate error.
	t.Run("install", func(t *testing.T) {
		defer testFail(t, "unknown install path to <uses-library> \"m_a\"")
		m.AddContext(ctx, "m_a", nonshared, buildPath(ctx, "m_a"), nil)
	})

	// "Maybe" variant in the good case: add as usual.
	m_c := "m_c"
	m.MaybeAddContext(ctx, &m_c, nonshared, buildPath(ctx, "m_c"), installPath(ctx, "m_c"))

	// "Maybe" variant in the bad case: don't add library with unknown name, keep going.
	m.MaybeAddContext(ctx, nil, nonshared, nil, nil)

	// "Maybe" variant in the middle case (use a separate map to keep the main one valid).
	m3 := make(ClassLoaderContextMap)
	m_x := "m_x"
	m3.MaybeAddContext(ctx, &m_x, nonshared, nil, nil)
	// The library should be added to <uses-library> tags by the manifest_fixer.
	t.Run("uses libs (maybe case)", func(t *testing.T) {
		haveUsesLibs := m3.UsesLibs()
		wantUsesLibs := []string{"m_x"}
		if !reflect.DeepEqual(wantUsesLibs, haveUsesLibs) {
			t.Errorf("\nwant uses libs: %s\nhave uses libs: %s", wantUsesLibs, haveUsesLibs)
		}
	})
	// But class loader context in such cases should fail validation.
	t.Run("validate (maybe case)", func(t *testing.T) {
		defer testFail(t, "invalid path for <uses-library> \"m_x\"")
		validateClassLoaderContext(ctx, m3)
	})

	// Compatibility libraries with unknown install paths get default paths.
	// Test that "android.hidl.manager" gets an extra dependency on "android.hidl.base".
	m.AddContextForSdk(ctx, 29, AndroidHidlManager, nonshared, buildPath(ctx, AndroidHidlManager), nil, nil)
	m.AddContextForSdk(ctx, 29, AndroidHidlBase, nonshared, buildPath(ctx, AndroidHidlBase), nil, nil)

	// Add "android.test.mock" without "android.test.runner", observe that is gets removed.
	m.AddContextForSdk(ctx, 30, AndroidTestMock, nonshared, buildPath(ctx, AndroidTestMock), nil, nil)

	// Add some libraries with nested subcontexts.
	m.AddContextForSdk(ctx, AnySdkVersion, "m_d", nonshared, buildPath(ctx, "m_d"), installPath(ctx, "m_d"), m2)
	m.AddContextForSdk(ctx, 28, "m_e", nonshared, buildPath(ctx, "m_e"), installPath(ctx, "m_e"), m1)

	// An attempt to add conditional nested subcontext should fail.
	t.Run("nested conditional", func(t *testing.T) {
		defer testFail(t, "nested class loader context shouldn't have conditional part")
		m3 := make(ClassLoaderContextMap)
		m3.AddContextForSdk(ctx, 42, "m_a", nonshared, buildPath(ctx, "m_a"), installPath(ctx, "m_a"), nil)
		m.AddContextForSdk(ctx, AnySdkVersion, "m_b", nonshared, buildPath(ctx, "m_b"), installPath(ctx, "m_b"), m3)
	})

	// When the same library is both in conditional and unconditional context, it should be removed
	// from conditional context.
	m.AddContextForSdk(ctx, 42, "m_f", nonshared, buildPath(ctx, "m_f"), installPath(ctx, "m_f"), nil)
	m.AddContextForSdk(ctx, AnySdkVersion, "m_f", nonshared, buildPath(ctx, "m_f"), installPath(ctx, "m_f"), nil)

	// Merge map with implicit root library that is among toplevel contexts => does nothing.
	m.AddContextMap(m1, "m_c")

	// Merge map with implicit root library that is not among toplevel contexts => all subcontexts
	// of the other map are added as toplevel contexts.
	m.AddContextMap(m4, "m_g")

	// Test the lookup of a subcontext.
	t.Run("find context", func(t *testing.T) {
		clc := m.findContext(AnySdkVersion, "m_d")
		if clc == nil {
			t.Errorf("cannot find context")
		}
		haveHost, _, _ := computeClassLoaderContextRec([]*ClassLoaderContext{clc})
		wantHost := "PCL[out/m_d.jar]" +
			"{PCL[out/m2_a.jar]#PCL[out/m2_b.jar]#PCL[out/m2_c.jar]" +
			"{PCL[out/m1_a.jar]#PCL[out/m1_b.jar]}}"
		if haveHost != wantHost {
			t.Errorf("found something, but it's not the expected \"m_d\" context")
		}
	})

	fixClassLoaderContext(ctx, m)

	// Test that validation is successful (all paths are known).
	t.Run("validate", func(t *testing.T) {
		if !validateClassLoaderContext(ctx, m) {
			t.Errorf("invalid class loader context")
		}
	})

	haveStr, havePaths := ComputeClassLoaderContext(m)

	wantStr := " --host-context-for-sdk 28 " +
		"PCL[out/m_e.jar]" +
		"{PCL[out/m1_a.jar]#PCL[out/m1_b.jar]}" +
		" --target-context-for-sdk 28 " +
		"PCL[/system/m_e.jar]" +
		"{PCL[/system/m1_a.jar]#PCL[/system/m1_b.jar]}" +
		" --host-context-for-sdk 29 " +
		"PCL[out/" + AndroidHidlManager + ".jar]" +
		"{PCL[out/" + AndroidHidlBase + ".jar]}#" +
		"PCL[out/" + AndroidHidlBase + ".jar]" +
		" --target-context-for-sdk 29 " +
		"PCL[/system/framework/" + AndroidHidlManager + ".jar]" +
		"{PCL[/system/framework/" + AndroidHidlBase + ".jar]}#" +
		"PCL[/system/framework/" + AndroidHidlBase + ".jar]" +
		" --host-context-for-sdk any " +
		"PCL[out/m_a.jar]#PCL[out/m_b.jar]#PCL[out/m_c.jar]#PCL[out/m_d.jar]" +
		"{PCL[out/m2_a.jar]#PCL[out/m2_b.jar]#PCL[out/m2_c.jar]" +
		"{PCL[out/m1_a.jar]#PCL[out/m1_b.jar]}}#" +
		"PCL[out/m_f.jar]#PCL[out/m4_a.jar]#PCL[out/m4_b.jar]" +
		" --target-context-for-sdk any " +
		"PCL[/system/m_a.jar]#PCL[/system/m_b.jar]#PCL[/system/m_c.jar]#PCL[/system/m_d.jar]" +
		"{PCL[/system/m2_a.jar]#PCL[/system/m2_b.jar]#PCL[/system/m2_c.jar]" +
		"{PCL[/system/m1_a.jar]#PCL[/system/m1_b.jar]}}#" +
		"PCL[/system/m_f.jar]#PCL[/system/m4_a.jar]#PCL[/system/m4_b.jar]"

	wantPaths := []string{"out/m_e.jar",
		"out/m1_a.jar", "out/m1_b.jar",
		"out/android.hidl.manager-V1.0-java.jar", "out/android.hidl.base-V1.0-java.jar",
		"out/m_a.jar", "out/m_b.jar", "out/m_c.jar", "out/m_d.jar",
		"out/m2_a.jar", "out/m2_b.jar", "out/m2_c.jar",
		"out/m_f.jar",
		"out/m4_a.jar", "out/m4_b.jar",
	}

	// Test that class loader context structure is correct.
	t.Run("string", func(t *testing.T) {
		if wantStr != haveStr {
			t.Errorf("\nwant class loader context: %s\nhave class loader context: %s", wantStr, haveStr)
		}
	})

	// Test that all expected build paths are gathered.
	t.Run("paths", func(t *testing.T) {
		if !reflect.DeepEqual(wantPaths, havePaths.Strings()) {
			t.Errorf("\nwant paths: %s\nhave paths: %s", wantPaths, havePaths)
		}
	})

	// Test for libraries that are added by the manifest_fixer.
	t.Run("uses libs", func(t *testing.T) {
		haveUsesLibs := m.UsesLibs()
		wantUsesLibs := []string{"m_a", "m_b", "m_c", "m_d", "m2_a", "m2_b", "m2_c", "m_f", "m4_a", "m4_b"}
		if !reflect.DeepEqual(wantUsesLibs, haveUsesLibs) {
			t.Errorf("\nwant uses libs: %s\nhave uses libs: %s", wantUsesLibs, haveUsesLibs)
		}
	})
}

func testFail(t *testing.T, errmsg string) {
	if r := recover(); r == nil {
		t.Errorf("\nwant error: '%s'\nhave: none", errmsg)
	} else if s, ok := r.(string); !ok || !strings.HasPrefix(s, errmsg) {
		t.Errorf("\nwant error: '%s'\nhave error: '%s'\n", errmsg, r)
	}
}

func testContext() android.ModuleInstallPathContext {
	config := android.TestConfig("out", nil, "", nil)
	return android.ModuleInstallPathContextForTesting(config)
}

func buildPath(ctx android.PathContext, lib string) android.Path {
	return android.PathForOutput(ctx, lib+".jar")
}

func installPath(ctx android.ModuleInstallPathContext, lib string) android.InstallPath {
	return android.PathForModuleInstall(ctx, lib+".jar")
}
