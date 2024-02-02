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

// Provider published by tradefed.
type BaseTestProviderData struct {
	// data files and apps for android_test
	InstalledFiles android.Paths
	// apk for android_test
	OutputFile android.Path
	// Either handwritten or generated TF xml.
	TestConfig android.Path
}

var BaseTestProviderKey = blueprint.NewProvider[BaseTestProviderData]()

// ErrTestModuleDataNotFound is the error message for missing test module provider data.
const ErrBaseModuleNotFound = "The module '%s' does not provide test BaseTestProviderData.  Only some Tests are can be used in `test_module_config`\n"

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
	Options             []Option
	Include_annotations []string
	Exclude_annotations []string
	Include_filters     []string
	Exclude_filters     []string

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

// Takes base's Tradefed Config xml file and generates a new one with properties
// set on this module.
func (a *testModuleConfigModule) fixTestConfig(ctx android.ModuleContext, baseTestConfig android.Path) android.OutputPath {
	// Test safe to do when no test_runner_options, but check for that earlier?
	fixedConfig := android.PathForModuleOut(ctx, "test_config_fixer", ctx.ModuleName()+".config")
	rule := android.NewRuleBuilder(pctx, ctx)
	command := rule.Command().BuiltTool("test_config_fixer").Input(baseTestConfig).Output(fixedConfig)
	options := a.composeOptions()
	xmlTestModuleConfigSnippet, _ := json.Marshal(options)
	escaped := proptools.NinjaAndShellEscape(string(xmlTestModuleConfigSnippet))
	command.Flag("--test-runner-options ").Text(escaped)
	rule.Build("fix_test_config", "fix test config")
	return fixedConfig.OutputPath
}

// Convert --exclude_filters: ["filter1", "filter2"] ->
// [ Option{Name: "exclude-filters", Value: "filter1"}, Option{Name: "exclude-filters", Value: "filter2"},
// ... + include + annotations ]
func (a *testModuleConfigModule) composeOptions() []Option {
	options := []Option{}
	for _, e := range a.Exclude_filters {
		options = append(options, Option{Name: "exclude-filter", Value: e})
	}
	for _, i := range a.Include_filters {
		options = append(options, Option{Name: "include-filter", Value: i})
	}
	for _, e := range a.Exclude_annotations {
		options = append(options, Option{Name: "exclude-annotation", Value: e})
	}
	for _, i := range a.Include_annotations {
		options = append(options, Option{Name: "include-annotation", Value: i})
	}
	return options
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

	var installDir android.InstallPath
	if ctx.InstallInTestcases() {
		installDir = android.PathForModuleInstall(ctx, ctx.ModuleName())
	} else {
		// TODO(ron): verify on java_host_test or something.
		installDir = android.PathForModuleInstall(ctx, "framework")
	}
	fmt.Printf("TestConfig installDir: %s  %v\n", ctx.ModuleName(), installDir)

	// We already base name, we don't need to visit it.
	// perhaps walk the base properties looking for an installPath?

	base := ctx.GetDirectDepWithTag(*m.tradefedProperties.Base, testModuleConfigTag)
	if base == nil {
		ctx.ModuleErrorf("Base property not setup correctly on %s", ctx.ModuleName())
		return
	}

	provider, ok := android.OtherModuleProvider(ctx, base, BaseTestProviderKey)
	if !ok {
		// TODO(ron): Error here, we have a dependency we didn't expect.
		fmt.Printf("NO BASE TIF: \n")
		ctx.ModuleErrorf(ErrBaseModuleNotFound, base.Name())
		return
	}

	// TODO(ron); write-only fix in mkEntries?
	// m.base = base
	out := android.PathForModuleOut(ctx, "test_module_config.manifest")
	android.WriteFileRule(ctx, out, base.Name())
	m.output = out.OutputPath
	// fmt.Printf("MANIFEST: out (%s)  installed (%v)\n", out.OutputPath, manifest_installed)

	// 2) TestConfig
	// Note, there is still a "test-tag" element with base's module name, but
	// TF team says its ignored anyway.

	fmt.Printf("TestConfig found: %s  %v\n", ctx.ModuleName(), provider.TestConfig)

	// TestConfig if written by app_prebuilt, not InstallFile.
	m.testConfig = m.fixTestConfig(ctx, provider.TestConfig)

	base_apk := provider.OutputFile
	base_installed := ctx.InstallFile(installDir, base_apk.Rel(), base_apk)
	deps := []android.InstallPath{base_installed}

	// 3) deps
	// TODO(ron): In DepsMutator, ensure it can be type asserted
	// doesn't work?
	for _, f := range provider.InstalledFiles {
		fmt.Printf("BASE TIF: %+v, (%s)\n", f, f.Rel())
		deps = append(deps, ctx.InstallFile(installDir, f.Rel(), f))
	}

	ctx.InstallFile(installDir, out.Base(), out, deps...)

	// out/soong/Android-aosp_shiba.mk:LOCAL_FULL_TEST_CONFIG := out/soong/.intermediates/frameworks/base/services/tests/servicestests/FrameworksServicesTests/android_common_FrameworksServicesOverride/c7b2e1d59c67dbe9379df594344f61da/test_config_fixer/AndroidTest.xml

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
