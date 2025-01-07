// Copyright 2018 Google Inc. All rights reserved.
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
	"fmt"
	"testing"

	"android/soong/android"
)

func AssertJarJarRename(t *testing.T, result *android.TestResult, libName, original, expectedRename string) {
	module := result.ModuleForTests(libName, "android_common")

	provider, found := android.OtherModuleProvider(result.OtherModuleProviderAdaptor(), module.Module(), JarJarProvider)
	android.AssertBoolEquals(t, fmt.Sprintf("found provider (%s)", libName), true, found)

	rename := !android.InList(original, provider.NonRenamedClasses)
	android.AssertBoolEquals(t, fmt.Sprintf("%s has been renamed", libName), original != expectedRename, rename)
}

func TestJarJarRenameDifferentModules(t *testing.T) {
	t.Parallel()
	result := android.GroupFixturePreparers(
		prepareForJavaTest,
	).RunTestWithBp(t, `
		java_library {
			name: "their_lib",
			jarjar_rename: ["com.example.a"],
		}

		java_library {
			name: "boundary_lib",
			jarjar_prefix: "RENAME",
			static_libs: ["their_lib"],
		}

		java_library {
			name: "my_lib",
			static_libs: ["boundary_lib"],
		}
	`)

	original := "com.example.a"
	renamed := "RENAME.com.example.a"
	AssertJarJarRename(t, result, "their_lib", original, original)
	AssertJarJarRename(t, result, "boundary_lib", original, renamed)
	AssertJarJarRename(t, result, "my_lib", original, renamed)
}

func TestJarJarRenameSameModule(t *testing.T) {
	t.Parallel()
	result := android.GroupFixturePreparers(
		prepareForJavaTest,
	).RunTestWithBp(t, `
		java_library {
			name: "their_lib",
			jarjar_rename: ["com.example.a"],
			jarjar_prefix: "RENAME",
		}

		java_library {
			name: "my_lib",
			static_libs: ["their_lib"],
		}
	`)

	original := "com.example.a"
	renamed := "RENAME.com.example.a"
	AssertJarJarRename(t, result, "their_lib", original, renamed)
	AssertJarJarRename(t, result, "my_lib", original, renamed)
}

func TestJarJarRenameFile(t *testing.T) {
	t.Parallel()
	result := android.GroupFixturePreparers(
		prepareForJavaTest,
		android.FixtureMergeMockFs(android.MockFS{
			"their_lib/rename_classes.txt": nil,
		}),
	).RunTestWithBp(t, `
		java_library {
			name: "their_lib",
			jarjar_rename_file: "their_lib/rename_classes.txt",
		}
		java_library {
			name: "my_lib",
			jarjar_prefix: "RENAME",
			static_libs: ["their_lib"],
		}
	`)

	myLib := result.ModuleForTests("my_lib", "android_common")
	myLibRuleTextFileCommand := myLib.Output("repackaged-jarjar/repackaging.txt").RuleParams.Command
	android.AssertStringDoesContain(
		t,
		"jarjar rule command expected to contain dependencies' jarjar_rename_file",
		myLibRuleTextFileCommand,
		"--classnames-files their_lib/rename_classes.txt",
	)
}
