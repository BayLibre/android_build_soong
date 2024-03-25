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

package android

import (
	"encoding/json"
	"testing"
)

var buildFlagsFixture = GroupFixturePreparers(
	PrepareForIntegrationTestWithAndroid,
	PrepareForTestWithReleaseFlags,
)

var prepareForPackagingTest = GroupFixturePreparers(
	buildFlagsFixture,
	FixtureModifyProductVariables(func(variables FixtureProductVariables) {
		variables.BuildFlags = map[string]BuildFlagData{
			"RELEASE_BOARD_API_LEVEL": BuildFlagData{"RELEASE_BOARD_API_LEVEL",
				"202404",
				"a.scl",
				"",
				"d.scl",
			},
			"RELEASE_PLATFORM_SDK_VERSION": BuildFlagData{"RELEASE_PLATFORM_SDK_VERSION",
				"35",
				"b.scl",
				"34",
				"e.scl",
			},
			"RELEASE_TEST_VERSION": BuildFlagData{"RELEASE_TEST_VERSION",
				"34",
				"c.scl",
				"30",
				"f.scl",
			},
		}
		variables.BuildFlagsPartitions = map[string][]string{
			"system":     []string{"RELEASE_PLATFORM_SDK_VERSION", "RELEASE_TEST_VERSION"},
			"system_ext": []string{"RELEASE_TEST_VERSION"},
			"product":    []string{"RELEASE_TEST_VERSION"},
			"vendor":     []string{"RELEASE_BOARD_API_LEVEL", "RELEASE_TEST_VERSION"},
		}
	}),
)

func checkJsonHas(t *testing.T, flags []BuildFlagData, partition, name, val, set, def, dcl string) {
	for _, flag := range flags {
		if flag.Name == name {
			if flag.Value == val && flag.Set == set && flag.Default == def && flag.Declared == dcl {
				return
			}
			t.Errorf("%s has %q, but wrong values", partition, name)
		}
	}
	t.Errorf("%s must have %q", partition, name)
}

func checkJsonNotHave(t *testing.T, flags []BuildFlagData, partition, name string) {
	for _, flag := range flags {
		if flag.Name == name {
			t.Errorf("%s must not have %q", partition, name)
		}
	}
}

func getJsonBytes(t *testing.T, ctx *TestContext, moduleName string) releaseFlagsJson {
	mod := ctx.ModuleForTests(moduleName, "android_common").Module().(*releaseFlags)
	var jsonData releaseFlagsJson
	err := json.Unmarshal([]byte(mod.jsonString), &jsonData)
	if err != nil {
		t.Errorf("Failed to unmarshal json output")
	}
	return jsonData
}

func TestPackagingFlagsJsonGeneration(t *testing.T) {
	t.Parallel()
	bp := `
		release_flags_json {
			name: "build_flag_system",
		}
		release_flags_json {
			name: "build_flag_system_ext",
			system_ext_specific: true,
		}
		release_flags_json {
			name: "build_flag_product",
			product_specific: true,
		}
		release_flags_json {
			name: "build_flag_vendor",
			vendor: true,
		}
		release_flags_json {
			name: "build_flag_odm",
			device_specific: true,
		}
	`

	ctx := prepareForPackagingTest.RunTestWithBp(t, bp).TestContext

	systemJsonData := getJsonBytes(t, ctx, "build_flag_system")
	checkJsonNotHave(t, systemJsonData.Flags, "system", "RELEASE_BOARD_API_LEVEL")
	checkJsonHas(t, systemJsonData.Flags, "system", "RELEASE_PLATFORM_SDK_VERSION", "35", "b.scl", "34", "e.scl")
	checkJsonHas(t, systemJsonData.Flags, "system", "RELEASE_TEST_VERSION", "34", "c.scl", "30", "f.scl")

	systemExtJsonData := getJsonBytes(t, ctx, "build_flag_system_ext")
	checkJsonNotHave(t, systemExtJsonData.Flags, "system_ext", "RELEASE_BOARD_API_LEVEL")
	checkJsonNotHave(t, systemExtJsonData.Flags, "system_ext", "RELEASE_PLATFORM_SDK_VERSION")
	checkJsonHas(t, systemExtJsonData.Flags, "system_ext", "RELEASE_TEST_VERSION", "34", "c.scl", "30", "f.scl")

	productJsonData := getJsonBytes(t, ctx, "build_flag_product")
	checkJsonNotHave(t, productJsonData.Flags, "product", "RELEASE_BOARD_API_LEVEL")
	checkJsonNotHave(t, productJsonData.Flags, "product", "RELEASE_PLATFORM_SDK_VERSION")
	checkJsonHas(t, productJsonData.Flags, "product", "RELEASE_TEST_VERSION", "34", "c.scl", "30", "f.scl")

	vendorJsonData := getJsonBytes(t, ctx, "build_flag_vendor")
	checkJsonHas(t, vendorJsonData.Flags, "vendor", "RELEASE_BOARD_API_LEVEL", "202404", "a.scl", "", "d.scl")
	checkJsonNotHave(t, vendorJsonData.Flags, "vendor", "RELEASE_PLATFORM_SDK_VERSION")
	checkJsonHas(t, vendorJsonData.Flags, "vendor", "RELEASE_TEST_VERSION", "34", "c.scl", "30", "f.scl")

	odmJsonData := getJsonBytes(t, ctx, "build_flag_odm")
	checkJsonNotHave(t, odmJsonData.Flags, "odm", "RELEASE_BOARD_API_LEVEL")
	checkJsonNotHave(t, odmJsonData.Flags, "odm", "RELEASE_PLATFORM_SDK_VERSION")
	checkJsonNotHave(t, odmJsonData.Flags, "odm", "RELEASE_TEST_VERSION")
}
