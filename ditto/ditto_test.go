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

package ditto

import (
	"os"
	"testing"

	"android/soong/android"
	"android/soong/cc"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

var prepareForDittoTest = android.GroupFixturePreparers(
	cc.PrepareForTestWithCcDefaultModules,
	android.FixtureMergeMockFs(
		map[string][]byte{
			"ditto.c":              nil,
			"ditto_invalid_name.c": nil,
			"DittoTest.cpp":        nil,
		},
	),
	PrepareForTestWithDitto,
)

func TestDittoDataDependency(t *testing.T) {
	bp := `
		ditto {
			name: "ditto.o",
			srcs: ["ditto.c"],
		}

		cc_test {
			name: "vts_test_binary_ditto_module",
			srcs: ["DittoTest.cpp"],
			data: [":ditto.o"],
			gtest: false,
		}
	`

	prepareForDittoTest.RunTestWithBp(t, bp)

	// We only verify the above BP configuration is processed successfully since the data property
	// value is not available for testing from this package.
	// TODO(jungjw): Add a check for data or move this test to the cc package.
}

func TestDittoSourceName(t *testing.T) {
	bp := `
		ditto {
			name: "ditto_invalid_name.o",
			srcs: ["ditto_invalid_name.c"],
		}
	`
	prepareForDittoTest.ExtendWithErrorHandler(android.FixtureExpectsOneErrorPattern(
		`\QAndroid.bp:2:3: module "ditto_invalid_name.o" variant "android_common": invalid character '_' in source name\E`)).
		RunTestWithBp(t, bp)
}

func TestDittoWithBazel(t *testing.T) {
	bp := `
		ditto {
			name: "ditto.o",
			srcs: ["ditto.c"],
			bazel_module: { label: "//ditto" },
		}
	`

	result := android.GroupFixturePreparers(
		prepareForDittoTest, android.FixtureModifyConfig(func(config android.Config) {
			config.BazelContext = android.MockBazelContext{
				OutputBaseDir: "outputbase",
				LabelToOutputFiles: map[string][]string{
					"//ditto": []string{"ditto.o"}}}
		})).RunTestWithBp(t, bp)

	output := result.Module("ditto.o", "android_common").(*ditto)

	expectedOutputFiles := []string{"outputbase/execroot/__main__/ditto.o"}
	android.AssertDeepEquals(t, "output files", expectedOutputFiles, output.objs.Strings())
}
