// Copyright 2023 Google Inc. All rights reserved.
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
	"fmt"
	"testing"

	"android/soong/apex"
)

func runPrebuiltApexTestCase(t *testing.T, tc Bp2buildTestCase) {
	t.Helper()
	(&tc).ModuleTypeUnderTest = "prebuilt_apex"
	(&tc).ModuleTypeUnderTestFactory = apex.PrebuiltFactory
	RunBp2BuildTestCase(t, registerPrebuiltModuleTypes, tc)
}

func TestPrebuiltApexSimple(t *testing.T) {
	runPrebuiltApexTestCase(t, Bp2buildTestCase{
		Description: "prebuilt_apex - simple example",
		Filesystem:  map[string]string{},
		Blueprint: `
prebuilt_apex {
    name: "apex_tz_version",
    src: "version/tz_version",
    filename: "tz_version",
    installable: false,
}
`,
		ExpectedBazelTargets: []string{
			MakeBazelTarget("prebuilt_file", "apex_tz_version", AttrNameToString{
				"filename":    `"tz_version"`,
				"installable": `False`,
				"src":         `"version/tz_version"`,
				"dir":         `""`,
			})}})
}

func TestPrebuiltApexArchVariant(t *testing.T) {
	runPrebuiltApexTestCase(t, Bp2buildTestCase{
		Description: "prebuilt_apex - arch variant",
		Filesystem:  map[string]string{},
		Blueprint: `
prebuilt_apex {
    name: "apex_tz_version",
    src: "version/tz_version",
    filename: "tz_version",
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
		ExpectedBazelTargets: []string{
			MakeBazelTarget("prebuilt_file", "apex_tz_version", AttrNameToString{
				"filename":    `"tz_version"`,
				"installable": `False`,
				"src": `select({
        "//build/bazel_common_rules/platforms/arch:arm": "arm",
        "//build/bazel_common_rules/platforms/arch:arm64": "arm64",
        "//conditions:default": "version/tz_version",
    })`,
				"dir": `""`,
			})}})
}

func TestPrebuiltApexProductVariables(t *testing.T) {
	runPrebuiltApexTestCase(t, Bp2buildTestCase{
		Description: "prebuilt etc - product variables",
		Filesystem:  map[string]string{},
		Blueprint: `
prebuilt_apex {
    name: "apex_tz_version",
    src: "version/tz_version",
    filename: "tz_version",
    product_variables: {
      native_coverage: {
        src: "src1",
      },
    },
}
`,
		ExpectedBazelTargets: []string{
			MakeBazelTarget("prebuilt_file", "apex_tz_version", AttrNameToString{
				"filename": `"tz_version"`,
				"src": `select({
        "//build/bazel/product_config/config_settings:native_coverage": "src1",
        "//conditions:default": "version/tz_version",
    })`,
				"dir": `""`,
			})}})
}

func TestPrebuiltApexNoSubdir(t *testing.T) {
	runPrebuiltApexTestCase(t, Bp2buildTestCase{
		Description: "prebuilt_apex - no subdir",
		Filesystem:  map[string]string{},
		Blueprint: `
prebuilt_apex {
    name: "apex_tz_version",
    src: "version/tz_version",
    filename: "tz_version",
    installable: false,
}
`,
		ExpectedBazelTargets: []string{
			MakeBazelTarget("prebuilt_file", "apex_tz_version", AttrNameToString{
				"filename":    `"tz_version"`,
				"installable": `False`,
				"src":         `"version/tz_version"`,
				"dir":         `""`,
			})}})
}

func TestApexFilenameAsProperty(t *testing.T) {
	runPrebuiltApexTestCase(t, Bp2buildTestCase{
		Description: "prebuilt_apex - filename is specified as a property ",
		Filesystem:  map[string]string{},
		Blueprint: `
prebuilt_apex {
    name: "foo",
    src: "fooSrc",
    filename: "fooFileName",
}
`,
		ExpectedBazelTargets: []string{
			MakeBazelTarget("prebuilt_file", "foo", AttrNameToString{
				"filename": `"fooFileName"`,
				"src":      `"fooSrc"`,
				"dir":      `""`,
			})}})
}

func TestApexFilenameFromModuleName(t *testing.T) {
	runPrebuiltApexTestCase(t, Bp2buildTestCase{
		Description: "prebuilt_apex - neither filename nor filename_from_src are specified ",
		Filesystem:  map[string]string{},
		Blueprint: `
prebuilt_apex {
    name: "foo",
}
`,
		ExpectedBazelTargets: []string{
			MakeBazelTarget("prebuilt_file", "foo", AttrNameToString{
				"filename": `"foo"`,
				"dir":      `""`,
			})}})
}

func TestPrebuiltApexProductVariableArchSrcs(t *testing.T) {
	runPrebuiltApexTestCase(t, Bp2buildTestCase{
		Description: "prebuilt apex- Srcs from arch variant product variables",
		Filesystem:  map[string]string{},
		Blueprint: `
prebuilt_apex {
    name: "foo",
    filename: "fooFilename",
    arch: {
      arm: {
        src: "armSrc",
        product_variables: {
          native_coverage: {
            src: "nativeCoverageArmSrc",
          },
        },
      },
    },
}`,
		ExpectedBazelTargets: []string{
			MakeBazelTarget("prebuilt_file", "foo", AttrNameToString{
				"filename": `"fooFilename"`,
				"dir":      `""`,
				"src": `select({
        "//build/bazel/product_config/config_settings:native_coverage-arm": "nativeCoverageArmSrc",
        "//build/bazel_common_rules/platforms/arch:arm": "armSrc",
        "//conditions:default": None,
    })`,
			})}})
}

func TestPrebuiltApexProductVariableError(t *testing.T) {
	runPrebuiltApexTestCase(t, Bp2buildTestCase{
		Description: "",
		Filesystem:  map[string]string{},
		Blueprint: `
prebuilt_apex {
    name: "foo",
    filename: "fooFilename",
    arch: {
      arm: {
        src: "armSrc",
      },
    },
    product_variables: {
      native_coverage: {
        src: "nativeCoverageArmSrc",
      },
    },
}`,
		ExpectedErr: fmt.Errorf("label attribute could not be collapsed"),
	})
}

func TestPrebuiltApexNoConversionIfSrcEqualsName(t *testing.T) {
	runPrebuiltApexTestCase(t, Bp2buildTestCase{
		Description: "",
		Filesystem:  map[string]string{},
		Blueprint: `
prebuilt_apex {
    name: "foo",
    filename: "fooFilename",
		src: "foo",
}`,
		ExpectedBazelTargets: []string{},
	})
}
