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
	"path/filepath"

	"android/soong/android"
	// "android/soong/java"
	"android/soong/tradefed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/google/blueprint"
)

/*

   The test_suite rule packages up files (via symlinks) so a "suite" directory contains all the files
   required to run all the tests defined by the suite.
   See: http://goto.google.com/atc-test-suite-packaging for more details.

   Sample output should look like this:
.
└── out/soong/
    ├── packaging/
    │   └── my-team-suite/
    │       ├── my-team-suite.json
    │       ├── host/
    │       │   └── testcases
    │       │       └── MyTeamTest1 [ *** ]
    │       │           ├── MyTeamTest1.apk -> _path_to_MyTeamTest1.apk_   [ ** ]
    │       │           └── MyTeamTest1.config -> _path_to_MyTeamTest1.config_
    │       └── target
    │           └── testcases
    │               └── MyTeamTest2
    │                   ├── MyTeamTest2.apk -> _path_to_MyTeamTest2.apk_
    │                   └── MyTeamTest2.config -> _path_to_MyTeamTest12.config_

   [**] - the target of the link can be one of several places:

      1) ../../../../../out/soong/.intermediates/.../MyTeamTest1/MyTeamTest.apk
      2) ../../../../../out/soong/targets/testcases/MyTeamTest1/MyTeamTest.apk
   [***] - or we can just symlink to the testcases dir rather than "install" each file
└── out/soong/
    ├── packaging/
    │   └── my-team-suite/
    │       ├── my-team-suite.json
    │       ├── host/
    │       │   └── testcases
    │       │       └── MyTeamTest1 -> out/soong/targets/testcases/MyTeamTest1
    │       └── target
    │           └── testcases
    │               └── MyTeamTest2 -> out/soong/host/testcases/MyTeamTest2


   with `my-team-suite.json` containing:
{
  "name": "my-team-suite",
  "version": "1.0",
  "files": [
      "host/testcases/MyTeamTest1/MyTeamTest1.config",
      "host/testcases/MyTeamTest1/MyTeamTest1.apk",
      "target/testcases/MyTeamTest2/MyTeamTest2.config",
      "target/testcases/MyTeamTest2/MyTeamTest2.apk"
  ]
}

   // A test_suite declaration that explicitly lists tests and find tests via tags:
   // Test suites
   test_suite {
     name: "mts-art",
     description: "Runs all ART modules with the Mainline module installed",
     module_filter: {
       tags: [
	 "mts-art-tag",
       ],
     }
   }

   test_module_tag(
     name: "mts-art-tag",
     description: "Identifies *all* ART tests",
     visibility: [":__subpackages__"],  // Only usable by ART.
   )

   // And nested test suites:
   test_suite {
     name: "art-platform-presubmit",
     description: "Runs all ART modules tagged `presubmit` without installing the
		   Mainline module",
     // `tests` can refer to test module names or other test_suite names.
     tests: [
       "mts-art",
     ],
     module_filter: {
       tags: [
	   "art:presubmit",
       ],
     }
   }


*/

// Directory inside of out/soong to store suite symlink forrests and manifests
const suiteDirectory = "suites"

// const manifestFileName = "suite.%s.manifest.json"
const manifestFileName = "%s.json"
const zipFileName = "%s.zip"

func init() {
	RegisterTestSuiteBuildComponents(android.InitRegistrationContext)
}

func RegisterTestSuiteBuildComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("test_suite", TestSuiteFactory)
	ctx.RegisterModuleType("test_module_tag", TestModuleTagFactory)
	ctx.RegisterModuleType("tradefed_xml_config", TestConfigFactory)
	// TODO(rbraunstein): config_includes
}

var PrepareForTestWithTestSuiteBuildComponents = android.GroupFixturePreparers(
	android.FixtureRegisterWithContext(RegisterTestSuiteBuildComponents),
)

type testSuiteProperties struct {
	Description   string
	Tests         []string `android:"path,arch_variant"`
	Module_filter struct {
		Tags []string `android:"path"` // test_module_tag module
	}
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
	android.ModuleBase
	android.DefaultableModuleBase
	testSuiteProperties

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
	t.manifestOrZip(ctx)
}

//func (t *testModuleGroup) GenerateAndroidBuildActions(ctx android.ModuleContext) {
// TODO(rbraunstein): Visit all deps and ensure we list all tags, other field err
//     i.e. we are doing an "AND", not an "OR"
// TODO(rbraunstein): validate that Module.Modules are valid or in AddDep
//     I think that is already done for us.
// TODO(rbraunstein):
//    Ensure that if people uses Tests: and list a test_module_config entry, but
//    the base isn't listed, then we get an error.
//    Or somehow fix the symlinks to copy instead of link in this case.
//}

func (t *testModuleTagModule) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	// Nothing to do.
}

// Add our direct "tests:" deps here.
// The "tags:" deps are added in providers with AddReverseDependency
// Or do we need to register a top down mutator to add it ourselves?
func (t *testSuiteModule) DepsMutator(ctx android.BottomUpMutatorContext) {
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
	for _, tag := range t.Module_filter.Tags {
		ctx.AddDependency(ctx.Module(), tradefed.TestSuiteTag{}, tag)
	}
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
	module.AddProperties(&module.testSuiteProperties)

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

type manifest struct {
	Name   string `json:"name"`
	Config string `json:"config"`
	// Sorted list of files to expect in output dir.
	Files []string `json:"files"`
}

func (t *testSuiteModule) generateInstallsFromSource(ctx android.ModuleContext,
	suiteName string,
	provider tradefed.BaseTestProviderData,
	moduleDir string,
	man *manifest,
	whereToWriteSymlink android.InstallPath,
	moduleName string) {
	var installed android.InstallPath
	if provider.OutputFile != nil {
		// TODO(rbraunstein): should output apk go in arch dir, like it does in testcases?
		man.Files = append(man.Files, moduleDir+provider.OutputFile.Rel())
		// HACK:
		if strings.HasPrefix(provider.OutputFile.String(), "out/") {
			installed = installSymlink(ctx, whereToWriteSymlink, provider.OutputFile.Rel(), provider.OutputFile)
		} else {
			installed = ctx.InstallFile(whereToWriteSymlink, provider.OutputFile.Rel(), provider.OutputFile)
		}

		ctx.Phony(suiteName, installed)
	}
	if provider.TestConfig != nil {
		// TODO(rbraunstein): fix providers to give us the name "MyModule.config" file, not AndroidTest.xml
		// This is still not quite right for "HelloWorldTests", which writes this to its testcases dir.
		// It has its own "test_config" entry and an extra config file:
		// % tree -ls out/target/product/vsoc_x86_64/testcases/HelloWorldTests
		// [       4096]  out/target/product/vsoc_x86_64/testcases/HelloWorldTests
		// ├── [       1665]  hallo-welt.config
		// ├── [       1527]  HelloWorldTests.config
		// └── [       4096]  x86_64
		//     └── [     968473]  HelloWorldTests.apk

		tradefedConfig := provider.TestConfig.Base()
		if tradefedConfig == "AndroidTest.xml" {
			tradefedConfig = moduleName + ".config"
		}
		// Remove "test_config_fixer" part of path with Base()
		man.Files = append(man.Files, moduleDir+tradefedConfig)
		var installed android.InstallPath
		if false {
			installed = installSymlink(ctx, whereToWriteSymlink, provider.TestConfig.Base(), provider.TestConfig)
		} else {
			installed = ctx.InstallFile(whereToWriteSymlink, tradefedConfig, provider.TestConfig)
		}
		ctx.Phony(suiteName, installed)
	}

	for _, f := range provider.InstalledFiles {
		man.Files = append(man.Files, moduleDir+f.Rel())
		// TODO(ron): maybe have src here be a Path since we don't serialize it.
		// HACK: link to testcase, copy from .intermediates
		if strings.HasPrefix(f.String(), "out/") {
			installed = installSymlink(ctx, whereToWriteSymlink, f.Rel(), f)
		} else {
			installed = ctx.InstallFile(whereToWriteSymlink, f.Rel(), f)
		}
		ctx.Phony(suiteName, installed)
	}
}

// Create rules to writes config files and modules inside the testsuite directory
// suite -> cfg
//
//	cfg.Srcs
//	cfg -> Deps  (needed to add dependency earlier)
func (t *testSuiteModule) setupConfigModulesAndFile(ctx android.ModuleContext, suiteName string, suiteRoot android.InstallPath, manifest *manifest, configRoot android.OutputPath) {
	// d := t.mkdir(ctx, configRoot.Join(ctx, "modules"))
	// ctx.WalkDeps(func(child, parent android.Module) bool {
	// 	if _, ok := ctx.OtherModuleDependencyTag(child).(configTag); !ok {
	// 		return false
	// 	}
	// 	if provider, ok := android.OtherModuleProvider(ctx, child, java.JavaInfoProvider); ok {
	// 		for _, jar := range provider.ImplementationJars {
	// 			manifest.Files = append(manifest.Files, fmt.Sprintf("config/modules/%s", jar.Base()))
	// 			ifl := ctx.InstallFile(suiteRoot.Join(ctx, "config/modules"), jar.Base(), jar)
	// 			// ctx.Phony(suiteName, ifl)
	// 			copiedFile := t.cpConfig(ctx, ifl, configRoot.Join(ctx, "modules", jar.Base()))
	// 			// ctx.Phony(suiteName+".config", copiedFile, d)
	// 		}
	// 		return false
	// 	}
	// 	// At the config module,  write the sources and continue walk down to its Deps
	// 	for _, src := range child.(*testConfigModule).Srcs {
	// 		manifest.Files = append(manifest.Files, fmt.Sprintf("config/%s", src))
	// 		ifl := ctx.InstallFile(suiteRoot.Join(ctx, "config"), src, android.PathForSource(ctx, ctx.ModuleDir()).Join(ctx, src))
	// 		// ctx.Phony(suiteName, ifl)
	// 		copiedFile := t.cpConfig(ctx, ifl, configRoot.Join(ctx, src))
	// 		// ctx.Phony(suiteName+".config", copiedFile, d)
	// 	}

	// 	return true
	// })
	// ctx.Phony(suiteName, android.PathForPhony(ctx, suiteName+".config"))
	ctx.Phony(suiteName)
}

// TODO(ron): do I need the .Phony target on the "installed" files in addition to the module name?

func (t *testSuiteModule) manifestOrZip(ctx android.ModuleContext) {
	// For a given test_suite,
	//   For each suite tag that is part of the suite
	//      Write the module and artifacts
	suite := ctx.ModuleName()
	manifestInstallPath := android.PathForSuiteInstall(ctx, suite, fmt.Sprintf(manifestFileName, suite))
	zipInstallPath := android.PathForSuiteInstall(ctx, suite, fmt.Sprintf(zipFileName, suite))
	configOutputPath := android.PathForOutput(ctx, suiteDirectory, suite, "config")
	listOutputPath := android.PathForModuleOut(ctx, "manifest.filelist")
	suiteModules := []android.Module{}
	moduleNames := map[string]bool{}
	moduleProviders := map[string]tradefed.BaseTestProviderData{}

	ctx.WalkDeps(func(child, parent android.Module) bool {
		// Descend on level to the "Tag"
		if _, ok := child.(*testModuleTagModule); ok {
			return true
		}
		// Skip config module
		if _, ok := child.(*testConfigModule); ok {
			return false
		}

		// Only write out test suite dependencies here.
		if _, ok := ctx.OtherModuleDependencyTag(child).(tradefed.TestSuiteTag); !ok {
			return false
		}

		dep := child
		suiteModules = append(suiteModules, dep)
		moduleNames[dep.Name()] = true
		if provider, ok := android.OtherModuleProvider(ctx, dep, tradefed.BaseTestProviderKey); ok {
			moduleProviders[dep.Name()] = provider
		} else {
			ctx.ModuleErrorf("Dep: %+v, of %s is not a test provider. (%+v):", child, ctx.OtherModuleType(child), parent)
		}
		return false
	})

	// manifest
	man := &manifest{Name: suite}
	suiteRoot := android.PathForSuiteInstall(ctx, suite)

	// Create rules to install config files and add them to the manifest.
	t.setupConfigModulesAndFile(ctx, suite, suiteRoot, man, configOutputPath)

	for moduleName, _ := range moduleNames {
		provider := moduleProviders[moduleName]
		hostOrTarget := "target"
		if provider.IsHost {
			hostOrTarget = "host"
		}
		moduleDir := fmt.Sprintf("%s/testcases/%s/",
			hostOrTarget,
			moduleName)

		// Perhaps this should be PathForModuleOutput and the output should go under the build tree,
		// but then we can't use InstallSymlink.  It needs an InstallPath to write to.
		// TODO(ron): maybe use PathForArbitraryOutput
		suiteModuleDir := suiteRoot.Join(ctx, hostOrTarget, "testcases", moduleName)
		// TODO(ron): are dstDir and moduleDir the same?

		t.generateInstallsFromSource(ctx, suite, provider, moduleDir, man, suiteModuleDir, moduleName)
		// The zip requires that each module is built.
		ctx.Phony(suite, android.PathForPhony(ctx, moduleName))
	}

	// Write manifest file directly from our knowledge of artifacts of what should be there, not what is there.
	sort.Strings(man.Files)
	data, err := json.Marshal(man)
	if err != nil {
		ctx.ModuleErrorf("Unable to marshal suite manifest file. %s", err)
	}
	android.WriteFileRule(ctx, manifestInstallPath, string(data))
	filesToZip := []string{}
	for _, f := range man.Files {
		filesToZip = append(filesToZip, suiteRoot.Join(ctx, f).String())
	}
	android.WriteFileRule(ctx, listOutputPath, strings.Join(filesToZip, "\n"))

	// Rule to create zip file from manifest
	t.zipSuiteDir(ctx, suiteRoot, zipInstallPath)
	// TODO(ron): add each module and config mod and config src as dep for the zip as well.
	ctx.Phony(suite, manifestInstallPath)
	ctx.Phony(suite, android.PathForPhony(ctx, suite))

	// Add it our outputs and also install it in our test suite output
	t.outputFiles = append(t.outputFiles, listOutputPath.OutputPath)

	// ctx.Phony(suite, manifestInstallPath, android.PathForPhony(ctx, suite+".config"))
}

// If we zip the the suite dir, we have two levels of symlinks, one for the module dir and one for the test_module_config
// back to base.  We want to keep the second symlink, but not the first (as it points out of the zip root).
// So we need to pass a list of files zip to soong_zip and then tell it no symlinks.
// We can generate this ourselves, or we can read it from the output of the manifest generator.
func (m *testSuiteModule) zipSuiteDir(ctx android.ModuleContext, suiteDir android.InstallPath, suiteZip android.InstallPath) {
	rule := android.NewRuleBuilder(pctx, ctx)
	// -C out/host/-/suites/the-suite-life -D  out/host/-/suites/the-suite-life -d -o test.zip -symlinks=false
	rule.Command().BuiltTool("soong_zip").
		Flag("-symlinks=true").
		Flag("-d").
		FlagWithArg("-C ", suiteDir.String()).
		FlagWithArg("-D ", suiteDir.String()).
		FlagWithOutput("-o ", suiteZip)

	rule.Build("suite_zip", fmt.Sprintf("generate suite zip for %s", suiteDir.Base()))
}

func (m *testSuiteModule) cpConfig(ctx android.ModuleContext, suiteFile android.InstallPath, dst android.WritablePath) android.WritablePath {
	rule := android.NewRuleBuilder(pctx, ctx)
	// -C out/host/-/suites/the-suite-life -D  out/host/-/suites/the-suite-life -d -o test.zip -symlinks=false
	rule.Command().Text("cp").
		Input(suiteFile).
		Output(dst)

	// TODO(ron): cp -r once instead and collect all the deps? i.e.e one rule.
	rule.Build(fmt.Sprintf("cp_config_file %s", suiteFile.Base()), fmt.Sprintf("Copying suite configs  %s", suiteFile))
	return dst
}

func (m *testSuiteModule) mkdir(ctx android.ModuleContext, dst android.WritablePath) android.WritablePath {
	rule := android.NewRuleBuilder(pctx, ctx)
	rule.Command().Text("rm").Flag("-rf").Text(dst.String())
	rule.Command().Text("mkdir").Flag("-p").Output(dst)

	// TODO(ron): cp -r once instead and collect all the deps? i.e.e one rule.
	rule.Build("mkdir_config_dir", fmt.Sprintf("Create config output dir"))
	return dst
}

func installSymlink(ctx android.ModuleContext, src android.InstallPath, name string, dst android.Path) android.InstallPath {
	fmt.Printf("Installing symlink from %s to %s named %s\n", src.String(), dst.String(), name)
	r, err := filepath.Rel(filepath.Join(src.String(), filepath.Dir(name)), dst.String())
	if err != nil {
		ctx.ModuleErrorf("Unable to create symlink from %s to %s: %s", src.String(), dst.String(), err)
	}
	return ctx.InstallAbsoluteSymlink(src, name, r)
}

/*
BUGS:


   % tree -ls  out/host/-/suites
[       4096]  out/host/-/suites
└── [       4096]  the-suite-life
    └── [       4096]  FrameworksServicesTests_om
        ├── [       4096]  data
        │   └── [        113]  broken_shortcut.xml -> ../../../../../../../out/target/product/vsoc_x86_64/testcases/FrameworksServicesaTests_om/data/broken_shortcut.xml
        ├── [        154]  FrameworksServicesTests.apk -> ../../../../../../out/soong/.intermediates/frameworks/base/services/tests/servicestests/FrameworksServicesTests/android_common/FrameworksServicesTests.apk
        ├── [        128]  MediaButtonReceiverHolderTestHelperApp.apk -> ../../../../../../out/target/product/vsoc_x86_64/testcases/FrameworksServicesTests_om/MediaButtonReceiverHolderTestHelperApp.apk
        ├── [        111]  SimpleServiceTestApp1.apk -> ../../../../../../out/target/product/vsoc_x86_64/testcases/FrameworksServicesTests_om/SimpleServiceTestApp1.apk
        ├── [        111]  SimpleServiceTestApp2.apk -> ../../../../../../out/target/product/vsoc_x86_64/testcases/FrameworksServicesTests_om/SimpleServiceTestApp2.apk
        ├── [        111]  SimpleServiceTestApp3.apk -> ../../../../../../out/target/product/vsoc_x86_64/testcases/FrameworksServicesTests_om/SimpleServiceTestApp3.apk
        ├── [        104]  SuspendTestApp.apk -> ../../../../../../out/target/product/vsoc_x86_64/testcases/FrameworksServicesTests_om/SuspendTestApp.apk
        └── [       4096]  x86_64
            └── [        123]  FrameworksServicesTests.apk -> ../../../../../../../out/target/product/vsoc_x86_64/testcases/FrameworksServicesTests_om/x86_64/FrameworksServicesTests.apk


   1) tests where we really set the variations correctly (i.e. it fails without by other-variation check)

   These don't update when the the-suite updates. Can they depend on the phony suite target?

   2) Add Android.mk entries so we can see where the output file goes via module-info.json?
*/
