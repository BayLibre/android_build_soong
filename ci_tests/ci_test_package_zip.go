// Copyright (C) 2025 The Android Open Source Project
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

package ci_tests

import (
	"fmt"
	"runtime"

	"android/soong/android"
	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

func init() {
	registerTestPackageZipBuildComponents(android.InitRegistrationContext)
}

func registerTestPackageZipBuildComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("package_zip", TestPackageZipFactory)
}

type testPackageZip struct {
	android.ModuleBase
	android.DefaultableModuleBase

	properties CITestPackageProperties
}

type CITestPackageProperties struct {
	Tests            []string `android:"arch_variant"`
	Device_tests     []string `android:"arch_variant"`
	Host_tests       []string `android:"arch_variant"`
	Additional_tests []string `android:"arch_variant"`
}

type testPackageZipDepTagType struct {
	blueprint.BaseDependencyTag
}

var testPackageZipDepTag testPackageZipDepTagType

var (
	moduleNamesAllowed   = []string{"continuous_native_tests_soong", "continuous_instrumentation_tests_soong", "platform_tests_soong"}
	platformsNeedCfTests = []string{"vsoc_arm", "vsoc_arm64", "vsoc_x86", "vsoc_x86_64"}
	cfTests              = []string{"CuttlefishRilTests", "CuttlefishWifiTests"}
)

func (p *testPackageZip) DepsMutator(ctx android.BottomUpMutatorContext) {
	for _, t := range p.properties.Tests {
		ctx.AddVariationDependencies(ctx.Config().AndroidCommonTarget.Variations(), testPackageZipDepTag, t)
	}

	for _, t := range p.properties.Device_tests {
		ctx.AddVariationDependencies(ctx.Config().AndroidFirstDeviceTarget.Variations(), testPackageZipDepTag, t)
	}

	for _, t := range p.properties.Host_tests {
		ctx.AddVariationDependencies(ctx.Config().BuildOSTarget.Variations(), testPackageZipDepTag, t)
	}

	if proptools.StringDefault(ctx.Config().ProductVariables().BoardPerfsetupScript, "") != "" {
		ctx.AddDependency(ctx.Module(), testPackageZipDepTag, "perf-setup")
	}

	// Additional deps need to be handled for platform_tests
	if ctx.ModuleName() == "platform_tests_soong" {
		boardPlatform := ctx.Config().ProductVariables().BoardPlatform
		if boardPlatform != nil && android.InList(*boardPlatform, platformsNeedCfTests) {
			for _, t := range cfTests {
				ctx.AddDependency(ctx.Module(), testPackageZipDepTag, t)
			}
		}
		if runtime.GOOS == "linux" {
			ctx.AddVariationDependencies(ctx.Config().BuildOSTarget.Variations(), testPackageZipDepTag, "root-canal")
		}

		//widevineTests := ctx.Config().ProductVariables().WidevineTestMakeTargets
		//if len(widevineTests) > 0 {
		//	for _, t := range widevineTests {
		//		ctx.AddDependency(ctx.Module(), testPackageZipDepTag, t)
		//	}
		//}
	}
}

func TestPackageZipFactory() android.Module {
	module := &testPackageZip{}

	module.AddProperties(&module.properties)

	android.InitAndroidArchModule(module, android.DeviceSupported, android.MultilibCommon)
	android.InitDefaultableModule(module)

	return module
}

func (p *testPackageZip) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	if !android.InList(ctx.ModuleName(), moduleNamesAllowed) {
		ctx.ModuleErrorf("%s is not allowed to use module type package_zip")
	}

	ctx.VisitDirectDepsWithTag(testPackageZipDepTag, func(m android.Module) {
		if m.Name() == "MemoryUsage" {
			fmt.Println(android.OtherModuleProvider(ctx, m, android.OutputFilesProvider))
		}
		fmt.Println("This is a dep: ", m.Name())
	})

}
