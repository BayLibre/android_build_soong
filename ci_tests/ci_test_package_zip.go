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
	"path/filepath"
	"runtime"
	"strings"

	"android/soong/android"
	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

func init() {
	pctx.Import("android/soong/android")
	registerTestPackageZipBuildComponents(android.InitRegistrationContext)
}

func registerTestPackageZipBuildComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("package_zip", TestPackageZipFactory)
}

type testPackageZip struct {
	android.ModuleBase
	android.DefaultableModuleBase

	properties CITestPackageProperties

	output android.Path
}

type CITestPackageProperties struct {
	// test modules will be added as dependencies using the device os and the common architecture's variant.
	Tests []string `android:"arch_variant"`
	// test modules that will be added as dependencies based on the first supported arch variant and the device os variant
	Device_first_tests []string `android:"arch_variant"`
	// test modules that will be added as dependencies based on both 32bit and 64bit arch variant and the device os variant
	Device_both_tests []string `android:"arch_variant"`
	// test modules that will be added as dependencies based on host
	Host_tests []string `android:"arch_variant"`
	// git-main only test modules. Will only be added as dependencies if exists.
	Additional_tests []string `android:"arch_variant"`
}

type testPackageZipDepTagType struct {
	blueprint.BaseDependencyTag
}

var testPackageZipDepTag testPackageZipDepTagType

var (
	pctx = android.NewPackageContext("android/soong/ci_tests")
	// package_zip module type should only be used for the following modules.
	// TODO: remove "_soong" from the module names inside when eliminating the corresponding make modules
	moduleNamesAllowed   = []string{"continuous_native_tests_soong", "continuous_instrumentation_tests_soong", "platform_tests"}
	platformsNeedCfTests = []string{"vsoc_arm", "vsoc_arm64", "vsoc_x86", "vsoc_x86_64"}
	cfTests              = []string{"CuttlefishRilTests", "CuttlefishWifiTests"}
)

func (p *testPackageZip) DepsMutator(ctx android.BottomUpMutatorContext) {
	// adding tests property deps
	for _, t := range p.properties.Tests {
		ctx.AddVariationDependencies(ctx.Config().AndroidCommonTarget.Variations(), testPackageZipDepTag, t)
	}

	// adding device_first_tests property deps
	for _, t := range p.properties.Device_first_tests {
		ctx.AddVariationDependencies(ctx.Config().AndroidFirstDeviceTarget.Variations(), testPackageZipDepTag, t)
	}

	// adding device_both_tests property deps
	var maybeAndroid32Target *android.Target
	var maybeAndroid64Target *android.Target
	android32TargetList := android.FirstTarget(ctx.Config().Targets[android.Android], "lib32")
	android64TargetList := android.FirstTarget(ctx.Config().Targets[android.Android], "lib64")
	if len(android32TargetList) > 0 {
		maybeAndroid32Target = &android32TargetList[0]
		ctx.AddFarVariationDependencies(maybeAndroid32Target.Variations(), testPackageZipDepTag, p.properties.Device_both_tests...)
	}
	if len(android64TargetList) > 0 {
		maybeAndroid64Target = &android64TargetList[0]
		ctx.AddFarVariationDependencies(maybeAndroid64Target.Variations(), testPackageZipDepTag, p.properties.Device_both_tests...)
	}

	// adding host_tests property deps
	for _, t := range p.properties.Host_tests {
		ctx.AddVariationDependencies(ctx.Config().BuildOSTarget.Variations(), testPackageZipDepTag, t)
	}

	// Checking module exists before adding additional_tests prpperty deps
	for _, t := range p.properties.Additional_tests {
		if ctx.OtherModuleExists(t) {
			ctx.AddVariationDependencies(ctx.Config().AndroidCommonTarget.Variations(), testPackageZipDepTag, t)
		}
	}

	// perf-setup dep is needed by all the package_zip modules
	if proptools.StringDefault(ctx.Config().ProductVariables().BoardPerfsetupScript, "") != "" {
		ctx.AddVariationDependencies(ctx.Config().AndroidFirstDeviceTarget.Variations(), testPackageZipDepTag, "perf-setup")
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

		widevineTests := ctx.Config().ProductVariables().WidevineTestMakeTargets
		if len(widevineTests) > 0 {
			for _, t := range widevineTests {
				ctx.AddDependency(ctx.Module(), testPackageZipDepTag, t)
			}
		}
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

	p.output = createOutput(ctx, pctx)

	ctx.SetOutputFiles(android.Paths{p.output}, "")

	// dist the test output
	if ctx.ModuleName() == "platform_tests_soong" {
		distedName := ctx.Config().Getenv("TARGET_PRODUCT") + "-tests-" + ctx.Config().BuildId() + ".zip"
		ctx.DistForGoalWithFilename("platform_tests", p.output, distedName)
	}
}

func createOutput(ctx android.ModuleContext, pctx android.PackageContext) android.ModuleOutPath {
	productOut := filepath.Join(ctx.Config().OutDir(), "target", "product", ctx.Config().DeviceName())
	stagingDir := android.PathForModuleOut(ctx, "STAGING")
	productVariables := ctx.Config().ProductVariables()
	arch := proptools.String(productVariables.DeviceArch)
	secondArch := proptools.String(productVariables.DeviceSecondaryArch)

	builder := android.NewRuleBuilder(pctx, ctx)
	builder.Command().Text("rm").Flag("-rf").Text(stagingDir.String())
	builder.Command().Text("mkdir").Flag("-p").Output(stagingDir)
	builder.Temporary(stagingDir)
	ctx.VisitDirectDepsWithTag(testPackageZipDepTag, func(m android.Module) {
		info, ok := android.OtherModuleProvider(ctx, m, android.ModuleInfoJSONProvider)
		if !ok {
			ctx.ModuleErrorf("This MODULE %s doesn't set ModuleInfoJSON provider:", m.Name())
		}
		if len(info) != 1 {
			ctx.ModuleErrorf("Module %s doesn't provide exactly one ModuleInfoJSON", m.Name())
		}

		classes := info[0].GetClass()
		if len(info[0].Class) != 1 {
			ctx.ModuleErrorf("Module %s doesn't have exactly one class in its ModuleInfoJSON", m.Name())
		}
		class := strings.ToLower(classes[0])
		if class == "apps" {
			class = "app"
		} else if class == "java_libraries" {
			class = "framework"
		}

		installedFilesInfo, ok := android.OtherModuleProvider(ctx, m, android.InstallFilesProvider)
		if !ok {
			ctx.ModuleErrorf("Module %s doesn't set InstallFilesProvider", m.Name())
		}

		for _, installedFile := range installedFilesInfo.InstallFiles {
			name := removeFileExtension(installedFile.Base())
			f := strings.TrimPrefix(installedFile.String(), productOut+"/")
			if strings.HasPrefix(f, "out") {
				continue
			}
			f = strings.ReplaceAll(f, "system/", "DATA/")
			f = strings.ReplaceAll(f, filepath.Join("testcases", name, arch), filepath.Join("DATA", class, name))
			f = strings.ReplaceAll(f, filepath.Join("testcases", name, secondArch), filepath.Join("DATA", class, name))
			f = strings.ReplaceAll(f, "testcases", filepath.Join("DATA", class))
			f = strings.ReplaceAll(f, "data/", "DATA/")
			f = strings.ReplaceAll(f, "DATA_other", "system_other")
			f = strings.ReplaceAll(f, "system_other/DATA", "system_other/system")
			dir := filepath.Dir(f)
			tempOut := android.PathForModuleOut(ctx, "STAGING", f)
			builder.Command().Text("mkdir").Flag("-p").Text(filepath.Join(stagingDir.String(), dir))
			builder.Command().Text("cp").Flag("-Rf").Input(installedFile).Output(tempOut)
			builder.Temporary(tempOut)
		}
	})

	output := android.PathForModuleOut(ctx, ctx.ModuleName()+".zip")
	builder.Command().
		BuiltTool("soong_zip").
		Flag("-o").Output(output).
		Flag("-C").Text(stagingDir.String()).
		Flag("-D").Text(stagingDir.String())
	builder.Command().Text("rm").Flag("-rf").Text(stagingDir.String())
	builder.Build("package_zip", fmt.Sprintf("build package_zip for %s", ctx.ModuleName()))
	return output
}

func removeFileExtension(filename string) string {
	return strings.TrimSuffix(filename, filepath.Ext(filename))
}

// The only purpose of this method is to make sure we can build the module directly
// without adding suffix "-soong"
func (p *testPackageZip) AndroidMkEntries() []android.AndroidMkEntries {
	return []android.AndroidMkEntries{
		android.AndroidMkEntries{
			Class:      "ETC",
			OutputFile: android.OptionalPathForPath(p.output),
		},
	}
}
