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
	"testing"
)

func runPackageTestCase(t *testing.T, tc bp2buildTestCase) {
	t.Helper()
	runBp2BuildTestCase(t, registerDependentModules, tc)
}

func registerDependentModules(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("license", android.LicenseFactory)
}

func TestPackage(t *testing.T) {
	tests := []struct {
		description string
		modules     string
		expected    []expectedBazelRule
	}{
		{
			description: "with default applicable licenses",
			modules: `
license {
  name: "my_license",
  visibility: [":__subpackages__"],
    license_kinds: ["SPDX-license-identifier-Apache-2.0"],
    license_text: ["NOTICE"],
}

package {
  default_applicable_licenses: ["my_license"],
}
`,
			expected: []expectedBazelRule{
				{
					"package",
					"",
					attrNameToString{
						"default_applicable_licenses": `["my_license"]`,
					},
					android.HostAndDeviceDefault,
				},
				{
					"license",
					"my_license",
					attrNameToString{
						"license_kinds": `["SPDX-license-identifier-Apache-2.0"]`,
						"license_text":  `"NOTICE"`,
						"visibility":    `[":__subpackages__"]`,
					},
					android.HostAndDeviceDefault,
				},
			},
		},
	}
	for _, test := range tests {
		expected := make([]string, 0, len(test.expected))
		for _, e := range test.expected {
			expected = append(expected, e.String())
		}
		runPackageTestCase(t,
			bp2buildTestCase{
				description:                test.description,
				moduleTypeUnderTest:        "package",
				moduleTypeUnderTestFactory: android.PackageFactory,
				blueprint:                  test.modules,
				expectedBazelTargets:       expected,
			})
	}
}
