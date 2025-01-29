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
}

type CITestPackageProperties struct {
	Tests              []string `android:"arch_variant"`
	Device_first_tests []string `android:"arch_variant"`
	Device_both_tests  []string `android:"arch_variant"`
	Host_tests         []string `android:"arch_variant"`
	Additional_tests   []string `android:"arch_variant"`
}

type testPackageZipDepTagType struct {
	blueprint.BaseDependencyTag
}

var testPackageZipDepTag testPackageZipDepTagType

var (
	pctx                 = android.NewPackageContext("android/soong/ci_tests")
	moduleNamesAllowed   = []string{"continuous_native_tests_soong", "continuous_instrumentation_tests_soong", "platform_tests_soong"}
	platformsNeedCfTests = []string{"vsoc_arm", "vsoc_arm64", "vsoc_x86", "vsoc_x86_64"}
	cfTests              = []string{"CuttlefishRilTests", "CuttlefishWifiTests"}
)

func (p *testPackageZip) DepsMutator(ctx android.BottomUpMutatorContext) {
	for _, t := range p.properties.Tests {
		ctx.AddVariationDependencies(ctx.Config().AndroidCommonTarget.Variations(), testPackageZipDepTag, t)
	}

	for _, t := range p.properties.Device_first_tests {
		ctx.AddVariationDependencies(ctx.Config().AndroidFirstDeviceTarget.Variations(), testPackageZipDepTag, t)
	}

	var maybeAndroid32Target *android.Target
	var maybeAndroid64Target *android.Target
	android32TargetList := android.FirstTarget(ctx.Config().Targets[android.Android], "lib32")
	android64TargetList := android.FirstTarget(ctx.Config().Targets[android.Android], "lib64")
	if len(android32TargetList) > 0 {
		maybeAndroid32Target = &android32TargetList[0]
	}
	if len(android64TargetList) > 0 {
		maybeAndroid64Target = &android64TargetList[0]
	}
	fmt.Println("AAAAAA", maybeAndroid32Target, maybeAndroid64Target)
	ctx.AddFarVariationDependencies(maybeAndroid32Target.Variations(), testPackageZipDepTag, p.properties.Device_both_tests...)
	ctx.AddFarVariationDependencies(maybeAndroid64Target.Variations(), testPackageZipDepTag, p.properties.Device_both_tests...)

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

	// fmt.Println("Module Name:", p.Name())

	installedFilesToClass := make(map[string]string)
	ctx.VisitDirectDepsWithTag(testPackageZipDepTag, func(m android.Module) {
		updateInstalledFilesToClass(ctx, m, installedFilesToClass)
	})

	// fmt.Println(installedFilesToClass, len(installedFilesToClass))
	stagingDir := android.PathForModuleOut(ctx, "STAGING").String()
	// fmt.Println("StagingDir:", stagingDir)

	output := createOutput(ctx, pctx, stagingDir, installedFilesToClass)

	ctx.SetOutputFiles(android.Paths{output}, "")
	ctx.CheckbuildFile(output)
	//installDir := android.PathForModuleInstall(ctx, p.Name())

}

func updateInstalledFilesToClass(ctx android.ModuleContext, m android.Module, installedFilesToClass map[string]string) {
	info, ok := android.OtherModuleProvider(ctx, m, android.ModuleInfoJSONProvider)
	if ok {
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

		installed := info[0].GetInstalled()
		// fmt.Println("Installed files of module", m.Name(), "||", info[0].GetInstalled(), "|| Class:", info[0].GetClass())
		for _, f := range installed {
			installedFilesToClass[f] = class
		}
	} else {
		fmt.Println("This MODULE doesn't set ModuleInfoJSON provider:", m.Name())
	}
}

func createOutput(ctx android.ModuleContext, pctx android.PackageContext, stagingDir string, installedFilesToClass map[string]string) android.ModuleOutPath {
	productOut := filepath.Join(ctx.Config().OutDir(), "target", "product", ctx.Config().DeviceName())
	// fmt.Println("ProductOut:", productOut)
	productVariables := ctx.Config().ProductVariables()
	arch := proptools.String(productVariables.DeviceArch)
	secondArch := proptools.String(productVariables.DeviceSecondaryArch)
	// fmt.Println("Arch", arch, "2nd Arch", secondArch)
	builder := android.NewRuleBuilder(pctx, ctx)
	builder.Command().Text("rm").Flag("-rf").Text(stagingDir)
	builder.Command().Text("mkdir").Flag("-p").Text(stagingDir)
	for file, class := range installedFilesToClass {
		name := removeFileExtension(filepath.Base(file))
		f := strings.TrimPrefix(file, productOut+"/")
		f = strings.ReplaceAll(f, "system/", "DATA/")
		f = strings.ReplaceAll(f, filepath.Join("testcases", name, arch), filepath.Join("DATA", class, name))
		f = strings.ReplaceAll(f, filepath.Join("testcases", name, secondArch), filepath.Join("DATA", class, name))
		f = strings.ReplaceAll(f, "testcases", filepath.Join("DATA", class))
		f = strings.ReplaceAll(f, "data/", "DATA/")
		f = strings.ReplaceAll(f, "DATA_other", "system_other")
		f = strings.ReplaceAll(f, "system_other/DATA", "system_other/system")
		if strings.HasPrefix(f, "out") {
			// fmt.Println("!!!!!!!!!!!!!!!!!Not gonna copy:", file, class)
			continue
		}
		// fmt.Println(f)
		dir := filepath.Dir(f)
		builder.Command().Text("mkdir").Flag("-p").Text(filepath.Join(stagingDir, dir))
		builder.Command().Text("cp").Flag("-Rf").Text(file).Text(filepath.Join(stagingDir, f))
	}
	builder.Command().Text("cd").Text(stagingDir)
	output := android.PathForModuleOut(ctx, "aaa.zip")
	fmt.Println("Output is:", output.String())
	builder.Command().Text("zip").Flag("-rqX").Output(output)
	builder.Command().Text("rm").Flag("-rf").Text(stagingDir)
	builder.Build("package_zip_copy", "Copy package_zip dep installed files")
	return output
}

func removeFileExtension(filename string) string {
	return strings.TrimSuffix(filename, filepath.Ext(filename))
}

func (p *testPackageZip) AndroidMkEntries() []android.AndroidMkEntries {
	return []android.AndroidMkEntries{}
}
