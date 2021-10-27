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
	"android/soong/java"

	"testing"
)

func runAndroidAppCertificateTestCase(t *testing.T, tc Bp2BuildTestCase) {
	t.Helper()
	RunBp2BuildTestCase(t, registerAndroidAppCertificateModuleTypes, tc)
}

func registerAndroidAppCertificateModuleTypes(ctx android.RegistrationContext) {
}

func TestAndroidAppCertificateSimple(t *testing.T) {
	runAndroidAppCertificateTestCase(t, Bp2BuildTestCase{
		Description:                        "Android app certificate - simple example",
		ModuleTypeUnderTest:                "android_app_certificate",
		ModuleTypeUnderTestFactory:         java.AndroidAppCertificateFactory,
		ModuleTypeUnderTestBp2BuildMutator: java.AndroidAppCertificateBp2Build,
		Filesystem:                         map[string]string{},
		Blueprint: `
android_app_certificate {
        name: "com.android.apogee.cert",
        certificate: "chamber_of_secrets_dir",
}
`,
		ExpectedBazelTargets: []string{`android_app_certificate(
    name = "com.android.apogee.cert",
    certificate = "chamber_of_secrets_dir",
)`}})
}
