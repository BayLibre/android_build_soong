package tradefed

import (
	"android/soong/android"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

// Some tests are shared between rules, where the binaries for the tests
// are exactly the same, but the options for the test differ.
// This interface allows differernt test types to publicize where the
// binary parts are (like data apks for android_tests)
// TODO(ron): make a provider.
type SharedTest interface {
	// Return paths to the base tests' installed/testcases dir.
	// I guess it doesn't have to be installed tree if we are resolving when zipping on the build.
	InstalledFiles() android.Paths
	// OutputFiles doesn't do what ne need either.
	OutputFile() android.Path

	// Or fixup the install path in IDEInfo to be correct?
	TestConfig() android.Path
	ExtraTestConfigs() android.Paths
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

	// Resolved test module that we will run.
	// We delegate many methods here, but can't set until GenerateAndroidBuildActions.
	base android.Module
	tradefedProperties

	// Do we need to declare our outs?
	// Using InstallPath gave me missing package.apk
	output android.OutputPath

	// Our updated testConfig.
	testConfig android.OutputPath

	testData android.InstallPaths // needed path reflection based output.
}

// Properties to list in Android.bp for this module.
type tradefedProperties struct {
	// Module name of the base test that we will run.
	Base *string `android:"path,arch_variant"`
	// Options to add our copy of AndroidTest.config
	Options []Option

	Test_suites []string `android:"arch_variant"`
}

type dependencyTag struct {
	blueprint.BaseDependencyTag
	name string
}

var testModuleConfigTag = dependencyTag{name: "TestModuleConfigBase"}

// Do I need an InstallForceOS too?
func (m *testModuleConfigModule) InstallInTestcases() bool {
	// TODO(ron): test for both java_test_host and android_test?
	return !m.Host()
}

func (m *testModuleConfigModule) DepsMutator(ctx android.BottomUpMutatorContext) {
	// TODO(rbraunstein); ask about use variant dependency?
	ctx.AddDependency(ctx.Module(), testModuleConfigTag, *m.Base)
}

func (a *testModuleConfigModule) fixTestConfig(ctx android.ModuleContext, testConfig android.Path) android.OutputPath {
	// Test safe to do when no test_runner_options, but check for that earlier?
	fixedConfig := android.PathForModuleOut(ctx, "test_config_fixer", ctx.ModuleName()+".config")
	rule := android.NewRuleBuilder(pctx, ctx)
	command := rule.Command().BuiltTool("test_config_fixer").Input(testConfig).Output(fixedConfig)
	xmlTestModuleConfignerSnippet, _ := json.Marshal(a.tradefedProperties.Options)
	escaped := proptools.NinjaAndShellEscape(string(xmlTestModuleConfignerSnippet))
	command.Flag("--test-runner-options ").Text(escaped)
	rule.Build("fix_test_config", "fix test config")
	return fixedConfig.OutputPath
}

// derivedClass, BaseClassProvider
//
//	test_config, and other stuff needed in androidmk
//
// Files to write and where they come from:
// 1) Module.config
//   - comes from base's module.config, and then we add our test_options.
//
// 2) $ARCH/Module.apk
//   - I think this comes from our "out" property and soong_app_prebuilt.mk wants to build it.
//     The actual apk is garbage, maybe it should link back to base's apk?
//
// 3) Base.apk
//   - We put a copy of base.apk in our install tree. I'm not sure if we need to, since
//     we have to install base too.
//
// 4) [bases data]
//   - We copy all of bases data (like helper apks) to our install directory too.
//     The provider gives the "intermediate" paths, not the test-data paths, but
//     we don't really care.
func (m *testModuleConfigModule) GenerateAndroidBuildActions(ctx android.ModuleContext) {

	const useLinks = false

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
		out := android.PathForModuleOut(ctx, "test_module_config.manifest")
		ctx.InstallFile(installDir, out.Base(), out)
		android.WriteFileRule(ctx, out, base.Name())
		m.output = out.OutputPath

		baseApk := base.(SharedTest).OutputFile()
		if useLinks {
			symlinkName := baseApk.Base()
			relLink := relativeToHere(installDir, baseApk.String())
			/*out := */ ctx.InstallAbsoluteSymlink(installDir, symlinkName, relLink)
		} else {
			// out := android.PathForModuleOut(ctx
			// ctx.InstallFile(installDir, out.Base(), out)
			// TODO(ron): add back if base needed.
		}

		// 2) TestConfig
		// Note, there is still a "test-tag" element with base's module name, but
		// TF team says its ignored anyway.

		if provider, ok := base.(SharedTest); ok {
			fmt.Printf("TestConfig found: %s  %v\n", ctx.ModuleName(), provider.TestConfig())
			// TODO(ron): we let the .mk file write the AndroidTestConfig after we save its location here
			// and emit in AndroidMkEntries.
			m.base = base // TODO(ron); write-only fix in mkEntries?
			m.testConfig = m.fixTestConfig(ctx, provider.TestConfig())

			// 3) deps
			// TODO(ron): In DepsMutator, ensure it can be type asserted
			if useLinks {
				for _, f := range provider.InstalledFiles() {
					symlinkName := f.Base()
					// TODO(ron): do I wan Rel() instead of String?
					// ln creates  installDir/symlinkName -> f
					fmt.Printf("BASE TIF: %+v\n", f)

					// If in "android", I can get ctx.Config().soongOutDir
					// Need method to make relative to top in Android
					relLink := relativeToHere(installDir, f.String())
					ctx.InstallAbsoluteSymlink(installDir, symlinkName, relLink)
				}
			} else {
				// doesn't work?
				// ctx.InstallFile(installDir, f.Base(), f)
				dataFiles := make([]android.DataPath, len(provider.InstalledFiles()))
				for i, f := range provider.InstalledFiles() {
					dataFiles[i] = android.DataPath{SrcPath: f}
				}
				// TODO(ron): this doesn't do anything either?
				m.testData = ctx.InstallTestData(installDir, dataFiles)

			}

		} else {
			// TODO(ron): Error here, we have a dependency we didn't expect.
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

	module.AddProperties(&module.tradefedProperties)
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
				if len(r.tradefedProperties.Test_suites) > 0 {
					entries.AddCompatibilityTestSuites(r.tradefedProperties.Test_suites...)
				} else {
					// TODO(ron) or inherit from base?
					entries.AddCompatibilityTestSuites("null-suite")
				}
				// TODO(ron): see *Test version for more.

				// entries.AddStrings("LOCAL_COMPATIBILITY_SUPPORT_FILES", "DataFile")
				entries.SetPath("LOCAL_FULL_TEST_CONFIG", r.testConfig)
				// entries.SetString("LOCAL_MODULE_PATH", r.installDir.String())
				// entries.SetString("LOCAL_INSTALLED_MODULE_STEM", f.installFileName())
			},
		},

		// Ensure our "base" module is built and installed in testcases.
		ExtraFooters: []android.AndroidMkExtraFootersFunc{
			func(w io.Writer, name, prefix, moduleDir string) {
				fmt.Fprintln(w, r.Name()+"-target:", *r.Base+"-target")
			},
		},
	}}
}
