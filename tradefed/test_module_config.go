package tradefed

import (
	"android/soong/android"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

// 0) concepts: depend on "stuff in dir" or ask "Base"
// 1) in tradefed dir or android dir
// 2) All the java bits are private, and things get written out via app_prebuilt_internal.mk

// Or fixup the install path in IDEInfo to be correct?
type TestConfigProvider interface {
	TestConfig() android.Path
	ExtraTestConfigs() android.Paths
}

// Some tests are shared between rules, where the binaries for the tests
// are exactly the same, but the options for the test differ.
// This interface allows differernt test types to publicize where the
// binary parts are (like data apks for android_tests)
type SharedTest interface {
	// Return paths to the base tests' installed/testcases dir.
	// I guess it doesn't have to be installed tree if we are resolving when zipping on the build.
	InstalledFiles() android.Paths
	// OuputFiles doesn't do what ne need either.
	OutputFile() android.Path
}

func init() {
	RegisterTestModuleConfigBuildComponents(android.InitRegistrationContext)
}

// Register the license_kind module type.
func RegisterTestModuleConfigBuildComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("test_module_config", TestModuleConfigFactory)
}

type testModuleConfigModule struct {
	android.ModuleBase
	android.DefaultableModuleBase

	base android.Module // Or ModuleBase?
	tfProperties
	// Do we need to declare our outs?
	// Using InstallPath gave me missing package.apk
	output android.OutputPath
	// Our updated one.
	testConfig android.OutputPath
}

type testBase struct {
	android.ModuleBase
	android.AndroidMkEntriesProvider
}

type tfProperties struct {
	Base    *string `android:"path,arch_variant"`
	Options []Option
}

type dependencyTag struct {
	blueprint.BaseDependencyTag
	name string
}

var testModuleConfigTag = dependencyTag{name: "TestModuleConfigBase"}

// Do I need an InstallForceOS too?
func (m *testModuleConfigModule) InstallInTestcases() bool {
	// TODO(ron): should depend on base, right?
	// This should work for host tests too.
	return true
}

func (m *testModuleConfigModule) DepsMutator(ctx android.BottomUpMutatorContext) {
	// Todo(ron); make tag, use variant dependency?
	ctx.AddDependency(ctx.Module(), testModuleConfigTag, *m.Base)
}

func (a *testModuleConfigModule) fixTestConfig(ctx android.ModuleContext, testConfig android.Path) android.OutputPath {
	// Test safe to do when no test_runner_options, but check for that earlier?
	fixedConfig := android.PathForModuleOut(ctx, "test_config_fixer", ctx.ModuleName()+".config")
	rule := android.NewRuleBuilder(pctx, ctx)
	command := rule.Command().BuiltTool("test_config_fixer").Input(testConfig).Output(fixedConfig)
	xmlTestModuleConfignerSnippet, _ := json.Marshal(a.tfProperties.Options)
	escaped := proptools.NinjaAndShellEscape(string(xmlTestModuleConfignerSnippet))
	command.Flag("--test-runner-options ").Text(escaped)
	rule.Build("fix_test_config", "fix test config")
	return fixedConfig.OutputPath
}

func (m *testModuleConfigModule) GenerateAndroidBuildActions(ctx android.ModuleContext) {

	if "hi" == "hi" {
		m.oldGenerateAndroidBuildActions(ctx)
		return
	} else {
		m.newGenerateAndroidBuildActions(ctx)
		return
	}
}
func (m *testModuleConfigModule) newGenerateAndroidBuildActions(ctx android.ModuleContext) {

	var installDir android.InstallPath
	if ctx.InstallInTestcases() {
		installDir = android.PathForModuleInstall(ctx, ctx.ModuleName())
	} else {
		// TODO(ron): verify on java_host_test or something.
		installDir = android.PathForModuleInstall(ctx, "framework")
	}

	ctx.VisitDirectDepsWithTag(testModuleConfigTag, func(base android.Module) {
		fmt.Printf("BASE FTI: %+v\n", base.FilesToInstall())
		// have: out/target/product/vsoc_x86_64/testcases/FrameworksServicesTests/x86_64/FrameworksServicesTests.apk
		// want: out/target/product/vsoc_x86_64/testcases/FrameworksServicesTests/x86_64/FrameworksServicesTests.apk

		if st, ok := base.(SharedTest); ok {
			for _, f := range st.InstalledFiles() {
				fmt.Printf("BASE TIF: %+v\n", f)
			}
		}

		m.base = base
		if provider, ok := base.(TestConfigProvider); ok {
			fmt.Printf("TestConfig found: %s  %v\n", ctx.ModuleName(), provider.TestConfig())
			// TODO(ron): we let the .mk file write the AndroidTestConfig after we save its location here
			// and emit in AndroidMkEntries.
			m.testConfig = m.fixTestConfig(ctx, provider.TestConfig())
			// Adding InstallFile back in to get symlinks? Seems to help, but it doesn't show up?
			// YES, this is really needed.
			ctx.InstallFile(installDir, m.testConfig.Base()+"_extra_if", m.testConfig)
		}

	})

}

// TODO(ron): more interfaces
//
//	derivedClass, BaseClassProvider
//	    test_config, and other stuff needed in androidmk
func (m *testModuleConfigModule) oldGenerateAndroidBuildActions(ctx android.ModuleContext) {

	var installDir android.InstallPath
	if ctx.InstallInTestcases() {
		installDir = android.PathForModuleInstall(ctx, ctx.ModuleName())
	} else {
		// TODO(ron): verify on java_host_test or something.
		installDir = android.PathForModuleInstall(ctx, "framework")
	}

	// We already base name, we don't need to visit it.
	// perhaps walk the base properties looking for an installPath?
	ctx.VisitDirectDepsWithTag(testModuleConfigTag, func(base android.Module) {
		/*
			// TODO RON 1<2
			out := android.PathForModuleOut(ctx, "test_module_config.manifest")
			android.WriteFileRule(ctx, out, base.Name())
		*/
		// TODO(ron): FIXME: NOTE: using install file is forcing this be installed as apk too!
		// Still need to get in install dir?
		// For some reason I need this to get all symlinks working.
		// ctx.InstallAbsoluteSymlink(installDir, symlinkName, f.String())
		// 0) base apk.
		fmt.Printf("BASE FTI: %+v\n", base.FilesToInstall())
		fmt.Printf("BASE PackSpec: %+v\n", base.PackagingSpecs())
		fmt.Printf("BASE OutFile: %+v\n", base.(SharedTest).OutputFile())
		// PackagingSpec.srcPath has what I want, but need to be in android to read it.

		out := android.PathForModuleOut(ctx, "test_module_config.manifest")
		ctx.InstallFile(installDir, out.Base(), out)
		android.WriteFileRule(ctx, out, base.Name())
		m.output = out.OutputPath
		// package.apk either comes from reflection on OutputPath or calling OutputFiles()
		// or from InstallFile or ??

		// TODO(ron): Should we install this as base.Name() or ctx.ModuleName() + ".apk"
		// Symlink to base apk
		baseApk := base.(SharedTest).OutputFile()
		symlinkName := baseApk.Base()
		relLink := relativeToHere(installDir, baseApk.String())
		/*out := */ ctx.InstallAbsoluteSymlink(installDir, symlinkName, relLink)

		//		for _, i := range base.FilesToInstall() {
		// Did making this a symlink break the others?

		// Put it in our intermediate dir
		// And then link to that.
		// Fixes problems with no package.apk
		// (does duplicate?)
		// TODO(ron): fix FilesToInstall to not give install dir, but base dir?
		// out := android.PathForModuleOut(ctx
		// ctx.InstallFile(installDir, out.Base(), out)

		// symlinkName := i.Base()
		// relLink := relativeToHere(installDir, i.String())
		// /*out := */ ctx.InstallAbsoluteSymlink(installDir, symlinkName, relLink+"__FTI")

		// m.output = out
		// TOOD(ron): implement OutputFiles, InstallFiles(), or just rely on the reflection and InstallXX
		//		}

		// 1) Manifest file, not needed any more?
		/*
			        // TODO RON 3,4,
				ctx.InstallFile(installDir, out.Base(), out)
				ctx.Build(pctx, android.BuildParams{
					Rule:   android.Cp,
					Input:  pathOnHost,
					Output: out,
				})
		*/

		// 2) TestConfig
		//  !! TODO(ron)! Do we change the test-tag to be our new module name?
		m.base = base
		if provider, ok := base.(TestConfigProvider); ok {
			fmt.Printf("TestConfig found: %s  %v\n", ctx.ModuleName(), provider.TestConfig())
			// TODO(ron): we let the .mk file write the AndroidTestConfig after we save its location here
			// and emit in AndroidMkEntries.
			m.testConfig = m.fixTestConfig(ctx, provider.TestConfig())
			// Adding InstallFile back in to get symlinks? Seems to help, but it doesn't show up?
			// YES, this is really needed.
			ctx.InstallFile(installDir, m.testConfig.Base()+"_extra_if", m.testConfig)
		}

		// 3) deps
		// TODO(ron): In DepsMutator, ensure it can be type asserted
		if st, ok := base.(SharedTest); ok {
			for _, f := range st.InstalledFiles() {
				symlinkName := f.Base()
				// or InstallTestData?
				// ctx.InstallSymlink(installDir, symlinkName, f)
				// TODO(ron): do I wan Rel() instead of String?
				// ln creates  installDir/symlinkName -> f
				fmt.Printf("BASE TIF: %+v\n", f)

				// If in "android", I an get ctx.Config().soongOutDir
				// Need method to make relative to top in Android
				relLink := relativeToHere(installDir, f.String())
				ctx.InstallAbsoluteSymlink(installDir, symlinkName, relLink)
			}
		} else {
			fmt.Printf("NO BASE TIF: \n")
		}

		// out/soong/Android-aosp_shiba.mk:LOCAL_FULL_TEST_CONFIG := out/soong/.intermediates/frameworks/base/services/tests/servicestests/FrameworksServicesTests/android_common_FrameworksServicesOverride/c7b2e1d59c67dbe9379df594344f61da/test_config_fixer/AndroidTest.xml

	})
}

// prepend fileRelToTop with enough "../" for each path component of dirRelativeToTop
func relativeToHere(dirRelativeToTop android.Path, fileRelToTop string) string {
	fmt.Printf("%s %s %d\n", dirRelativeToTop, fileRelToTop, len(strings.Split(dirRelativeToTop.String(), "/")))
	return strings.Repeat("../", len(strings.Split(dirRelativeToTop.String(), "/"))) + fileRelToTop
}

func TestModuleConfigFactory() android.Module {
	module := &testModuleConfigModule{}

	module.AddProperties(&module.tfProperties)
	//Do we need to add host and device properties, do overload modules do it?
	//android.InitAndroidModule(module)
	android.InitAndroidArchModule(module, android.DeviceSupported, android.MultilibCommon)
	android.InitDefaultableModule(module)

	return module
}

// Implements android.AndroidMkEntriesProvider
func (r *testModuleConfigModule) AndroidMkEntries() []android.AndroidMkEntries {
	// TODO(ron); or r.base.Library.MkEntries() like ravenswood does.
	// maybe override LOCAL_MODULE or something.
	return []android.AndroidMkEntries{android.AndroidMkEntries{
		Class:      "APPS",
		OutputFile: android.OptionalPathForPath(r.output),
		// TODO(ron): right include
		// Include: "$(BUILD_SYSTEM)/test_module_config.mk",
		Include: "$(BUILD_SYSTEM)/soong_app_prebuilt.mk",
		// Include: "$(BUILD_SYSTEM)/phony_package.mk",
		// TODO(ron): Required: ?? field needed for base?
		// DistFiles:  android.MakeDefaultDistFiles(f.outputFile),
		ExtraEntries: []android.AndroidMkExtraEntriesFunc{
			func(ctx android.AndroidMkExtraEntriesContext, entries *android.AndroidMkEntries) {
				// TODO(ron): our name or base name here?
				// Failes uniqueness test.
				// entries.SetString("LOCAL_MODULE", r.base.Name() /* . app.installApkName*/)
				entries.SetString("LOCAL_MODULE", r.Name() /* . app.installApkName*/)
				// Get these from base somehow, needed to use soong_app_prebuilt.

				entries.SetString("LOCAL_SDK_VERSION", "current")   // prebuilt.sdkVersion.String())
				entries.SetString("LOCAL_CERTIFICATE", "PRESIGNED") // app.certificate.AndroidMkString())
				entries.SetString("LOCAL_MODULE_TAGS", "tests")
				entries.AddCompatibilityTestSuites("null-suite")
				// TODO(ron): see *Test version for more.

				// entries.AddStrings("LOCAL_COMPATIBILITY_SUPPORT_FILES", "DataFile")
				entries.SetPath("LOCAL_FULL_TEST_CONFIG", r.testConfig)
				// entries.SetString("LOCAL_MODULE_PATH", r.installDir.String())
				// entries.SetString("LOCAL_INSTALLED_MODULE_STEM", f.installFileName())
			},
		},

		// Ensure our "base" module is built and installed in testcases.
		/*
			ExtraFooters: []android.AndroidMkExtraFootersFunc{
				func(w io.Writer, name, prefix, moduleDir string) {
					fmt.Fprintln(w, r.Name()+"-target:", *r.Base+"-target")
				},
			},
		*/
	}}
}
