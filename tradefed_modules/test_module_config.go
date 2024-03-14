package tradefed_modules

import (
	"android/soong/android"
	"android/soong/tradefed"
	"encoding/json"
	"fmt"
	"io"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

func init() {
	RegisterTestModuleConfigBuildComponents(android.InitRegistrationContext)
}

// Register the license_kind module type.
func RegisterTestModuleConfigBuildComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("test_module_config", TestModuleConfigFactory)
	ctx.RegisterModuleType("test_module_config_host", TestModuleConfigHostFactory)
}

type testModuleConfigModule struct {
	android.ModuleBase
	android.DefaultableModuleBase
	base android.Module

	tradefedProperties

	// Our updated testConfig.
	testConfig android.OutputPath
	manifest   android.InstallPath
	provider   tradefed.BaseTestProviderData
}

// Host is mostly the same as non-host, just some diffs for AddDependency and
// AndroidMkEntries, but the properties are the same.
type testModuleConfigHostModule struct {
	testModuleConfigModule
}

// Properties to list in Android.bp for this module.
type tradefedProperties struct {
	// Module name of the base test that we will run.
	Base *string `android:"path,arch_variant"`

	// Tradefed Options to add to tradefed xml when not one of the include or exclude filter or property.
	// Sample: [{name: "TestRunnerOptionName", value: "OptionValue" }]
	Options []tradefed.Option

	// List of tradefed include annotations to add to tradefed xml, like "android.platform.test.annotations.Presubmit".
	// Tests will be restricted to those matching an include_annotation or include_filter.
	Include_annotations []string

	// List of tradefed include annotations to add to tradefed xml, like "android.support.test.filters.FlakyTest".
	// Tests matching an exclude annotation or filter will be skipped.
	Exclude_annotations []string

	// List of tradefed include filters to add to tradefed xml, like "fully.qualified.class#method".
	// Tests will be restricted to those matching an include_annotation or include_filter.
	Include_filters []string

	// List of tradefed exclude filters to add to tradefed xml, like "fully.qualified.class#method".
	// Tests matching an exclude annotation or filter will be skipped.
	Exclude_filters []string

	// List of compatibility suites (for example "cts", "vts") that the module should be
	// installed into.
	Test_suites []string
}

type dependencyTag struct {
	blueprint.BaseDependencyTag
	name string
}

var (
	testModuleConfigTag = dependencyTag{name: "TestModuleConfigBase"}
	pctx                = android.NewPackageContext("android/soong/tradefed_modules")
)

func (m *testModuleConfigModule) InstallInTestcases() bool {
	return true
}

func (m *testModuleConfigModule) DepsMutator(ctx android.BottomUpMutatorContext) {
	ctx.AddDependency(ctx.Module(), testModuleConfigTag, *m.Base)
}

// Takes base's Tradefed Config xml file and generates a new one with the test properties
// appeneded from this module.
// Rewrite the name of the apk in "test-file-name" to be our module's name, rather than the original one.
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
func (m *testModuleConfigModule) composeOptions() []tradefed.Option {
	options := m.Options
	for _, e := range m.Exclude_filters {
		options = append(options, tradefed.Option{Name: "exclude-filter", Value: e})
	}
	for _, i := range m.Include_filters {
		options = append(options, tradefed.Option{Name: "include-filter", Value: i})
	}
	for _, e := range m.Exclude_annotations {
		options = append(options, tradefed.Option{Name: "exclude-annotation", Value: e})
	}
	for _, i := range m.Include_annotations {
		options = append(options, tradefed.Option{Name: "include-annotation", Value: i})
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

	ctx.VisitDirectDepsWithTag(testModuleConfigTag, func(dep android.Module) {
		if provider, ok := android.OtherModuleProvider(ctx, dep, tradefed.BaseTestProviderKey); ok {
			m.base = dep
			m.provider = provider
		} else {
			ctx.ModuleErrorf("The base module '%s' does not provide test BaseTestProviderData.  Only 'android_test' modules are supported.", dep.Name())
			return
		}
	})

	// 1) A manifest file listing the base.
	// TODO(ron): add testcases back in? remove false flag from IS_TESTCASE or something?
	installDir := android.PathForModuleInstall(ctx, ctx.ModuleName())
	out := android.PathForModuleOut(ctx, "test_module_config.manifest")
	android.WriteFileRule(ctx, out, fmt.Sprintf("{%q: %q}", "base", *m.tradefedProperties.Base))
	ctx.InstallFile(installDir, out.Base(), out)

	// 2) Module.config / AndroidTest.xml
	// Note, there is still a "test-tag" element with base's module name, but
	// Tradefed team says its ignored anyway.
	m.testConfig = m.fixTestConfig(ctx, m.provider.TestConfig)

	// 3) Write ARCH/Module.apk in testcases.
	// Handled by soong_app_prebuilt and OutputFile in entries.
	// Nothing to do here.

	// 4) Copy base's data files.
	// Handled by soong_app_prebuilt and LOCAL_COMPATIBILITY_SUPPORT_FILES.
	// Nothing to do here.
}

func TestModuleConfigFactory() android.Module {
	module := &testModuleConfigModule{}

	module.AddProperties(&module.tradefedProperties)
	android.InitAndroidArchModule(module, android.DeviceSupported, android.MultilibCommon)
	android.InitDefaultableModule(module)

	return module
}

func TestModuleConfigHostFactory() android.Module {
	module := &testModuleConfigHostModule{}

	module.AddProperties(&module.tradefedProperties)
	android.InitAndroidMultiTargetsArchModule(module, android.HostSupported, android.MultilibCommon)
	android.InitDefaultableModule(module)
	return module
}

// Implements android.AndroidMkEntriesProvider
var _ android.AndroidMkEntriesProvider = (*testModuleConfigModule)(nil)

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
		entries.AddStrings("LOCAL_HOST_REQUIRED_MODULES", m.provider.HostRequiredModuleNames...)

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

func (m *testModuleConfigHostModule) InstallInTestcases() bool {
	return false
}

func (m *testModuleConfigHostModule) DepsMutator(ctx android.BottomUpMutatorContext) {
	ctx.AddFarVariationDependencies(ctx.Config().BuildOSCommonTarget.Variations(), testModuleConfigTag, *m.Base)
}

func (m *testModuleConfigHostModule) GenerateAndroidBuildActions(ctx android.ModuleContext) {

	visited := false
	ctx.VisitDirectDepsWithTag(testModuleConfigTag, func(dep android.Module) {
		if provider, ok := android.OtherModuleProvider(ctx, dep, tradefed.BaseTestProviderKey); ok {
			m.base = dep
			m.provider = provider
			visited = true
		} else {
			ctx.ModuleErrorf("The base module '%s' does not provide test BaseTestProviderData.  Only 'java_test_host' modules are supported.", dep.Name())
			return
		}
	})

	if !visited {
		ctx.ModuleErrorf("The base module '%s' does not provide test BaseTestProviderData.  Only 'java_test_host' modules are supported.", *m.Base)
		return

	}

	// a) out/host/linux-x86/testcases/derived-module/derived-module.config
	// b) out/host/linux-x86/testcases/derived-module/base.jar
	// c) out/host/linux-x86/framework/base.jar  [** multi ** ]
	//      ctx.InstallFile works for this.
	//

	// 1) A manifest file listing the base, write text to a tiny file.
	installDir := android.PathForModuleInstall(ctx, ctx.ModuleName())

	manifest := android.PathForModuleOut(ctx, "test_module_config.manifest")
	android.WriteFileRule(ctx, manifest, fmt.Sprintf("{%q: %q}", "base", *m.tradefedProperties.Base))
	// TODO(ron): bring back?
	ctx.InstallFile(installDir, manifest.Base(), manifest)

	// 2) Module.config / AndroidTest.xml
	// Note, there is still a "test-tag" element with base's module name, but
	// Tradefed team says its ignored anyway.
	m.testConfig = m.fixTestConfig(ctx, m.provider.TestConfig)

	// build/soong/android/androidmk.go has this comment:
	//    Assume the primary install file is last
	// so we need to Install our file last.

	fmt.Printf("BASE: outfile: %s\n", m.provider.OutputFile)
	// But the installDir should be:
	//   		installDir = android.PathForModuleInstall(ctx, "framework")
	// We don't want two rules installing the jar, rename again?
	// TODO(ron): not needed.
	// ctx.InstallFile(installDir, m.provider.OutputFile.Base()+"-derived_copy", m.provider.OutputFile)
	// ctx.InstallAbsoluteSymlink(installDir, m.provider.OutputFile.Base(), m.provider.OutputFile.String())

	// 3) Write ARCH/Module.apk in testcases.
	// Handled by soong_app_prebuilt and OutputFile in entries.
	// Nothing to do here.

	// 4) Copy base's data files.
	// Handled by soong_app_prebuilt and LOCAL_COMPATIBILITY_SUPPORT_FILES.
	// Nothing to do here.
}

var _ android.AndroidMkEntriesProvider = (*testModuleConfigHostModule)(nil)

// Things to fix:
//  * SOONG_INSTALLED_MODULE is the config, should be out/host/linux-x86/framework/CtsAppSecurityHostTestCases.jar
//  * INSTALL_PAIRS?
/*
 % grep -B2 -A22 'LOCAL_MODULE := CtsAppSecurityHostTestCases' ~/aosp-main-with-phones/out/soong/Android-aosp_shiba.mk| head -24 > local.mk.cash
 % grep -B2 -A22 'LOCAL_MODULE := CtsAppSecurityHostTestCases_' ~/aosp-main-with-phones/out/soong/Android-aosp_shiba.mk| head -24 > local.mk.cash.pre
*/

func (m *testModuleConfigHostModule) AndroidMkEntries() []android.AndroidMkEntries {
	// We rely on base writing LOCAL_COMPATIBILITY_SUPPORT_FILES for its data files
	entriesList := m.base.(android.AndroidMkEntriesProvider).AndroidMkEntries()
	entries := &entriesList[0]
	// entries.OutputFile = android.OptionalPathForPath(m.provider.OutputFile)
	entries.ExtraEntries = append(entries.ExtraEntries, func(ctx android.AndroidMkExtraEntriesContext, entries *android.AndroidMkEntries) {
		entries.SetString("LOCAL_MODULE", m.Name()) //  out module name, not base's

		// Out update config file with extra options.
		entries.SetPath("LOCAL_FULL_TEST_CONFIG", m.testConfig)
		entries.SetString("LOCAL_MODULE_TAGS", "tests")
		// Required for atest to run additional tradefed testtypes
		entries.AddStrings("LOCAL_REQUIRED_MODULES", m.provider.HostRequiredModuleNames...)

		// Don't append to base's test-suites, only use the ones we define, so clear it before
		// appending to it.
		entries.SetString("LOCAL_COMPATIBILITY_SUITE", "")
		if len(m.tradefedProperties.Test_suites) > 0 {
			entries.AddCompatibilityTestSuites(m.tradefedProperties.Test_suites...)
		} else {
			entries.AddCompatibilityTestSuites("null-suite")
		}
	})

	// The "base" jar is written to out/host/ARCH/framework dir.
	// All tests seems to share that directory in addition to writing to testcases.
	// If our module (derived), depends on base, then our config file depend on base.jar
	// We don't really want to make an extra copy to derived.jar and we can't use an install
	// rule to ensure base.jar gets written to the framework dir, because then there would
	// be two rules creating that file.  Instead we just add a dependency on the target.
	entries.ExtraFooters = []android.AndroidMkExtraFootersFunc{
		func(w io.Writer, name, prefix, moduleDir string) {
			baseDep := *m.tradefedProperties.Base
			fmt.Fprintln(w, m.Name()+":", baseDep)
		},
	}

	return entriesList
}

func (m *testModuleConfigHostModule) MakeVars(ctx android.MakeVarsContext) {
	neededBaseJar := m.provider.OutputFile
	fmt.Printf("GOAL: %s - %s\n", m.Name(), neededBaseJar)
	// ctx.DistForGoal(m.Name()+"-host", neededBaseJar+"-host")
	ctx.DistForGoal(m.Name(), neededBaseJar)
	// "all_teams", this.outputPath)
}

// out/host/linux-x86/framework
// vs out/host/linux-x86/testcases/framework

/*
   atest --collect-tests-only 'android.appsecurity.cts.EphemeralTest'
   atest -v --collect-tests-only 'android.appsecurity.cts.EphemeralTest#testEphemeralStartExposed01'
*/
