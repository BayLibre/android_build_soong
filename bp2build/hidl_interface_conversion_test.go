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

package bp2build

import (
	"testing"

	"android/soong/android"
	"android/soong/cc"
	"android/soong/hidl"
)

func runHidlInterfaceTestCase(t *testing.T, tc bp2buildTestCase) {
	t.Helper()
	runBp2BuildTestCase(
		t,
		func(ctx android.RegistrationContext) {
			ctx.RegisterModuleType("cc_defaults", func() android.Module { return cc.DefaultsFactory() })
			ctx.RegisterModuleType("hidl_interface", hidl.HidlInterfaceFactory)
			ctx.RegisterModuleType("hidl_package_root", hidl.HidlPackageRootFactory)
		},
		tc,
	)
}

func TestHidlInterface(t *testing.T) {
	runHidlInterfaceTestCase(t, bp2buildTestCase{
		description: `hidl_interface with common usage of properties`,
		blueprint: `
hidl_package_root {
		name: "android.hardware",
		use_current: true,
}
cc_defaults {
		name: "hidl-module-defaults",
}
hidl_interface {
		name: "android.hardware.nfc@1.0",
		srcs: ["types.hal", "IBase.hal"],
		root: "android.hardware",
		gen_java: false,
}
hidl_interface {
		name: "android.hardware.nfc@1.1",
		srcs: ["types.hal", "INfc.hal"],
		interfaces: ["android.hardware.nfc@1.0"],
		root: "android.hardware",
		gen_java: false,
}`,
		expectedBazelTargets: []string{
			makeBazelTargetNoRestrictions("hidl_interface", "android.hardware.nfc@1.0", attrNameToString{
				"min_sdk_version":     `"29"`,
				"root":                `"android.hardware"`,
				"root_interface_file": `":current.txt"`,
				"srcs": `[
        "types.hal",
        "IBase.hal",
    ]`,
			}),
			makeBazelTargetNoRestrictions("hidl_interface", "android.hardware.nfc@1.1", attrNameToString{
				"deps":                `[":android.hardware.nfc@1.0"]`,
				"min_sdk_version":     `"29"`,
				"root":                `"android.hardware"`,
				"root_interface_file": `":current.txt"`,
				"srcs": `[
        "types.hal",
        "INfc.hal",
    ]`,
			}),
		},
	})
}
