// Copyright 2022 Google Inc. All rights reserved.
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

package cc

import (
	"strings"
	"testing"

	"android/soong/android"

	"github.com/google/blueprint"
)

func TestAfdoDeps(t *testing.T) {
	t.Parallel()
	bp := `
	cc_library_shared {
		name: "libTest",
		srcs: ["foo.c"],
		static_libs: ["libFoo"],
		afdo: true,
	}

	cc_library_static {
		name: "libFoo",
		static_libs: ["libBar"],
	}

	cc_library_static {
		name: "libBar",
	}
	`
	prepareForAfdoTest := android.FixtureAddTextFile("toolchain/pgo-profiles/sampling/libTest.afdo", "TEST")

	result := android.GroupFixturePreparers(
		prepareForCcTest,
		prepareForAfdoTest,
	).RunTestWithBp(t, bp)

	libTest := result.ModuleForTests("libTest", "android_arm64_armv8-a_shared")
	libFoo := result.ModuleForTests("libFoo", "android_arm64_armv8-a_static_afdo-libTest").Module()
	libBar := result.ModuleForTests("libBar", "android_arm64_armv8-a_static_afdo-libTest").Module()

	hasDep := func(m android.Module, wantDep android.Module) bool {
		var found bool
		result.VisitDirectDeps(m, func(dep blueprint.Module) {
			if dep == wantDep {
				found = true
			}
		})
		return found
	}

	if !hasDep(libTest.Module(), libFoo) {
		t.Errorf("libTest missing dependency on afdo variant of libFoo")
	}

	if !hasDep(libFoo, libBar) {
		t.Errorf("libTest missing dependency on afdo variant of libBar")
	}

	cFlags := libTest.Rule("cc").Args["cFlags"]
	if w := "-fprofile-sample-accurate"; !strings.Contains(cFlags, w) {
		t.Errorf("Expected 'libTest' to enable afdo, but did not find %q in cflags %q", w, cFlags)
	}
}

func TestAfdoEnabledOnStaticDep(t *testing.T) {
	t.Parallel()
	bp := `
	cc_library_shared {
		name: "libTest",
		srcs: ["foo.c"],
		static_libs: ["libFoo"],
	}

	cc_library_static {
		name: "libFoo",
		srcs: ["foo.c"],
		static_libs: ["libBar"],
		afdo: true, // TODO(b/256670524): remove support for enabling afdo from static only libraries, this can only propagate from shared libraries/binaries
	}

	cc_library_static {
		name: "libBar",
	}
	`
	prepareForAfdoTest := android.FixtureAddTextFile("toolchain/pgo-profiles/sampling/libFoo.afdo", "TEST")

	result := android.GroupFixturePreparers(
		prepareForCcTest,
		prepareForAfdoTest,
	).RunTestWithBp(t, bp)

	libTest := result.ModuleForTests("libTest", "android_arm64_armv8-a_shared").Module()
	libFoo := result.ModuleForTests("libFoo", "android_arm64_armv8-a_static")
	libBar := result.ModuleForTests("libBar", "android_arm64_armv8-a_static").Module()

	hasDep := func(m android.Module, wantDep android.Module) bool {
		var found bool
		result.VisitDirectDeps(m, func(dep blueprint.Module) {
			if dep == wantDep {
				found = true
			}
		})
		return found
	}

	if !hasDep(libTest, libFoo.Module()) {
		t.Errorf("libTest missing dependency on afdo variant of libFoo")
	}

	if !hasDep(libFoo.Module(), libBar) {
		t.Errorf("libFoo missing dependency on afdo variant of libBar")
	}

	fooVariants := result.ModuleVariantsForTests("foo")
	for _, v := range fooVariants {
		if strings.Contains(v, "afdo-") {
			t.Errorf("Expected no afdo variant of 'foo', got %q", v)
		}
	}

	cFlags := libFoo.Rule("cc").Args["cFlags"]
	if w := "-fprofile-sample-accurate"; strings.Contains(cFlags, w) {
		t.Errorf("Expected 'foo' to not enable afdo, but found %q in cflags %q", w, cFlags)
	}
}
