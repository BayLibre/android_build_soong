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

package bp2build

import (
	"android/soong/android"
	"android/soong/cc"
	"testing"
)

func runSoongConfigModuleTypeTest(t *testing.T, tc bp2buildTestCase) {
	t.Helper()
	runBp2BuildTestCase(t, registerSoongConfigModuleTypes, tc)
}

func registerSoongConfigModuleTypes(ctx android.RegistrationContext) {
	cc.RegisterCCBuildComponents(ctx)

	ctx.RegisterModuleType("soong_config_module_type_import", android.SoongConfigModuleTypeImportFactory)
	ctx.RegisterModuleType("soong_config_module_type", android.SoongConfigModuleTypeFactory)
	ctx.RegisterModuleType("soong_config_string_variable", android.SoongConfigStringVariableDummyFactory)
	ctx.RegisterModuleType("soong_config_bool_variable", android.SoongConfigBoolVariableDummyFactory)
}

func TestSoongConfigModuleType_CcLibraryStatic(t *testing.T) {
	bp := `
soong_config_bool_variable {
	name: "feature1",
}

soong_config_bool_variable {
	name: "feature2",
}

soong_config_string_variable {
	name: "board",
	values: ["soc_a", "soc_b", "soc_c"],
}

soong_config_module_type {
	name: "custom_cc_library_static",
	module_type: "cc_library_static",
	config_namespace: "acme",
	variables: ["feature1", "feature2", "board"],
	properties: ["cflags"],
    bazel_module: { bp2build_available: true },
}

custom_cc_library_static {
	name: "foo",
    bazel_module: { bp2build_available: true },
	soong_config_variables: {
		feature1: {
			conditions_default: {
				cflags: ["-DDEFAULT1"],
			},
			cflags: ["-DFEATURE1"],
		},
		feature2: {
			cflags: ["-DFEATURE2"],
			conditions_default: {
				cflags: ["-DDEFAULT2"],
			},
		},
		board: {
			soc_a: {
				cflags: ["-DSOC_A"],
			},
			soc_b: {
				cflags: ["-SOC_B"],
			},
			soc_c: {},
			conditions_default: {
				cflags: ["-DSOC_DEFAULT"]
			},
		},
	},
}
`

	// TODO(b/198556411): foo_library should generate selects for custom config vars.
	runSoongConfigModuleTypeTest(t, bp2buildTestCase{
		description:                        "soong config variables - wraps cc_library_static",
		moduleTypeUnderTest:                "cc_library_static",
		moduleTypeUnderTestFactory:         cc.LibraryStaticFactory,
		moduleTypeUnderTestBp2BuildMutator: cc.CcLibraryStaticBp2Build,
		blueprint:                          bp,
		expectedBazelTargets: []string{`cc_library_static(
    name = "foo",
    copts = select({
        "//build/bazel/product_variables/vendor:acme__board__soc_a": ["-DSOC_A"],
        "//build/bazel/product_variables/vendor:acme__board__soc_b": ["-SOC_B"],
        "//conditions:default": ["-DSOC_DEFAULT"],
    }) + select({
        "//build/bazel/product_variables/vendor:acme__feature1__feature1": ["-DFEATURE1"],
        "//conditions:default": ["-DDEFAULT1"],
    }) + select({
        "//build/bazel/product_variables/vendor:acme__feature2__feature2": ["-DFEATURE2"],
        "//conditions:default": ["-DDEFAULT2"],
    }),
    local_includes = ["."],
)`}})
}
