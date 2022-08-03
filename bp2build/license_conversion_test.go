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

func TestLicense(t *testing.T) {
	runBp2BuildTestCase(
		t,
		registerLicenseModuleTypes,
		bp2buildTestCase{
			description:                "license -- license kind and text notice",
			moduleTypeUnderTest:        "license",
			moduleTypeUnderTestFactory: android.LicenseFactory,
			blueprint: `
license {
    name: "my_license",
    license_kinds: [
        "SPDX-license-identifier-Apache-2.0",
    ],
    license_text: [
        "NOTICE",
    ],
}
`,
			expectedBazelTargets: []string{
				makeBazelTargetNoRestrictions("license", "my_license", attrNameToString{
					"license_kinds": `["SPDX-license-identifier-Apache-2.0"]`,
					"license_text":  `"NOTICE"`,
				}),
			},
		},
	)
}
