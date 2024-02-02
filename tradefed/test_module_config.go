package tradefed

import (
	"android/soong/android"
	"encoding/json"
	"fmt"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

// Ouput files we need from a base test that we derive from.
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
	base android.Module

	tradefedProperties

	// Our updated testConfig.
	testConfig android.OutputPath
	manifest   android.InstallPath
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
	ctx.AddDependency(ctx.Module(), testModuleConfigTag, *m.Base)
}

// Takes base's Tradefed Config xml file and generates a new one with the test properties
// appeneded from this module.
// Rewrite the name of the apk in "test-file-name" to be our module's name, rather than the orignal one.
func (m *testModuleConfigModule) fixTestConfig(ctx android.ModuleContext, baseTestConfig android.Path) android.OutputPath {
	// Test safe to do when no test_runner_options, but check for that earlier?
	fixedConfig := android.PathForModuleOut(ctx, "test_config_fixer", ctx.ModuleName()+".config")
	rule := android.NewRuleBuilder(pctx, ctx)
	command := rule.Command().BuiltTool("test_config_fixer").Input(baseTestConfig).Output(fixedConfig)
	options := m.composeOptions()
	if len(options) == 0 {
		ctx.ModuleErrorf("Test options must be given when using test_module_config. Set include/exclude filter or annotation.")
	}
	xmlTestModuleConfigSnippet, _ := json.Marshal(options)
	escaped := proptools.NinjaAndShellEscape(string(xmlTestModuleConfigSnippet))
	command.FlagWithArg("--test-file-name=", ctx.ModuleName()+".apk").
		FlagWithArg("--orig-test-file-name=", *m.tradefedProperties.Base+".apk").
		FlagWithArg("--test-runner-options=", escaped)
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
// 1) test_module_config.manifest
//   - Leave a trail of where we got files from in case other tools need it.
//
// 2) $Module.config
//   - comes from base's module.config (AndroidTest.xml), and then we add our test_options.
//     provider.TestConfig
//     [rules via soong_app_prebuilt]
//
// 3) $ARCH/$Module.apk
//   - comes from base
//     provider.OutputFile
//     [rules via soong_app_prebuilt]
//
// 4) [bases data]
//   - We copy all of bases data (like helper apks) to our install directory too.
//     Since we call AndroidMkEntries on base, it will write out LOCAL_COMPATIBILITY_SUPPORT_FILES
//     with this data and app_prebuilt.mk will generate the rules to copy it from base.
//     We have no direct rules here to add to ninja.
//
// If we change to symlinks, this all needs to change.
func (m *testModuleConfigModule) GenerateAndroidBuildActions(ctx android.ModuleContext) {

	base := ctx.GetDirectDepWithTag(*m.tradefedProperties.Base, testModuleConfigTag)
	if base == nil {
		ctx.PropertyErrorf("base", "module '%s' not found", *m.tradefedProperties.Base)
		return
	}
	m.base = base.(android.Module)

	provider, ok := android.OtherModuleProvider(ctx, base, BaseTestProviderKey)
	if !ok {
		ctx.ModuleErrorf("The base module '%s' does not provide test BaseTestProviderData.  Only 'android_test' modules are supported.", base.Name())
		return
	}
	m.provider = provider

	// 1) A manifest file listing the base.
	installDir := android.PathForModuleInstall(ctx, ctx.ModuleName())
	out := android.PathForModuleOut(ctx, "test_module_config.manifest")
	android.WriteFileRule(ctx, out, fmt.Sprintf("{%q: %q}", "base", base.Name()))
	ctx.InstallFile(installDir, out.Base(), out)

	// 2) Module.config / AndroidTest.xml
	// Note, there is still a "test-tag" element with base's module name, but
	// Tradefed team says its ignored anyway.
	m.testConfig = m.fixTestConfig(ctx, provider.TestConfig)
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
	// We rely on base writing LOCAL_COMPATIBILITY_SUPPORT_FILES for its data files
	entriesList := m.base.(android.AndroidMkEntriesProvider).AndroidMkEntries()
	entries := &entriesList[0]
	entries.OutputFile = android.OptionalPathForPath(m.provider.OutputFile)
	entries.ExtraEntries = append(entries.ExtraEntries, func(ctx android.AndroidMkExtraEntriesContext, entries *android.AndroidMkEntries) {
		entries.SetString("LOCAL_MODULE", m.Name()) //  out module name, not base's

		// Out update config file with extra options.
		entries.SetPath("LOCAL_FULL_TEST_CONFIG", m.testConfig)
		entries.SetString("LOCAL_MODULE_TAGS", "tests")
		// Required for atest to run additional tradefed testtypes
		entries.AddStrings("LOCAL_HOST_REQUIRED_MODULES", m.base.HostRequiredModuleNames()...)

		// Clear the JNI symbols because they belong to base not us. Either transform the names in the string
		// or clear the variable because we don't need it, we are copying bases libraries not generating
		// new ones.
		entries.SetString("LOCAL_SOONG_JNI_LIBS_SYMBOLS", "")

		// Don't append to base's test-suites, only use the ones we define, so clear it before
		// appending to it.
		entries.SetString("LOCAL_COMPATIBILITY_SUITE", "")
		if len(m.tradefedProperties.Test_suites) > 0 {
			entries.AddCompatibilityTestSuites(m.tradefedProperties.Test_suites...)
		} else {
			entries.AddCompatibilityTestSuites("null-suite")
		}
	})
	return entriesList
}
