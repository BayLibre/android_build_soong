// Copyright 2021 Google Inc. All rights reserved.
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
	"strings"
	"testing"

	"github.com/google/blueprint/proptools"

	"android/soong/android"
)

func TestBinary(t *testing.T) {
	ctx, _ := testJava(t, `
		java_library_host {
			name: "foo",
			srcs: ["a.java"],
		}

		java_binary_host {
			name: "bar",
			srcs: ["b.java"],
			static_libs: ["foo"],
			jni_libs: ["libjni"],
		}

		cc_library_shared {
			name: "libjni",
			host_supported: true,
			device_supported: false,
			stl: "none",
		}
	`)

	buildOS := android.BuildOs.String()

	bar := ctx.ModuleForTests("bar", buildOS+"_common")
	barJar := bar.Output("bar.jar").Output.String()
	barWrapper := ctx.ModuleForTests("bar", buildOS+"_x86_64")
	barWrapperDeps := barWrapper.Output("bar").Implicits.Strings()

	libjni := ctx.ModuleForTests("libjni", buildOS+"_x86_64_shared")
	libjniSO := libjni.Rule("Cp").Output.String()

	// Test that the install binary wrapper depends on the installed jar file
	if g, w := barWrapperDeps, barJar; !android.InList(w, g) {
		t.Errorf("expected binary wrapper implicits to contain %q, got %q", w, g)
	}

	// Test that the install binary wrapper depends on the installed JNI libraries
	if g, w := barWrapperDeps, libjniSO; !android.InList(w, g) {
		t.Errorf("expected binary wrapper implicits to contain %q, got %q", w, g)
	}
}

func TestHostBinaryNoJavaDebugInfoOverride(t *testing.T) {
	bp := `
		java_library {
			name: "target_library",
			srcs: ["a.java"],
		}

		java_binary_host {
			name: "host_binary",
			srcs: ["b.java"],
		}
	`
	config := testConfig(nil, bp, nil)
	config.TestProductVariables.MinimizeJavaDebugInfo = proptools.BoolPtr(true)

	ctx, _ := testJavaWithConfig(t, config)

	// first, check that the -g flag is added to target modules
	targetLibrary := ctx.ModuleForTests("target_library", "android_common")
	targetJavaFlags := targetLibrary.Module().VariablesForTests()["javacFlags"]
	if !strings.Contains(targetJavaFlags, "-g:source,lines") {
		t.Errorf("target library javac flags %v should contain "+
			"-g:source,lines override with MinimizeJavaDebugInfo", targetJavaFlags)
	}

	// check that -g is not overridden for host modules
	buildOS := android.BuildOs.String()
	hostBinary := ctx.ModuleForTests("host_binary", buildOS+"_common")
	hostJavaFlags := hostBinary.Module().VariablesForTests()["javacFlags"]
	if strings.Contains(hostJavaFlags, "-g:source,lines") {
		t.Errorf("java_binary_host javac flags %v should not have "+
			"-g:source,lines override with MinimizeJavaDebugInfo", hostJavaFlags)
	}
}
