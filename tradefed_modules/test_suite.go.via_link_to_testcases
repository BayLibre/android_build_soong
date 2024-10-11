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

package tradefed_modules

import (
	"android/soong/android"
	"android/soong/java"
	"android/soong/tradefed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/google/blueprint"
)

/*
   /*
   TODO(RON): here.
.
└── out/
    ├── suites/
    │   └── my-team-suite/
    │       ├── my-team-suite.json
    │       ├── config/
    │       │       └── my-team-suite.xml
    │       ├── host/
    │       │   └── testcases
    │       │       └── MyTeamTest1
    │       │           ├── MyTeamTest1.apk -> _path_to_MyTeamTest1.apk_
    │       │           └── MyTeamTest1.config -> _path_to_MyTeamTest1.config_
    │       └── target
    │           └── testcases
    │               └── MyTeamTest2
    │                   ├── MyTeamTest2.apk -> _path_to_MyTeamTest2.apk_
    │                   └── MyTeamTest2.config -> _path_to_MyTeamTest12.config_

   my-team-suite.json
{
  "name": "my-team-suite",
  "config": "config/my-team-suite.xml",
  "files": [
      "config/my-team-suite.xml",
      "host/testcases/MyTeamTest1/MyTeamTest1.config",
      "host/testcases/MyTeamTest1/MyTeamTest1.apk",
      "target/testcases/MyTeamTest2/MyTeamTest2.config",
      "target/testcases/MyTeamTest2/MyTeamTest2.apk"
  ]
}

*/

// Directory inside of out/soong to store suite symlink forrests and manifests
const suiteDirectory = "suites"

// const manifestFileName = "suite.%s.manifest.json"
const manifestFileName = "%s.json"
const zipFileName = "suite.%s.zip"

func init() {
	RegisterTestSuiteBuildComponents(android.InitRegistrationContext)
}

func RegisterTestSuiteBuildComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("test_suite", TestSuiteFactory)
	ctx.RegisterModuleType("test_module_group", TestModuleGroupFactory)
	ctx.RegisterModuleType("test_module_tag", TestModuleTagFactory)
	ctx.RegisterModuleType("tradefed_xml_config", TestConfigFactory)
	// TODO(rbraunstein): config_includes
}

var PrepareForTestWithTestSuiteBuildComponents = android.GroupFixturePreparers(
	android.FixtureRegisterWithContext(RegisterTestSuiteBuildComponents),
)

type testSuiteProperties struct {
	Config string `android:"path"`
}

type testModuleGroupProperties struct {
	Description string
	// Owner on base already?
	// generic test module
	Tests []string `android:"path,arch_variant"`
	// `test_module_tag` modules
	Tags []string `android:"path"`
}

type testModuleTagProperties struct {
	Description string
}

type testConfigProperties struct {
	// Tradefed xml files to include in order
	Srcs []string `android:"path,arch_variant"`
	// Modules containing classes referenced by the xml files.
	Deps []string `android:"path,arch_variant"`
}

type testSuiteModule struct {
	testModuleGroup

	properties testSuiteProperties
}

type testModuleGroup struct {
	android.ModuleBase
	android.DefaultableModuleBase
	testModuleGroupProperties

	// Path where to write the manifest per suite.
	// Add new one for zip vs manifest.
	outputFiles   []android.OutputPath
	zipOutputPath android.OutputPath
	//modules     []moduleName
	artifacts []android.Paths
}

type testModuleTagModule struct {
	android.ModuleBase
	android.DefaultableModuleBase

	testModuleTagProperties
}

type testConfigModule struct {
	android.ModuleBase
	android.DefaultableModuleBase

	testConfigProperties
}

func (t *testConfigModule) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	// Nothing to do, walk happens in the suite.
}

// Rules for
//   - suite
//     Generate the manifest json file.
//     Ensure all or deps are built before we are.
//   - suite.zip
func (t *testSuiteModule) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	t.testModuleGroup.GenerateAndroidBuildActions(ctx)
	t.manifestOrZip(ctx)
}

func (t *testModuleGroup) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	// TODO(rbraunstein): Visit all deps and ensure we list all tags, other field err
	//     i.e. we are doing an "AND", not an "OR"
	// TODO(rbraunstein): validate that Module.Modules are valid or in AddDep
	//     I think that is already done for us.
	// TODO(rbraunstein):
	//    Ensure that if people uses Tests: and list a test_module_config entry, but
	//    the base isn't listed, then we get an error.
	//    Or somehow fix the symlinks to copy instead of link in this case.
}

func (t *testModuleTagModule) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	// Nothing to do.
}

// Add our direct "tests:" deps here.
// The "tags:" deps are added in providers with AddReverseDependency
// Or do we need to register a top down mutator to add it ourselves?
func (t *testModuleGroup) DepsMutator(ctx android.BottomUpMutatorContext) {
	for _, test := range t.Tests {
		if ctx.OtherModuleDependencyVariantExists(ctx.Config().BuildOSCommonTarget.Variations(), test) {
			// Host tests seem to need this
			ctx.AddVariationDependencies(ctx.Config().BuildOSCommonTarget.Variations(), tradefed.TestSuiteTag{}, test)
		} else {
			// And device tests get added this way.
			ctx.AddDependency(ctx.Module(), tradefed.TestSuiteTag{}, test)
		}
	}

	// Add a dependency on the test_module_tag as well because _test providers add a
	// reverse dependency to the tag and later we need to find them.
	for _, tag := range t.Tags {
		ctx.AddDependency(ctx.Module(), tradefed.TestSuiteTag{}, tag)
	}
}

func (t *testSuiteModule) DepsMutator(ctx android.BottomUpMutatorContext) {
	t.testModuleGroup.DepsMutator(ctx)

	ctx.AddDependency(ctx.Module(), configTag{}, t.properties.Config)
}

type configTag struct {
	blueprint.BaseDependencyTag
	suiteName string
}

func (t *testConfigModule) DepsMutator(ctx android.BottomUpMutatorContext) {
	for _, dep := range t.Deps {
		if ctx.OtherModuleDependencyVariantExists(ctx.Config().BuildOSCommonTarget.Variations(), dep) {
			// Host tests seem to need this
			ctx.AddVariationDependencies(ctx.Config().BuildOSCommonTarget.Variations(), configTag{}, dep)
		} else {
			ctx.AddDependency(ctx.Module(), configTag{}, dep)
		}
	}
}

// Will it work for both host and device tests with FarVariationDependencies.
func TestSuiteFactory() android.Module {
	module := &testSuiteModule{}
	module.AddProperties(&module.testModuleGroupProperties, &module.properties)

	android.InitAndroidModule(module)
	android.InitDefaultableModule(module)

	return module
}

func TestModuleGroupFactory() android.Module {
	module := &testModuleGroup{}
	module.AddProperties(&module.testModuleGroupProperties)

	android.InitAndroidModule(module)
	android.InitDefaultableModule(module)

	return module
}

func TestModuleTagFactory() android.Module {
	module := &testModuleTagModule{}
	module.AddProperties(&module.testModuleTagProperties)

	android.InitAndroidModule(module)
	android.InitDefaultableModule(module)

	return module
}

func TestConfigFactory() android.Module {
	module := &testConfigModule{}
	module.AddProperties(&module.testConfigProperties)

	android.InitAndroidModule(module)
	android.InitDefaultableModule(module)

	return module
}

// Generate a manifest file that has our modules and their artifacts.
// Get the artifacts from the providers.
// The providers give us source paths in the .intermediate directory.
// We transform those paths to our new output directory of symlinks.

type packagingInfo struct {
	// The `Path` converted to a string for the input.  Generally an intermediate file.
	Target string `json:"target"`
	// Relative pathname under testcases where the file should go.
	Placement string `json:"placement"`
}

type manifest struct {
	Name   string `json:"name"`
	Config string `json:"config"`
	// Sorted list of files to expect in output dir.
	Files []string `json:"files"`
}

func (t *testSuiteModule) generateInstallsFromSource(ctx android.ModuleContext,
	suiteName string,
	moduleName string,
	provider tradefed.BaseTestProviderData,
	moduleDir string,
	man *manifest,
	dstDir string,
	whereToWriteSymlink android.InstallPath) []packagingInfo {
	packing := []packagingInfo{}
	if provider.OutputFile != nil {
		man.Files = append(man.Files, moduleDir+provider.OutputFile.Rel())
		packing = append(packing, packagingInfo{
			Target:    provider.OutputFile.String(),
			Placement: dstDir + provider.OutputFile.Rel()})
		// Not needed?
		installed := ctx.InstallAbsoluteSymlink(whereToWriteSymlink, provider.OutputFile.Rel(), relativeSymlink(provider.OutputFile))
		ctx.Phony(suiteName, installed)
	}
	if provider.TestConfig != nil {
		// Remove "test_config_fixer" part of path with Base()
		man.Files = append(man.Files, moduleDir+provider.TestConfig.Base())
		packing = append(packing, packagingInfo{Target: provider.TestConfig.String(),
			Placement: dstDir + provider.TestConfig.Base()})
		installed := ctx.InstallAbsoluteSymlink(whereToWriteSymlink, provider.TestConfig.Base(), relativeSymlink(provider.TestConfig))
		ctx.Phony(suiteName, installed)
	}

	for _, f := range provider.InstalledFiles {
		//artifactRelativePaths = append(artifactRelativePaths, moduleDir+f.Rel())
		man.Files = append(man.Files, moduleDir+f.Rel())
		// TODO(ron): maybe have src here be a Path since we don't serialize it.
		packing = append(packing, packagingInfo{Target: f.String(), Placement: dstDir + "/" + f.Rel()})
		installed := ctx.InstallAbsoluteSymlink(whereToWriteSymlink, f.Rel(), relativeSymlink(f))
		ctx.Phony(suiteName, installed)
	}
	ctx.Phony(suiteName, android.PathForPhony(ctx, moduleName))
	return packing
}

func (t *testSuiteModule) generateInstallsFromTestcases(ctx android.ModuleContext,
	suiteName string,
	moduleName string,
	provider tradefed.BaseTestProviderData,
	moduleDir string,
	man *manifest,
	dstDir string,
	whereToWriteSymlink android.InstallPath) []packagingInfo {
	packing := []packagingInfo{}
	if provider.OutputFile != nil {
		man.Files = append(man.Files, moduleDir+provider.OutputFile.Rel())
		packing = append(packing, packagingInfo{
			Target:    provider.OutputFile.String(),
			Placement: dstDir + provider.OutputFile.Rel()})
		// Not needed?
		installed := ctx.InstallAbsoluteSymlink(whereToWriteSymlink, provider.OutputFile.Rel(), relativeSymlink(provider.OutputFile))
		ctx.Phony(suiteName, installed)
	}
	if provider.TestConfig != nil {
		// Remove "test_config_fixer" part of path with Base()
		man.Files = append(man.Files, moduleDir+provider.TestConfig.Base())
		packing = append(packing, packagingInfo{Target: provider.TestConfig.String(),
			Placement: dstDir + provider.TestConfig.Base()})
		installed := ctx.InstallAbsoluteSymlink(whereToWriteSymlink, provider.TestConfig.Base(), relativeSymlink(provider.TestConfig))
		ctx.Phony(suiteName, installed)
	}

	for _, f := range provider.InstalledFiles {
		//artifactRelativePaths = append(artifactRelativePaths, moduleDir+f.Rel())
		man.Files = append(man.Files, moduleDir+f.Rel())
		// TODO(ron): maybe have src here be a Path since we don't serialize it.
		packing = append(packing, packagingInfo{Target: f.String(), Placement: dstDir + "/" + f.Rel()})
		installed := ctx.InstallAbsoluteSymlink(whereToWriteSymlink, f.Rel(), relativeSymlink(f))
		ctx.Phony(suiteName, installed)
	}
	ctx.Phony(suiteName, android.PathForPhony(ctx, moduleName))
	return packing
}

// Create rules to writes config files and modules inside the testsuite directory
// suite -> cfg
//
//	cfg.Srcs
//	cfg -> Deps  (needed to add dependency earlier)
func (t *testSuiteModule) setupConfigModulesAndFile(ctx android.ModuleContext, suiteName string, suiteRoot android.InstallPath) {
	ctx.WalkDeps(func(child, parent android.Module) bool {
		if _, ok := ctx.OtherModuleDependencyTag(child).(configTag); !ok {
			return false
		}
		if provider, ok := android.OtherModuleProvider(ctx, child, java.JavaInfoProvider); ok {
			for _, jar := range provider.ImplementationJars {
				ifl := ctx.InstallFile(suiteRoot.Join(ctx, "config/modules"), jar.Base(), jar)
				ctx.Phony(suiteName, ifl)
			}
			return false
		}
		// At the config module,  write the sources and continue walk down to its Deps
		for _, src := range child.(*testConfigModule).Srcs {
			ifl := ctx.InstallFile(suiteRoot.Join(ctx, "config"), src, android.PathForSource(ctx, ctx.ModuleDir()).Join(ctx, src))
			ctx.Phony(suiteName, ifl)
		}

		return true
	})
}

func (t *testSuiteModule) linkTestDirs(ctx android.ModuleContext,
	suiteName string,
	moduleName string,
	provider tradefed.BaseTestProviderData,
	moduleDir string,
	man *manifest,
	dstDir string,
	whereToWriteSymlink android.InstallPath) []packagingInfo {
	packing := []packagingInfo{}

	linkTarget := provider.TestcaseDir
	hostOrTarget := "target"
	if provider.IsHost {
		hostOrTarget = "host"
	}
	whereToWriteSymlink = android.PathForModuleInstall(ctx, "suites", suiteName, hostOrTarget, "testcases")

	installed := ctx.InstallAbsoluteSymlink(whereToWriteSymlink, moduleName, relativeSymlinkTestcases(linkTarget))
	ctx.Phony(suiteName, installed)
	ctx.Phony(suiteName, android.PathForPhony(ctx, moduleName))
	return packing
}

// TODO(ron): do I need the .Phony target on the "installed" files in addition to the module name?

func (t *testSuiteModule) manifestOrZip(ctx android.ModuleContext) {
	// For a given test_suite,
	//   For each suite tag that is part of the suite
	//      Write the module and artifacts
	suite := ctx.ModuleName()
	outputPath := android.PathForOutput(ctx, suiteDirectory, suite, fmt.Sprintf(manifestFileName, suite))
	suiteModules := []android.Module{}
	moduleNames := map[string]bool{}
	moduleProviders := map[string]tradefed.BaseTestProviderData{}

	// TODO(ron): implement AND of tags by ensuring module lists all tags for modules not listed directly
	// Iterate both Tests and Tags

	// ctx.VisitDirectDepsWithTag(tradefed.TestSuiteTag{}, func(dep android.Module) {
	ctx.WalkDeps(func(child, parent android.Module) bool {
		// Descend on level to the "Tag"
		if _, ok := child.(*testModuleTagModule); ok {
			return true
		}
		// Skip config module
		if _, ok := child.(*testConfigModule); ok {
			return false
		}

		dep := child
		suiteModules = append(suiteModules, dep)
		moduleNames[dep.Name()] = true
		if provider, ok := android.OtherModuleProvider(ctx, dep, tradefed.BaseTestProviderKey); ok {
			moduleProviders[dep.Name()] = provider
		} else {
			ctx.ModuleErrorf("Dep: %s is not a test provider.", dep)
		}
		return false
	})

	// manifest
	allFiles := map[string][]packagingInfo{}
	man := &manifest{Name: suite}
	suiteRoot := android.PathForModuleInstall(ctx, "suites", suite)

	for moduleName, _ := range moduleNames {
		provider := moduleProviders[moduleName]
		hostOrTarget := "target"
		if provider.IsHost {
			hostOrTarget = "host"
		}
		moduleDir := fmt.Sprintf("%s/testcases/%s/",
			hostOrTarget,
			moduleName)

		//		artifactRelativePaths := []string{}
		// dstDir is path beginnig with "suites/" (under out) where the file should be linked
		// src is the target of the link.
		// suites/SUITE_NAME/host/testcases/MODULE_NAME/FILE_NAME
		dstDir := fmt.Sprintf("%s/%s/%s/testcases/%s/", suiteDirectory, suite, hostOrTarget, moduleName)

		// Perhaps this should be PathForModuleOutput and the output should go under the build tree,
		// but then we can't use InstallSymlink.  It needs an InstallPath to write to.
		// TODO(ron): maybe use PathForArbitraryOutput
		whereToWriteSymlink := android.PathForModuleInstall(ctx, "suites", suite, moduleName)
		// TODO(ron): are dstDir and moduleDir the same?

		//packing := t.generateInstallsFromSource(ctx, suite, moduleName, provider, moduleDir, man, dstDir, whereToWriteSymlink)
		packing := t.linkTestDirs(ctx, suite, moduleName, provider, moduleDir, man, dstDir, whereToWriteSymlink)
		allFiles[moduleName] = packing
	}

	ShouldSymlinkTestDirs := true
	if ShouldSymlinkTestDirs {
		// Create a rule to generate the manifest.
		manifestOutputPath, listOutputPath := t.walkFilesToCreateManifest(ctx, suiteRoot)

		// TODO(ron); need phony?
		t.zipOutputPath = t.zipSuiteDir(ctx, suiteRoot, listOutputPath)
		ctx.Phony(suite+".zip", t.zipOutputPath, android.PathForPhony(ctx, suite))

		// Add it our outputs and also install it in our test suite output
		t.outputFiles = append(t.outputFiles, manifestOutputPath, listOutputPath, t.zipOutputPath)
	}
	t.setupConfigModulesAndFile(ctx, suite, suiteRoot)

	sort.Strings(man.Files)

	// Write the output to a manifest file based on test_suite module name.
	data, err := json.Marshal(man)
	if err != nil {
		ctx.ModuleErrorf("Unable to marshal suite data. %s", err)
	}

	android.WriteFileRuleVerbatim(ctx, outputPath, string(data))

	data, err = json.Marshal(allFiles)
	if err != nil {
		ctx.ModuleErrorf("Unable to marshal suite data. %s", err)
	}

	// Should I run one rule to generate all my symlinks given a json file of everything we need,
	// or should I generate one symlink rule per output?
	// Or call InstallSymlink, InstallAbsoluteSymlink?
	packingOptionsPath := android.PathForModuleOut(ctx, "packing.json")
	// TODO(ron): sort if we write the output
	android.WriteFileRuleVerbatim(ctx, packingOptionsPath, string(data))

	ctx.Phony(suite, outputPath)
	// TODO(ron): add deps on provider.InstalledFiles
	//ctx.Phony(suite, allSuiteDeps...)
	t.outputFiles = append(t.outputFiles, outputPath, packingOptionsPath.OutputPath)
}

func (m *testSuiteModule) walkFilesToCreateManifest(ctx android.ModuleContext, suiteInstallDir android.InstallPath) (android.OutputPath, android.OutputPath) {
	// Test safe to do when no test_runner_options, but check for that earlier?
	// manifestJson := suiteInstallDir.Join(ctx, "manifest.json")
	manifestJson := android.PathForModuleOut(ctx, "manifest.json")
	manifestList := android.PathForModuleOut(ctx, "manifest.filelist")
	rule := android.NewRuleBuilder(pctx, ctx)
	rule.Command().BuiltTool("suite_manifest").
		FlagWithInput("--suite_dir ", suiteInstallDir).
		FlagWithOutput("--output_json ", manifestJson).
		FlagWithOutput("--output_list ", manifestList)

	rule.Build("suite_manifest", fmt.Sprintf("generate suite manifest for %s", suiteInstallDir.Base()))
	return manifestJson.OutputPath, manifestList.OutputPath
}

// If we zip the the suite dir, we have two levels of symlinks, one for the module dir and one for the test_module_config
// back to base.  We want to keep the second symlink, but not the first (as it points out of the zip root).
// So we need to pass a list of files zip to soong_zip and then tell it no symlinks.
// We can generate this ourselves, or we can read it from the output of the manifest generator.
func (m *testSuiteModule) zipSuiteDir(ctx android.ModuleContext, suiteDir android.InstallPath, listOfFiles android.Path) android.OutputPath {
	suiteName := ctx.ModuleName()
	suiteZip := android.PathForModuleOut(ctx, fmt.Sprintf("%s.zip", suiteName))
	rule := android.NewRuleBuilder(pctx, ctx)
	// -C out/host/-/suites/the-suite-life -D  out/host/-/suites/the-suite-life -d -o test.zip -symlinks=false
	rule.Command().BuiltTool("soong_zip").
		Flag("-symlinks=true").
		Flag("-d").
		FlagWithArg("-C ", suiteDir.String()).
		FlagWithInput("-l ", listOfFiles).
		FlagWithOutput("-o ", suiteZip)

	rule.Build("suite_zip", fmt.Sprintf("generate suite zip for %s", suiteDir.Base()))
	return suiteZip.OutputPath
}

func relativeSymlink(src android.Path) string {
	// Backup the depth of where we write
	//backup := strings.Repeat("../", strings.Count(targetFileName, "/")+1)
	// backup := "../../../../../.." + strings.Repeat("/..", strings.Count(src.Rel(), "/"))
	backup := "../../../../../.." + strings.Repeat("/..", strings.Count(src.Rel(), "/"))

	return fmt.Sprintf("%s/%s", backup, src.String())
}

func relativeSymlinkTestcases(src android.Path) string {
	// Backup the depth of where we write
	//backup := strings.Repeat("../", strings.Count(targetFileName, "/")+1)
	// out/host/-/suites/SUITE_NAME
	backup := "../../../../../../.."

	return fmt.Sprintf("%s/%s", backup, src.String())
}

/*
BUGS:


   % tree -ls  out/host/-/suites                                                                                                                                                                                                                                   aosp_cf_x86_64_phone[2:15:46]/0
[       4096]  out/host/-/suites
└── [       4096]  the-suite-life
    └── [       4096]  FrameworksServicesTests_om
        ├── [       4096]  data
        │   └── [        113]  broken_shortcut.xml -> ../../../../../../../out/target/product/vsoc_x86_64/testcases/FrameworksServicesTests_om/data/broken_shortcut.xml
        ├── [        154]  FrameworksServicesTests.apk -> ../../../../../../out/soong/.intermediates/frameworks/base/services/tests/servicestests/FrameworksServicesTests/android_common/FrameworksServicesTests.apk
        ├── [        128]  MediaButtonReceiverHolderTestHelperApp.apk -> ../../../../../../out/target/product/vsoc_x86_64/testcases/FrameworksServicesTests_om/MediaButtonReceiverHolderTestHelperApp.apk
        ├── [        111]  SimpleServiceTestApp1.apk -> ../../../../../../out/target/product/vsoc_x86_64/testcases/FrameworksServicesTests_om/SimpleServiceTestApp1.apk
        ├── [        111]  SimpleServiceTestApp2.apk -> ../../../../../../out/target/product/vsoc_x86_64/testcases/FrameworksServicesTests_om/SimpleServiceTestApp2.apk
        ├── [        111]  SimpleServiceTestApp3.apk -> ../../../../../../out/target/product/vsoc_x86_64/testcases/FrameworksServicesTests_om/SimpleServiceTestApp3.apk
        ├── [        104]  SuspendTestApp.apk -> ../../../../../../out/target/product/vsoc_x86_64/testcases/FrameworksServicesTests_om/SuspendTestApp.apk
        └── [       4096]  x86_64
            └── [        123]  FrameworksServicesTests.apk -> ../../../../../../../out/target/product/vsoc_x86_64/testcases/FrameworksServicesTests_om/x86_64/FrameworksServicesTests.apk


 FIXED  1) Need to ensure modules are built.
   2) Missing config
      // src doesn't have config file :(
      // Need from testcases?  read from androidmk provider?
 {obsolete}  3) extra FrameworksServicesTests.apk
       InstalledFiles has the x86_64 one for FrameworksServicesTests
       OuputFile has the android/common one.
       oh Installed files points to testcases for OM
        (double check for test_module_config entries)?

 FIXED   4) dependency "CtsDevicePolicyManagerTestCases" of "the-suite-life" missing variant
   5) Updating Android.bp doesn't update .zip (i.e. added HelloWorldHostTest)
   6) Need a "dist" goal so builds can write zips somewhere?
      func (t *allTeamsSingleton) MakeVars(ctx MakeVarsContext) {
     	ctx.DistForGoal("all_teams", t.outputPath)
      }
   7) tests where we really set the variations correctly (i.e. it fails without by other-variation check)

   These don't update when the the-suite updates. Can they depend on the phony suite target?

% ls -l out/soong/.intermediates/platform_testing/tests/example/devcodelab/the-suite-life/                                                                                                                      aosp_cf_x86_64_phone[9:40:15]/0
total 47580
-rw-r--r-- 1 rbraunstein primarygroup     2141 Sep 19 21:26 manifest.filelist
-rw-r--r-- 1 rbraunstein primarygroup     1703 Sep 19 21:26 manifest.json
-rw-r--r-- 1 rbraunstein primarygroup 48706248 Sep 19 21:26 the-suite-life.zip


   ERRS if I do this:
% rm -rf out/host/-/suites/the-suite-life/                                                                                                                                                                      aosp_cf_x86_64_phone[9:56:43]/0
~/aosp-main-with-phones
 % m the-suite-life.zip

     8) Add Android.mk entries so we can see where the output file goes.
*/
