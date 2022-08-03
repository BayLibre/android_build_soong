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

func registerLicenseModuleTypes(_ android.RegistrationContext) {}

type expectedBazelRule struct {
	name  string
	attrs attrNameToString
}

var tests = []struct {
	description string
	module      string
	expected    expectedBazelRule
}{
	{
		description: "license kind and text notice",
		module: `
license {
    name: "my_license",
    license_kinds: [ "SPDX-license-identifier-Apache-2.0"],
    license_text: [ "NOTICE"],
}`,
		expected: expectedBazelRule{
			"my_license",
			attrNameToString{
				"license_kinds": `["SPDX-license-identifier-Apache-2.0"]`,
				"license_text":  `"NOTICE"`,
			},
		},
	},
	{
		description: "visibility, package_name, copyright_notice",
		module: `
license {	
	name: "my_license", 
	package_name: "my_package", 
	visibility: ["__subpackage__"],
	copyright_notice: "Copyright © 2022",
}`,
		expected: expectedBazelRule{
			"my_license",
			attrNameToString{
				"copyright_notice": `"Copyright © 2022"`,
				"package_name":     `"my_package"`,
				"visibility":       `["__subpackage__"]`,
			},
		},
	},
}

func TestLicense1(t *testing.T) {
	for _, test := range tests {
		runBp2BuildTestCase(t,
			registerLicenseModuleTypes,
			bp2buildTestCase{
				description:                test.description,
				moduleTypeUnderTest:        "license",
				moduleTypeUnderTestFactory: android.LicenseFactory,
				blueprint:                  test.module,
				expectedBazelTargets: []string{
					makeBazelTargetNoRestrictions("license", test.expected.name, test.expected.attrs),
				},
			})
	}
}
