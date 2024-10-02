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

package fsgen

import (
	"android/soong/android"
	"android/soong/filesystem"
	"testing"

	"github.com/google/blueprint/proptools"
)

var prepareForTestWithFsgenBuildComponents = android.FixtureRegisterWithContext(registerBuildComponents)

func TestFileSystemCreatorSystemImageProps(t *testing.T) {
	result := android.GroupFixturePreparers(
		android.PrepareForIntegrationTestWithAndroid,
		android.PrepareForTestWithAndroidBuildComponents,
		filesystem.PrepareForTestWithFilesystemBuildComponents,
		prepareForTestWithFsgenBuildComponents,
		android.FixtureModifyConfig(func(config android.Config) {
			config.TestProductVariables.PartitionVarsForSoongMigrationOnlyDoNotUse.BoardAvbEnable = true
			config.TestProductVariables.PartitionVarsForSoongMigrationOnlyDoNotUse.PartitionQualifiedVariables =
				map[string]android.PartitionQualifiedVariablesType{
					"system": {
						BoardAvbKeyPath:       "external/avb/test/data/testkey_rsa4096.pem",
						BoardAvbAlgorithm:     "SHA256_RSA4096",
						BoardAvbRollbackIndex: "0",
						BoardFileSystemType:   "ext4",
					},
				}
		}),
		android.FixtureMergeMockFs(android.MockFS{
			"external/avb/test/data/testkey_rsa4096.pem": nil,
		}),
	).RunTestWithBp(t, `
	soong_filesystem_creator {
		name: "foo",
		system: {
			name: "foo_system",
			enabled: true,
		},
	}
	`)

	fooSystem := result.ModuleForTests("foo_system", "android_common").Module().(*filesystem.SystemImage)
	android.AssertBoolEquals(
		t,
		"Property expected to match the product variable 'BOARD_AVB_ENABLE'",
		true,
		proptools.Bool(fooSystem.Filesystem.Properties.Use_avb),
	)
	android.AssertStringEquals(
		t,
		"Property expected to match the product variable 'BOARD_AVB_KEY_PATH'",
		"external/avb/test/data/testkey_rsa4096.pem",
		proptools.String(fooSystem.Filesystem.Properties.Avb_private_key),
	)
	android.AssertStringEquals(
		t,
		"Property expected to match the product variable 'BOARD_AVB_ALGORITHM'",
		"SHA256_RSA4096",
		proptools.String(fooSystem.Filesystem.Properties.Avb_algorithm),
	)
	android.AssertIntEquals(
		t,
		"Property expected to match the product variable 'BOARD_AVB_SYSTEM_ROLLBACK_INDEX'",
		0,
		proptools.Int(fooSystem.Filesystem.Properties.Rollback_index),
	)
	android.AssertStringEquals(
		t,
		"Property expected to match the product variable 'BOARD_SYSTEMIMAGE_FILE_SYSTEM_TYPE'",
		"ext4",
		proptools.String(fooSystem.Filesystem.Properties.Type),
	)
}
