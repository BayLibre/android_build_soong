package etc

import (
	"android/soong/android"
	"strings"
	"testing"
)

var prepareForInstallSymlinkTest = android.GroupFixturePreparers(
	android.PrepareForTestWithArchMutator,
	android.FixtureRegisterWithContext(RegisterInstallSymlinkBuildComponents),
)

func TestInstallSymlinkBasic(t *testing.T) {
	result := prepareForInstallSymlinkTest.RunTestWithBp(t, `
		install_symlink {
			name: "foo",
			installed_location: "bin/foo",
			symlink_target: "/system/system_ext/bin/foo",
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

func TestInstallSymlinkToRecovery(t *testing.T) {
	result := prepareForInstallSymlinkTest.RunTestWithBp(t, `
		install_symlink {
			name: "foo",
			installed_location: "bin/foo",
			symlink_target: "/system/system_ext/bin/foo",
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
