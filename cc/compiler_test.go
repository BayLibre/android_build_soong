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

package cc

import (
	"slices"
	"strings"
	"testing"

	"android/soong/android"
)

func TestIsThirdParty(t *testing.T) {
	thirdPartyPaths := []string{
		"external/foo/",
		"vendor/bar/",
		"hardware/underwater_jaguar/",
	}
	nonThirdPartyPaths := []string{
		"vendor/google/cts/",
		"hardware/google/pixel",
		"hardware/interfaces/camera",
		"hardware/ril/supa_ril",
		"bionic/libc",
	}
	for _, path := range thirdPartyPaths {
		if !android.IsThirdPartyPath(path) {
			t.Errorf("Expected %s to be considered third party", path)
		}
	}
	for _, path := range nonThirdPartyPaths {
		if android.IsThirdPartyPath(path) {
			t.Errorf("Expected %s to *not* be considered third party", path)
		}
	}
}

func TestReplaceNoErrorWithWarning(t *testing.T) {
	flags := []string {
		"-fPIC",
		"-Wno-error=warn1",
		"-Wno-warn2",
	}
	expectedFlags := []string {
		"-fPIC",
		"-Wno-warn1",
		"-Wno-warn2",
	}
	if replaceNoErrorWithWarning(flags); !slices.Equal(flags, expectedFlags) {
		t.Errorf("Expected flags: %v but got: %v", expectedFlags, flags)
	}
}

func TestReplaceNoErrorForThirdPartyPaths(t *testing.T) {
	t.Parallel()
	device_bp := `
	cc_library_shared {
		name: "libDeviceTest",
		srcs: ["test.c"],
		cflags: [
			"-fPIC",
			"-Wno-error=warn1",
			"-Wno-warn2",
		],
	}
	`
	external_bp := `
	cc_library_shared {
		name: "libExternalTest",
		srcs: ["test.c"],
		cflags: [
			"-fPIC",
			"-Wno-error=warn3",
			"-Wno-warn4",
		],
	}
	`

	result := android.GroupFixturePreparers(
		prepareForCcTest,
		android.FixtureAddTextFile("device/libnoerror/Android.bp", device_bp),
		android.FixtureAddTextFile("external/libnoerror/Android.bp", external_bp),
	).RunTest(t)

	expectedDeviceCFlags := []string{
		"-fPIC",
		"-Wno-error=warn1",
		"-Wno-warn2",
	}
	expectedExternalCFlags := []string{
		"-fPIC",
		"-Wno-warn3",
		"-Wno-warn4",
	}

	libDeviceTest := result.ModuleForTests("libDeviceTest", coreVariant)
	libExternalTest := result.ModuleForTests("libExternalTest", coreVariant)

	// Verify that modules not under //external preserve -Wno-error=foo.
	deviceCFlags := libDeviceTest.Rule("cc").Args["cFlags"]
	for _, flag := range expectedDeviceCFlags {
		if !strings.Contains(deviceCFlags, flag) {
			t.Errorf("Expected flag %q but did not find in cflags %q", flag, deviceCFlags)
		}
	}

	// Verify that modules under //external replace -Wno-error=foo with -Wno-foo.
	externalCFlags := libExternalTest.Rule("cc").Args["cFlags"]
	for _, flag := range expectedExternalCFlags {
		if !strings.Contains(externalCFlags, flag) {
			t.Errorf("Expected flag %q but did not find in cflags %q", flag, externalCFlags)
		}
	}
}

