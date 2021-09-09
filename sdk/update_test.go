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

package sdk

import (
	"fmt"
	"testing"

	"android/soong/android"
	"android/soong/cc"
	"github.com/google/blueprint"
)

// Mock android.ModuleContext that records any reported errors.
type updateTestMockModuleContext struct {
	android.ModuleContext

	errs []string
}

func (u *updateTestMockModuleContext) ModuleErrorf(msg string, args ...interface{}) {
	u.errs = append(u.errs, fmt.Sprintf(msg, args...))
}

// Test android.SdkMemberType.
type updateTestSdkMemberType struct {
	android.SdkMemberTypeBase
}

var _ android.SdkMemberType = (*updateTestSdkMemberType)(nil)

func (u *updateTestSdkMemberType) AddDependencies(_ android.SdkDependencyContext, _ blueprint.DependencyTag, _ []string) {
}

func (u *updateTestSdkMemberType) IsInstance(_ android.Module) bool {
	return true
}

func (u *updateTestSdkMemberType) AddPrebuiltModule(_ android.SdkMemberContext, _ android.SdkMember) android.BpModule {
	return nil
}

func (u *updateTestSdkMemberType) CreateVariantPropertiesStruct() android.SdkMemberProperties {
	return &updateTestSdkMemberProperties{}
}

// Test android.SdkMemberProperties.
type updateTestSdkMemberProperties struct {
	android.SdkMemberPropertiesBase

	// Property to record the module variant's arch variation.
	ArchVariation string `android:"arch_variant"`
}

var _ android.SdkMemberProperties = (*updateTestSdkMemberProperties)(nil)

func (u *updateTestSdkMemberProperties) PopulateFromVariant(_ android.SdkMemberContext, variant android.Module) {
	u.ArchVariation = variant.Target().ArchVariation()
}

func (u *updateTestSdkMemberProperties) AddToPropertySet(_ android.SdkMemberContext, propertySet android.BpPropertySet) {
	if u.ArchVariation != "" {
		propertySet.AddProperty("arch_variation", u.ArchVariation)
	}
}

// Adds a native bridge target to the configured list of targets.
var prepareForTestWithNativeBridgeTarget = android.FixtureModifyConfig(func(config android.Config) {
	config.Targets[android.Android] = append(config.Targets[android.Android], android.Target{
		Os: android.Android,
		Arch: android.Arch{
			ArchType:     android.Arm64,
			ArchVariant:  "armv8-a",
			CpuVariant:   "cpu",
			Abi:          nil,
			ArchFeatures: nil,
		},
		NativeBridge:             android.NativeBridgeEnabled,
		NativeBridgeHostArchName: "x86_64",
		NativeBridgeRelativePath: "native_bridge",
	})
})

func TestDetectsNativeBridgeSpecificProperties(t *testing.T) {
	result := android.GroupFixturePreparers(
		cc.PrepareForTestWithCcDefaultModules,
		PrepareForTestWithSdkBuildComponents,
		ccTestFs.AddToFixture(),
		prepareForTestWithNativeBridgeTarget,
	).RunTestWithBp(t, `
		cc_library_headers {
			name: "mynativeheaders",
			export_include_dirs: ["myinclude"],
			stl: "none",
			system_shared_libs: [],
			native_bridge_supported: true,
			target: {
				native_bridge: {
					export_include_dirs: ["myinclude_nativebridge"],
				},
			},
		}
	`)

	arm64Variant := result.Module("mynativeheaders", "android_arm64_armv8-a")
	nativeBridgeVariant := result.Module("mynativeheaders", "android_native_bridge_arm64_armv8-a_cpu")

	variantPropertiesFactory := func() android.SdkMemberProperties {
		properties := &updateTestSdkMemberProperties{}
		properties.Base().Os_count = 1
		return properties
	}

	commonProperties := variantPropertiesFactory()
	commonProperties.Base().Os = android.CommonOS

	commonExtractor := newCommonValueExtractor(commonProperties)

	moduleCtx := &updateTestMockModuleContext{}
	ctx := &memberContext{
		sdkMemberContext: moduleCtx,
		builder:          nil,
		memberType:       &updateTestSdkMemberType{},
		name:             "test",
		requiredTraits:   android.EmptySdkMemberTraitSet(),
	}
	osInfo := newOsTypeSpecificInfo(ctx, android.Android, variantPropertiesFactory, []android.Module{arm64Variant, nativeBridgeVariant})
	osInfo.optimizeProperties(ctx, commonExtractor)

	module := newModule("test")
	propertySet := newPropertySet()
	osInfo.addToPropertySet(ctx, module, propertySet)

	android.AssertDeepEquals(t, "errors", []string{
		`Architecture variant "arm64_native_bridge" of sdk member "test" has properties distinct from other variants; this is not yet supported. The properties are:
        arch_variation: "native_bridge_arm64_armv8-a_cpu",
`}, moduleCtx.errs)
}
