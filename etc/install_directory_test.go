// Copyright 2024 Google Inc. All rights reserved.
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

package etc

import (
	"android/soong/android"
	"strings"
	"testing"
)

var prepareForInstallDirectoryTest = android.GroupFixturePreparers(
	android.PrepareForTestWithArchMutator,
	android.FixtureRegisterWithContext(RegisterInstallDirectoryBuildComponents),
)

func TestInstalldirectoryBasic(t *testing.T) {
	result := prepareForInstallDirectoryTest.RunTestWithBp(t, `
		install_directory {
			name: "foo",
			installed_location: "bin/foo",
		}
	`)

	foo_variants := result.ModuleVariantsForTests("foo")
	if len(foo_variants) != 1 {
		t.Fatalf("expected 1 variant, got %#v", foo_variants)
	}

	foo := result.ModuleForTests("foo", "android_common").Module()
	androidMkEntries := android.AndroidMkEntriesForTest(t, result.TestContext, foo)
	if len(androidMkEntries) != 1 {
		t.Fatalf("expected 1 androidmkentry, got %d", len(androidMkEntries))
	}

	symlinks := androidMkEntries[0].EntryMap["LOCAL_SOONG_INSTALL_SYMLINKS"]
	if len(symlinks) != 1 {
		t.Fatalf("Expected 1 symlink, got %d", len(symlinks))
	}

	if !strings.HasSuffix(symlinks[0], "system/bin/foo") {
		t.Fatalf("Expected symlink install path to end in system/bin/foo, got: %s", symlinks[0])
	}
}

func TestInstallDirectoryToRecovery(t *testing.T) {
	result := prepareForInstallDirectoryTest.RunTestWithBp(t, `
		install_directory {
			name: "foo",
			installed_location: "bin/foo",
			recovery: true,
		}
	`)

	foo_variants := result.ModuleVariantsForTests("foo")
	if len(foo_variants) != 1 {
		t.Fatalf("expected 1 variant, got %#v", foo_variants)
	}

	foo := result.ModuleForTests("foo", "android_common").Module()
	androidMkEntries := android.AndroidMkEntriesForTest(t, result.TestContext, foo)
	if len(androidMkEntries) != 1 {
		t.Fatalf("expected 1 androidmkentry, got %d", len(androidMkEntries))
	}

	symlinks := androidMkEntries[0].EntryMap["LOCAL_SOONG_INSTALL_SYMLINKS"]
	if len(symlinks) != 1 {
		t.Fatalf("Expected 1 symlink, got %d", len(symlinks))
	}

	if !strings.HasSuffix(symlinks[0], "recovery/root/system/bin/foo") {
		t.Fatalf("Expected symlink install path to end in recovery/root/system/bin/foo, got: %s", symlinks[0])
	}
}

func TestErrorOnNonCleanInstalledLocationForInstallDirectory(t *testing.T) {
	prepareForInstallDirectoryTest.
		ExtendWithErrorHandler(android.FixtureExpectsOneErrorPattern("Should be a clean filepath")).
		RunTestWithBp(t, `
		install_directory {
			name: "foo",
			installed_location: "bin/../foo",
		}
	`)
}
