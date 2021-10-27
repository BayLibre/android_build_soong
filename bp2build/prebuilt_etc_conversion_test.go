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
	"android/soong/etc"

	"testing"
)

func runPrebuiltEtcTestCase(t *testing.T, tc Bp2BuildTestCase) {
	t.Helper()
	RunBp2BuildTestCase(t, registerPrebuiltEtcModuleTypes, tc)
}

func registerPrebuiltEtcModuleTypes(ctx android.RegistrationContext) {
}

func TestPrebuiltEtcSimple(t *testing.T) {
	runPrebuiltEtcTestCase(t, Bp2BuildTestCase{
		Description:                        "prebuilt_etc - simple example",
		ModuleTypeUnderTest:                "prebuilt_etc",
		ModuleTypeUnderTestFactory:         etc.PrebuiltEtcFactory,
		ModuleTypeUnderTestBp2BuildMutator: etc.PrebuiltEtcBp2Build,
		Filesystem:                         map[string]string{},
		Blueprint: `
prebuilt_etc {
    name: "apex_tz_version",
    src: "version/tz_version",
    filename: "tz_version",
    sub_dir: "tz",
    installable: false,
}
`,
		ExpectedBazelTargets: []string{`prebuilt_etc(
    name = "apex_tz_version",
    filename = "tz_version",
    installable = False,
    src = "version/tz_version",
    sub_dir = "tz",
)`}})
}

func TestPrebuiltEtcArchVariant(t *testing.T) {
	runPrebuiltEtcTestCase(t, Bp2BuildTestCase{
		Description:                        "prebuilt_etc - simple example",
		ModuleTypeUnderTest:                "prebuilt_etc",
		ModuleTypeUnderTestFactory:         etc.PrebuiltEtcFactory,
		ModuleTypeUnderTestBp2BuildMutator: etc.PrebuiltEtcBp2Build,
		Filesystem:                         map[string]string{},
		Blueprint: `
prebuilt_etc {
    name: "apex_tz_version",
    src: "version/tz_version",
    filename: "tz_version",
    sub_dir: "tz",
    installable: false,
    arch: {
      arm: {
        src: "arm",
      },
      arm64: {
        src: "arm64",
      },
    }
}
`,
		ExpectedBazelTargets: []string{`prebuilt_etc(
    name = "apex_tz_version",
    filename = "tz_version",
    installable = False,
    src = select({
        "//build/bazel/platforms/arch:arm": "arm",
        "//build/bazel/platforms/arch:arm64": "arm64",
        "//conditions:default": "version/tz_version",
    }),
    sub_dir = "tz",
)`}})
}
