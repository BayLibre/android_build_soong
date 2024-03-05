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

package apex

import (
	"testing"

	"android/soong/aconfig/codegen"
	"android/soong/android"
	"android/soong/cc"
	"android/soong/java"
	"android/soong/rust"
)

var withAconfigValidationError = android.FixtureModifyProductVariables(func(variables android.FixtureProductVariables) {
	variables.AconfigContainerValidation = "error"
})

func TestValidationAcrossContainersExportedPass(t *testing.T) {
	testCases := []struct {
		name string
		bp   string
	}{
		{
			name: "Java lib passes for exported containers cross",
			bp: apex_default_bp + `
				apex {
					name: "myapex",
					manifest: ":myapex.manifest",
					androidManifest: ":myapex.androidmanifest",
					key: "myapex.key",
					java_libs: [
						"my_java_library_foo",
					],
					updatable: false,
				}
		
				java_library {
					name: "my_java_library_foo",
					srcs: ["foo/bar/MyClass.java"],
					sdk_version: "none",
					system_modules: "none",
					static_libs: ["my_java_aconfig_library_foo"],
					apex_available: [
						"myapex",
					],
				}
		
				aconfig_declarations {
					name: "my_aconfig_declarations_foo",
					package: "com.example.package",
					container: "otherapex",
					srcs: ["foo.aconfig"],
				}
		
				java_aconfig_library {
					name: "my_java_aconfig_library_foo",
					aconfig_declarations: "my_aconfig_declarations_foo",
					mode: "exported",
					apex_available: [
						"myapex",
					],
				}`,
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			android.GroupFixturePreparers(
				java.PrepareForTestWithJavaDefaultModules,
				cc.PrepareForTestWithCcBuildComponents,
				rust.PrepareForTestWithRustDefaultModules,
				codegen.PrepareForTestWithAconfigBuildComponents,
				PrepareForTestWithApexBuildComponents,
				prepareForTestWithMyapex,
				withAconfigValidationError,
			).
				RunTestWithBp(t, test.bp)
		})
	}
}

func TestValidationAcrossContainersNotExportedFail(t *testing.T) {
	testCases := []struct {
		name          string
		expectedError string
		bp            string
	}{
		{
			name: "Java lib fails for non-exported containers cross",
			bp: apex_default_bp + `
				apex {
					name: "myapex",
					manifest: ":myapex.manifest",
					androidManifest: ":myapex.androidmanifest",
					key: "myapex.key",
					java_libs: [
						"my_java_library_foo",
					],
					updatable: false,
				}
		
				java_library {
					name: "my_java_library_foo",
					srcs: ["foo/bar/MyClass.java"],
					sdk_version: "none",
					system_modules: "none",
					static_libs: ["my_java_aconfig_library_foo"],
					apex_available: [
						"myapex",
					],
				}
		
				aconfig_declarations {
					name: "my_aconfig_declarations_foo",
					package: "com.example.package",
					container: "otherapex",
					srcs: ["foo.aconfig"],
				}
		
				java_aconfig_library {
					name: "my_java_aconfig_library_foo",
					aconfig_declarations: "my_aconfig_declarations_foo",
					apex_available: [
						"myapex",
					],
				}`,
			expectedError: `.*my_java_library_foo/myapex depends on my_java_aconfig_library_foo/otherapex/production across containers`,
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			errorHandler := android.FixtureExpectsNoErrors
			if test.expectedError != "" {
				errorHandler = android.FixtureExpectsAtLeastOneErrorMatchingPattern(test.expectedError)
			}
			android.GroupFixturePreparers(
				java.PrepareForTestWithJavaDefaultModules,
				cc.PrepareForTestWithCcBuildComponents,
				rust.PrepareForTestWithRustDefaultModules,
				codegen.PrepareForTestWithAconfigBuildComponents,
				PrepareForTestWithApexBuildComponents,
				prepareForTestWithMyapex,
				withAconfigValidationError,
			).
				ExtendWithErrorHandler(errorHandler).
				RunTestWithBp(t, test.bp)
		})
	}
}
