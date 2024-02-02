package tradefed

import (
	"android/soong/android"
	"encoding/json"
	"fmt"
	"io"

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

	tradefedProperties

	// Do we need to declare our outs?
	// Using InstallPath gave me missing package.apk
	// xx output android.Path

	// Our updated testConfig.
	testConfig android.OutputPath
	provider   BaseTestProviderData
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

	Test_suites []string
}

type dependencyTag struct {
	blueprint.BaseDependencyTag
	name string
}

var testModuleConfigTag = dependencyTag{name: "TestModuleConfigBase"}

func (m *testModuleConfigModule) InstallInTestcases() bool {
	return true
}

func (m *testModuleConfigModule) DepsMutator(ctx android.BottomUpMutatorContext) {
	dep := ctx.AddDependency(ctx.Module(), testModuleConfigTag, *m.Base)
	if len(dep) != 1 || dep[0] == nil {
		ctx.PropertyErrorf("base", "module '%s' Not found", *m.Base)
	}
}

// Takes base's Tradefed Config xml file and generates a new one with the test properties
// appeneded from this module.
func (m *testModuleConfigModule) fixTestConfig(ctx android.ModuleContext, baseTestConfig android.Path) android.OutputPath {
	// Test safe to do when no test_runner_options, but check for that earlier?
	fixedConfig := android.PathForModuleOut(ctx, "test_config_fixer", ctx.ModuleName()+".config")
	rule := android.NewRuleBuilder(pctx, ctx)
	command := rule.Command().BuiltTool("test_config_fixer").Input(baseTestConfig).Output(fixedConfig)
	options := m.composeOptions()
	xmlTestModuleConfigSnippet, _ := json.Marshal(options)
	escaped := proptools.NinjaAndShellEscape(string(xmlTestModuleConfigSnippet))
	command.Flag("--test-runner-options ").Text(escaped)
	rule.Build("fix_test_config", "fix test config")
	return fixedConfig.OutputPath
}

// Convert --exclude_filters: ["filter1", "filter2"] ->
// [ Option{Name: "exclude-filters", Value: "filter1"}, Option{Name: "exclude-filters", Value: "filter2"},
// ... + include + annotations ]
func (m *testModuleConfigModule) composeOptions() []Option {
	options := m.Options
	for _, e := range m.Exclude_filters {
		options = append(options, Option{Name: "exclude-filter", Value: e})
	}
	for _, i := range m.Include_filters {
		options = append(options, Option{Name: "include-filter", Value: i})
	}
	for _, e := range m.Exclude_annotations {
		options = append(options, Option{Name: "exclude-annotation", Value: e})
	}
	for _, i := range m.Include_annotations {
		options = append(options, Option{Name: "include-annotation", Value: i})
	}
	return options
}

// Files to write and where they come from:
// 1) $Module.config
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
//     The provider gives the "intermediate" paths, not the installed test-data paths, but
//     either is fine.
func (m *testModuleConfigModule) GenerateAndroidBuildActions(ctx android.ModuleContext) {

	installDir := android.PathForModuleInstall(ctx, ctx.ModuleName())

	// We already base name, we don't need to visit it.
	// perhaps walk the base properties looking for an installPath?

	base := ctx.GetDirectDepWithTag(*m.tradefedProperties.Base, testModuleConfigTag)
	if base == nil {
		ctx.PropertyErrorf("base", "module '%s' not found", *m.tradefedProperties.Base)
		return
	}

	provider, ok := android.OtherModuleProvider(ctx, base, BaseTestProviderKey)
	if !ok {
		ctx.ModuleErrorf("The module '%s' does not provide test BaseTestProviderData.  Only some Tests are can be used in `test_module_config`\n", base.Name())
		return
	}
	m.provider = provider

	// 0) A manifest file listing the base.
	out := android.PathForModuleOut(ctx, "test_module_config.manifest")
	android.WriteFileRule(ctx, out, fmt.Sprintf("{%q: %q}", "base", base.Name()))
	// xx m.output = out.OutputPath

	// 2) TestConfig
	// Note, there is still a "test-tag" element with base's module name, but
	// TF team says its ignored anyway.

	// TestConfig if written by app_prebuilt, not InstallFile.
	m.testConfig = m.fixTestConfig(ctx, provider.TestConfig)

	// xx base_apk := provider.OutputFile
	/*
		base_installed := ctx.InstallFile(installDir, base_apk.Rel(), base_apk)
		deps := []android.InstallPath{base_installed}
		m.output = base_installed
	*/
	deps := []android.InstallPath{}
	// xx m.output = base_apk
	// 3) deps
	// TODO(ron): In DepsMutator, ensure it can be type asserted
	// doesn't work?
	for _, f := range provider.InstalledFiles {
		deps = append(deps, ctx.InstallFile(installDir, f.Rel(), f))
	}

	ctx.InstallFile(installDir, out.Base(), out, deps...)
}

func TestModuleConfigFactory() android.Module {
	module := &testModuleConfigModule{}

	module.AddProperties(&module.tradefedProperties)
	android.InitAndroidArchModule(module, android.DeviceSupported, android.MultilibCommon)
	android.InitDefaultableModule(module)

	return module
}

// Implements android.AndroidMkEntriesProvider
func (m *testModuleConfigModule) AndroidMkEntries() []android.AndroidMkEntries {
	/*
		   // Perhaps try with OverrideName if this works, allong with LOCAL_MODULE.
		   // failed for other rasons?
			entriesList := m.base.(android.AndroidMkEntriesProvider).AndroidMkEntries()
			entries := &entriesList[0]
			entries.ExtraEntries = append(entries.ExtraEntries, func(ctx android.AndroidMkExtraEntriesContext, entries *android.AndroidMkEntries) {
				entries.SetString("LOCAL_MODULE", m.Name()) //  out module name, not base's
				entries.SetPath("LOCAL_FULL_TEST_CONFIG", m.testConfig)
				entries.SetString("LOCAL_MODULE_TAGS", "tests")
				if m.tradefedProperties.Is_host {
					entries.SetBool("LOCAL_IS_HOST_MODULE", true)
				}
				// entries.SetPath("LOCAL_SOONG_DEX_JAR", m.providem.OutputFile)
				if len(m.tradefedProperties.Test_suites) > 0 {
					entries.AddCompatibilityTestSuites(m.tradefedProperties.Test_suites...)
				} else {
					// TODO(ron) or inherit from base?
					entries.AddCompatibilityTestSuites("null-suite")
				}
			})
			return entriesList
	*/

	// maybe override LOCAL_MODULE or something.
	return []android.AndroidMkEntries{android.AndroidMkEntries{
		Class: "APPS",
		// OutputFile: android.OptionalPathForPath(m.output),
		OutputFile: android.OptionalPathForPath(m.provider.OutputFile),
		// TODO(ron): right include
		// Include: "$(BUILD_SYSTEM)/test_module_config.mk",
		Include: "$(BUILD_SYSTEM)/soong_app_prebuilt.mk",
		// Include: "$(BUILD_SYSTEM)/phony_package.mk",
		// TODO(ron): Required: ?? field needed for base?
		// DistFiles:  android.MakeDefaultDistFiles(f.outputFile),
		ExtraEntries: []android.AndroidMkExtraEntriesFunc{
			func(ctx android.AndroidMkExtraEntriesContext, entries *android.AndroidMkEntries) {
				// Setting LocalModuleStem will use that base name for the apk to install in our directory.
				// It finds the apk via OutputFile above.
				entries.SetString("LOCAL_MODULE_STEM", *m.tradefedProperties.Base)
				entries.SetString("LOCAL_MODULE", m.Name())
				// Get these from base somehow, needed to use soong_app_prebuilt.
				entries.SetString("LOCAL_SDK_VERSION", "current")   // prebuilt.sdkVersion.String())
				entries.SetString("LOCAL_CERTIFICATE", "PRESIGNED") // app.certificate.AndroidMkString())
				entries.SetString("LOCAL_MODULE_TAGS", "tests")
				if len(m.tradefedProperties.Test_suites) > 0 {
					entries.AddCompatibilityTestSuites(m.tradefedProperties.Test_suites...)
				} else {
					entries.AddCompatibilityTestSuites("null-suite")
				}
				// TODO(ron): see *Test version for more.

				// entries.AddStrings("LOCAL_COMPATIBILITY_SUPPORT_FILES", "DataFile")
				entries.SetPath("LOCAL_FULL_TEST_CONFIG", m.testConfig)
				// entries.SetString("LOCAL_MODULE_PATH", m.installDim.String())
				// entries.SetString("LOCAL_INSTALLED_MODULE_STEM", f.installFileName())
			},
		},

		// Ensure our "base" module is built and installed in testcases.
		ExtraFooters: []android.AndroidMkExtraFootersFunc{
			func(w io.Writer, name, prefix, moduleDir string) {
				fmt.Fprintln(w, m.Name()+"-target:", *m.Base+"-target")
			},
		},
	}}

}
