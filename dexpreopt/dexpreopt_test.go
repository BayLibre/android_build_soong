// Copyright 2018 Google Inc. All rights reserved.
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

package dexpreopt

import (
	"android/soong/android"
	"fmt"
	"testing"
)

func testSystemModuleConfig(ctx android.PathContext, name string) *ModuleConfig {
	return testModuleConfig(ctx, name, "system")
}

func testSystemProductModuleConfig(ctx android.PathContext, name string) *ModuleConfig {
	return testModuleConfig(ctx, name, "system/product")
}

func testProductModuleConfig(ctx android.PathContext, name string) *ModuleConfig {
	return testModuleConfig(ctx, name, "product")
}

func testModuleConfig(ctx android.PathContext, name, partition string) *ModuleConfig {
	return &ModuleConfig{
		Name:                            name,
		DexLocation:                     fmt.Sprintf("/%s/app/test/%s.apk", partition, name),
		BuildPath:                       android.PathForOutput(ctx, fmt.Sprintf("%s/%s.apk", name, name)),
		DexPath:                         android.PathForOutput(ctx, fmt.Sprintf("%s/dex/%s.jar", name, name)),
		UncompressedDex:                 false,
		HasApkLibraries:                 false,
		PreoptFlags:                     nil,
		ProfileClassListing:             android.OptionalPath{},
		ProfileIsTextListing:            false,
		EnforceUsesLibrariesStatusFile:  android.PathForOutput(ctx, fmt.Sprintf("%s/enforce_uses_libraries.status", name)),
		EnforceUsesLibraries:            false,
		ClassLoaderContexts:             nil,
		Archs:                           []android.ArchType{android.Arm},
		DexPreoptImages:                 android.Paths{android.PathForTesting("system/framework/arm/boot.art")},
		DexPreoptImagesDeps:             []android.OutputPaths{android.OutputPaths{}},
		DexPreoptImageLocations:         []string{},
		PreoptBootClassPathDexFiles:     nil,
		PreoptBootClassPathDexLocations: nil,
		PreoptExtractedApk:              false,
		NoCreateAppImage:                false,
		ForceCreateAppImage:             false,
		PresignedPrebuilt:               false,
	}
}

func TestDexPreopt(t *testing.T) {
	config := android.TestConfig("out", nil, "", nil)
	ctx := android.BuilderContextForTesting(config)
	globalSoong := globalSoongConfigForTests()
	global := GlobalConfigForTests(ctx)
	module := testSystemModuleConfig(ctx, "test")

	rule, err := GenerateDexpreoptRule(ctx, globalSoong, global, module)
	if err != nil {
		t.Fatal(err)
	}

	wantInstalls := android.RuleBuilderInstalls{
		{android.PathForOutput(ctx, "test/oat/arm/package.odex"), "/system/app/test/oat/arm/test.odex"},
		{android.PathForOutput(ctx, "test/oat/arm/package.vdex"), "/system/app/test/oat/arm/test.vdex"},
	}

	if rule.Installs().String() != wantInstalls.String() {
		t.Errorf("\nwant installs:\n   %v\ngot:\n   %v", wantInstalls, rule.Installs())
	}
}

func TestDexPreoptSystemOther(t *testing.T) {
	config := android.TestConfig("out", nil, "", nil)
	ctx := android.BuilderContextForTesting(config)
	globalSoong := globalSoongConfigForTests()
	global := GlobalConfigForTests(ctx)
	systemModule := testSystemModuleConfig(ctx, "Stest")
	systemProductModule := testSystemProductModuleConfig(ctx, "SPtest")
	productModule := testProductModuleConfig(ctx, "Ptest")

	global.HasSystemOther = true

	type moduleTest struct {
		module            *ModuleConfig
		expectedPartition string
	}
	tests := []struct {
		patterns    []string
		moduleTests []moduleTest
	}{
		{
			patterns: []string{"app/%"},
			moduleTests: []moduleTest{
				{module: systemModule, expectedPartition: "system_other/system"},
				{module: systemProductModule, expectedPartition: "system/product"},
				{module: productModule, expectedPartition: "product"},
			},
		},
		// product/app/% only applies to product apps inside the system partition
		{
			patterns: []string{"app/%", "product/app/%"},
			moduleTests: []moduleTest{
				{module: systemModule, expectedPartition: "system_other/system"},
				{module: systemProductModule, expectedPartition: "system_other/system/product"},
				{module: productModule, expectedPartition: "product"},
			},
		},
	}

	for _, test := range tests {
		global.PatternsOnSystemOther = test.patterns
		for _, mt := range test.moduleTests {
			rule, err := GenerateDexpreoptRule(ctx, globalSoong, global, mt.module)
			if err != nil {
				t.Fatal(err)
			}

			name := mt.module.Name
			wantInstalls := android.RuleBuilderInstalls{
				{android.PathForOutput(ctx, name+"/oat/arm/package.odex"), fmt.Sprintf("/%s/app/test/oat/arm/%s.odex", mt.expectedPartition, name)},
				{android.PathForOutput(ctx, name+"/oat/arm/package.vdex"), fmt.Sprintf("/%s/app/test/oat/arm/%s.vdex", mt.expectedPartition, name)},
			}

			if rule.Installs().String() != wantInstalls.String() {
				t.Errorf("\nwant installs:\n   %v\ngot:\n   %v", wantInstalls, rule.Installs())
			}
		}
	}

}

func TestDexPreoptProfile(t *testing.T) {
	config := android.TestConfig("out", nil, "", nil)
	ctx := android.BuilderContextForTesting(config)
	globalSoong := globalSoongConfigForTests()
	global := GlobalConfigForTests(ctx)
	module := testSystemModuleConfig(ctx, "test")

	module.ProfileClassListing = android.OptionalPathForPath(android.PathForTesting("profile"))

	rule, err := GenerateDexpreoptRule(ctx, globalSoong, global, module)
	if err != nil {
		t.Fatal(err)
	}

	wantInstalls := android.RuleBuilderInstalls{
		{android.PathForOutput(ctx, "test/profile.prof"), "/system/app/test/test.apk.prof"},
		{android.PathForOutput(ctx, "test/oat/arm/package.art"), "/system/app/test/oat/arm/test.art"},
		{android.PathForOutput(ctx, "test/oat/arm/package.odex"), "/system/app/test/oat/arm/test.odex"},
		{android.PathForOutput(ctx, "test/oat/arm/package.vdex"), "/system/app/test/oat/arm/test.vdex"},
	}

	if rule.Installs().String() != wantInstalls.String() {
		t.Errorf("\nwant installs:\n   %v\ngot:\n   %v", wantInstalls, rule.Installs())
	}
}

func TestDexPreoptConfigToJson(t *testing.T) {
	config := android.TestConfig("out", nil, "", nil)
	ctx := android.BuilderContextForTesting(config)
	module := testSystemModuleConfig(ctx, "test")
	data, err := moduleConfigToJSON(module)
	if err != nil {
		t.Errorf("Failed to convert module config data to JSON, %v", err)
	}
	parsed, err := ParseModuleConfig(ctx, data)
	if err != nil {
		t.Errorf("Failed to parse JSON, %v", err)
	}
	before := fmt.Sprintf("%v", module)
	after := fmt.Sprintf("%v", parsed)
	android.AssertStringEquals(t, "The result must be the same as the original after marshalling and unmarshalling it.", before, after)
}

func TestDexPreoptConfigCompatibility(t *testing.T) {
	config := android.TestConfig("out", nil, "", nil)
	ctx := android.BuilderContextForTesting(config)
	// An arbitrary json file at the moment to keep compatibility
	json := `
	{
		"BuildPath": "out/soong/.intermediates/packages/apps/DocumentsUI/DocumentsUI/android_common/dexpreopt/DocumentsUI.jar",
		"DexPath": "out/soong/.intermediates/packages/apps/DocumentsUI/DocumentsUI/android_common/aligned/DocumentsUI.jar",
		"ManifestPath": "out/soong/.intermediates/packages/apps/DocumentsUI/DocumentsUI/android_common/manifest_merger/AndroidManifest.xml",
		"ProfileClassListing": "",
		"ProfileBootListing": "",
		"EnforceUsesLibrariesStatusFile": "out/soong/.intermediates/packages/apps/DocumentsUI/DocumentsUI/android_common/enforce_uses_libraries.status",
		"ClassLoaderContexts": null,
		"DexPreoptImages": [
			"out/soong/vsoc_x86/dex_bootjars/android/system/framework/x86/boot-framework.art"
		],
		"DexPreoptImagesDeps": [
			[
				"out/soong/vsoc_x86/dex_bootjars/android/system/framework/x86/boot-framework.art",
				"out/soong/vsoc_x86/dex_bootjars/android/system/framework/x86/boot-framework.oat",
				"out/soong/vsoc_x86/dex_bootjars/android/system/framework/x86/boot-framework.vdex",
				"out/soong/vsoc_x86/dex_bootjars/android/system/framework/x86/boot-ext.art",
				"out/soong/vsoc_x86/dex_bootjars/android/system/framework/x86/boot-ext.oat",
				"out/soong/vsoc_x86/dex_bootjars/android/system/framework/x86/boot-ext.vdex",
				"out/soong/vsoc_x86/dex_bootjars/android/system/framework/x86/boot-core-icu4j.art",
				"out/soong/vsoc_x86/dex_bootjars/android/system/framework/x86/boot-core-icu4j.oat",
				"out/soong/vsoc_x86/dex_bootjars/android/system/framework/x86/boot-core-icu4j.vdex",
				"out/soong/vsoc_x86/dex_bootjars/android/system/framework/x86/boot-telephony-common.art",
				"out/soong/vsoc_x86/dex_bootjars/android/system/framework/x86/boot-telephony-common.oat",
				"out/soong/vsoc_x86/dex_bootjars/android/system/framework/x86/boot-telephony-common.vdex",
				"out/soong/vsoc_x86/dex_bootjars/android/system/framework/x86/boot-voip-common.art",
				"out/soong/vsoc_x86/dex_bootjars/android/system/framework/x86/boot-voip-common.oat",
				"out/soong/vsoc_x86/dex_bootjars/android/system/framework/x86/boot-voip-common.vdex",
				"out/soong/vsoc_x86/dex_bootjars/android/system/framework/x86/boot-ims-common.art",
				"out/soong/vsoc_x86/dex_bootjars/android/system/framework/x86/boot-ims-common.oat",
				"out/soong/vsoc_x86/dex_bootjars/android/system/framework/x86/boot-ims-common.vdex",
				"out/soong/vsoc_x86/dex_bootjars/android/system/framework/x86/boot-framework-atb-backward-compatibility.art",
				"out/soong/vsoc_x86/dex_bootjars/android/system/framework/x86/boot-framework-atb-backward-compatibility.oat",
				"out/soong/vsoc_x86/dex_bootjars/android/system/framework/x86/boot-framework-atb-backward-compatibility.vdex"
			]
		],
		"PreoptBootClassPathDexFiles": [
			"out/soong/vsoc_x86/dex_artjars_input/core-oj.jar",
			"out/soong/vsoc_x86/dex_artjars_input/core-libart.jar",
			"out/soong/vsoc_x86/dex_artjars_input/okhttp.jar",
			"out/soong/vsoc_x86/dex_artjars_input/bouncycastle.jar",
			"out/soong/vsoc_x86/dex_artjars_input/apache-xml.jar",
			"out/soong/vsoc_x86/dex_bootjars_input/framework.jar",
			"out/soong/vsoc_x86/dex_bootjars_input/ext.jar",
			"out/soong/vsoc_x86/dex_bootjars_input/core-icu4j.jar",
			"out/soong/vsoc_x86/dex_bootjars_input/telephony-common.jar",
			"out/soong/vsoc_x86/dex_bootjars_input/voip-common.jar",
			"out/soong/vsoc_x86/dex_bootjars_input/ims-common.jar",
			"out/soong/vsoc_x86/dex_bootjars_input/framework-atb-backward-compatibility.jar",
			"out/soong/vsoc_x86/updatable_bootjars/conscrypt.jar",
			"out/soong/vsoc_x86/updatable_bootjars/updatable-media.jar",
			"out/soong/vsoc_x86/updatable_bootjars/framework-mediaprovider.jar",
			"out/soong/vsoc_x86/updatable_bootjars/framework-statsd.jar",
			"out/soong/vsoc_x86/updatable_bootjars/framework-permission.jar",
			"out/soong/vsoc_x86/updatable_bootjars/framework-sdkextensions.jar",
			"out/soong/vsoc_x86/updatable_bootjars/framework-wifi.jar",
			"out/soong/vsoc_x86/updatable_bootjars/framework-tethering.jar",
			"out/soong/vsoc_x86/updatable_bootjars/android.net.ipsec.ike.jar"
		],
		"Name": "DocumentsUI",
		"DexLocation": "/system/priv-app/DocumentsUI/DocumentsUI.apk",
		"UncompressedDex": true,
		"HasApkLibraries": false,
		"PreoptFlags": null,
		"ProfileIsTextListing": false,
		"EnforceUsesLibraries": true,
		"ProvidesUsesLibrary": "DocumentsUI",
		"Archs": [
			"x86"
		],
		"DexPreoptImageLocations": [
			"out/soong/vsoc_x86/dex_artjars/android/apex/art_boot_images/javalib/boot.art",
			"out/soong/vsoc_x86/dex_bootjars/android/system/framework/boot-framework.art"
		],
		"PreoptBootClassPathDexLocations": [
			"/apex/com.android.art/javalib/core-oj.jar",
			"/apex/com.android.art/javalib/core-libart.jar",
			"/apex/com.android.art/javalib/okhttp.jar",
			"/apex/com.android.art/javalib/bouncycastle.jar",
			"/apex/com.android.art/javalib/apache-xml.jar",
			"/system/framework/framework.jar",
			"/system/framework/ext.jar",
			"/apex/com.android.i18n/javalib/core-icu4j.jar",
			"/system/framework/telephony-common.jar",
			"/system/framework/voip-common.jar",
			"/system/framework/ims-common.jar",
			"/system/framework/framework-atb-backward-compatibility.jar",
			"/apex/com.android.conscrypt/javalib/conscrypt.jar",
			"/apex/com.android.media/javalib/updatable-media.jar",
			"/apex/com.android.mediaprovider/javalib/framework-mediaprovider.jar",
			"/apex/com.android.os.statsd/javalib/framework-statsd.jar",
			"/apex/com.android.permission/javalib/framework-permission.jar",
			"/apex/com.android.sdkext/javalib/framework-sdkextensions.jar",
			"/apex/com.android.wifi/javalib/framework-wifi.jar",
			"/apex/com.android.tethering/javalib/framework-tethering.jar",
			"/apex/com.android.ipsec/javalib/android.net.ipsec.ike.jar"
		],
		"PreoptExtractedApk": false,
		"NoCreateAppImage": false,
		"ForceCreateAppImage": false,
		"PresignedPrebuilt": false
	}
	`
	_, err := ParseModuleConfig(ctx, []byte(json))
	if err != nil {
		t.Errorf("Failed to parse JSON, %v", err)
	}
}
