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
	"android/soong/apex"

	"testing"
)

func runApexKeyTestCase(t *testing.T, tc Bp2BuildTestCase) {
	t.Helper()
	RunBp2BuildTestCase(t, registerApexKeyModuleTypes, tc)
}

func registerApexKeyModuleTypes(ctx android.RegistrationContext) {
}

func TestApexKeySimple(t *testing.T) {
	runApexKeyTestCase(t, Bp2BuildTestCase{
		Description:                        "apex key - simple example",
		ModuleTypeUnderTest:                "apex_key",
		ModuleTypeUnderTestFactory:         apex.ApexKeyFactory,
		ModuleTypeUnderTestBp2BuildMutator: apex.ApexKeyBp2Build,
		Filesystem:                         map[string]string{},
		Blueprint: `
apex_key {
        name: "com.android.apogee.key",
        public_key: "com.android.apogee.avbpubkey",
        private_key: "com.android.apogee.pem",
}
`,
		ExpectedBazelTargets: []string{`apex_key(
    name = "com.android.apogee.key",
    private_key = "com.android.apogee.pem",
    public_key = "com.android.apogee.avbpubkey",
)`}})
}
