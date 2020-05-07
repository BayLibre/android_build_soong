// Copyright 2020 Google Inc. All rights reserved.
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

package main

import (
	"github.com/golang/protobuf/proto"
	"reflect"
	"testing"

	bp "android/soong/cmd/extract_apks/bundle_proto"
)

type TestConfigDesc struct {
	name         string
	targetConfig TargetConfig
	expected     []string
}

type TestDesc struct {
	protoText string
	toc       bp.BuildApksResult
	configs   []TestConfigDesc
}

var (
	testCases = []TestDesc{
		{
			protoText: `
variant {
  targeting {
    sdk_version_targeting { 
      value { min { value: 29 } } } }
  apk_set {
    module_metadata { 
      name: "base" targeting {} delivery_type: INSTALL_TIME }
    apk_description {
      targeting {
        screen_density_targeting { 
          value { density_alias: LDPI } }
        sdk_version_targeting {
          value { min { value: 21 } } } }
      path: "splits/base-ldpi.apk"
      split_apk_metadata { split_id: "config.ldpi" } }
    apk_description {
      targeting {
        screen_density_targeting {
          value { density_alias: MDPI } }
        sdk_version_targeting {
          value { min { value: 21 } } } }
      path: "splits/base-mdpi.apk"
      split_apk_metadata { split_id: "config.mdpi" } }
    apk_description {
      targeting {
        sdk_version_targeting {
          value { min { value: 21 } } } }
      path: "splits/base-master.apk"
      split_apk_metadata { is_master_split: true } }
    apk_description {
      targeting {
        abi_targeting {
	      value { alias: ARMEABI_V7A } }
        sdk_version_targeting {
          value { min { value: 21 } } } }
      path: "splits/base-armeabi_v7a.apk"
      split_apk_metadata { split_id: "config.armeabi_v7a" } }
    apk_description {
      targeting {
        abi_targeting {
          value { alias: ARM64_V8A } }
        sdk_version_targeting {
          value { min { value: 21 } } } }
      path: "splits/base-arm64_v8a.apk"
      split_apk_metadata { split_id: "config.arm64_v8a" } }
    apk_description {
      targeting {
        abi_targeting {
          value { alias: X86 } }
        sdk_version_targeting {
          value { min { value: 21 } } } }
      path: "splits/base-x86.apk"
      split_apk_metadata { split_id: "config.x86" } }
    apk_description {
      targeting {
        abi_targeting {
          value { alias: X86_64 } }
        sdk_version_targeting {
          value { min { value: 21 } } } }
      path: "splits/base-x86_64.apk"
      split_apk_metadata { split_id: "config.x86_64" } } }
}
bundletool {
  version: "0.10.3" }

`,
			configs: []TestConfigDesc{
				{
					name: "one",
					targetConfig: TargetConfig{
						sdkVersion: 29,
						screenDpi: map[bp.ScreenDensity_DensityAlias]bool{
							bp.ScreenDensity_DENSITY_UNSPECIFIED: true,
						},
						abis: map[bp.Abi_AbiAlias]bool{
							bp.Abi_ARMEABI_V7A: true,
							bp.Abi_ARM64_V8A:   true,
						},
					},
					expected: []string{
						"splits/base-ldpi.apk",
						"splits/base-mdpi.apk",
						"splits/base-master.apk",
						"splits/base-armeabi_v7a.apk",
						"splits/base-arm64_v8a.apk",
					},
				},
				{
					name: "two",
					targetConfig: TargetConfig{
						sdkVersion: 29,
						screenDpi: map[bp.ScreenDensity_DensityAlias]bool{
							bp.ScreenDensity_LDPI: true,
						},
						abis: map[bp.Abi_AbiAlias]bool{},
					},
					expected: []string{
						"splits/base-ldpi.apk",
						"splits/base-master.apk",
					},
				},
				{
					name: "three",
					targetConfig: TargetConfig{
						sdkVersion: 20,
						screenDpi: map[bp.ScreenDensity_DensityAlias]bool{
							bp.ScreenDensity_LDPI: true,
						},
						abis: map[bp.Abi_AbiAlias]bool{},
					},
					expected: []string{},
				},
			},
		},
	}
)

func TestSelectApks(t *testing.T) {
	for _, testCase := range testCases {
		var toc bp.BuildApksResult
		if err := proto.UnmarshalText(testCase.protoText, &toc); err != nil {
			t.Fatal(err)
		}
		for _, config := range testCase.configs {
			actual, err := selectApks(&toc, config.targetConfig)
			if err != nil {
				t.Error(err)
			}
			if !reflect.DeepEqual(config.expected, actual) {
				t.Errorf("%s: expected %v, got %v", config.name, config.expected, actual)
			}
		}
	}
}
