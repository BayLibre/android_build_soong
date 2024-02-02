package tradefed

import (
	"android/soong/android"
	"encoding/json"
	"fmt"
	"io"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

// Or fixup the install path in IDEInfo to be correct?
type TestConfigProvider interface {
	TestConfig() android.Path
	ExtraTestConfigs() android.Paths
}

func init() {
	RegisterTestRunBuildComponents(android.InitRegistrationContext)
}

// Register the license_kind module type.
func RegisterTestRunBuildComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("test_run", TestRunFactory)
}

type testRunModule struct {
	android.ModuleBase
	android.DefaultableModuleBase

	tfProperties
	// Do we need to declare our outs?
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

var testRunTag = dependencyTag{name: "TestRunBase"}

// Do I need an InstallForceOS too?
func (m *testRunModule) InstallInTestcases() bool {
	// TODO(ron): should depend on base, right?
	// This should work for host tests too.
	return true
}

func (m *testRunModule) DepsMutator(ctx android.BottomUpMutatorContext) {
	// Todo(ron); make tag, use variant dependency?
	ctx.AddDependency(ctx.Module(), testRunTag, *m.Base)
}

func (a *testRunModule) FixTestConfig(ctx android.ModuleContext, testConfig android.Path) android.OutputPath {
	// Test safe to do when no test_runner_options, but check for that earlier?
	fixedConfig := android.PathForModuleOut(ctx, "test_config_fixer", ctx.ModuleName()+".config")
	rule := android.NewRuleBuilder(pctx, ctx)
	command := rule.Command().BuiltTool("test_config_fixer").Input(testConfig).Output(fixedConfig)
	xmlTestRunnerSnippet, _ := json.Marshal(a.tfProperties.Options)
	escaped := proptools.NinjaAndShellEscape(string(xmlTestRunnerSnippet))
	command.Flag("--test-runner-options ").Text(escaped)
	rule.Build("fix_test_config", "fix test config")
	return fixedConfig.OutputPath
}

// TODO(ron): more interfaces
//
//	derivedClass, BaseClassProvider
//	    test_config, and other stuff needed in androidmk
func (m *testRunModule) GenerateAndroidBuildActions(ctx android.ModuleContext) {

	var installDir android.InstallPath
	if ctx.InstallInTestcases() {
		installDir = android.PathForModuleInstall(ctx, ctx.ModuleName())
	} else {
		// TODO(ron): verify on java_host_test or something.
		installDir = android.PathForModuleInstall(ctx, "framework")
	}

	// We already base name, we don't need to visit it.
	// perhaps walk the base properties looking for an installPath?
	ctx.VisitDirectDepsWithTag(testRunTag, func(base android.Module) {
		out := android.PathForModuleOut(ctx, "test_module_config.manifest")
		android.WriteFileRule(ctx, out, base.Name())
		// TODO(ron): FIXME: NOTE: using install file is forcing this be installed as apk too!
		ctx.InstallFile(installDir, out.Base(), out)
		m.output = out.OutputPath

		if provider, ok := base.(TestConfigProvider); ok {
			fmt.Printf("TestConfig found: %s  %v\n", ctx.ModuleName(), provider.TestConfig())
			// TODO(ron): we let the .mk file write the AndroidTestConfig after we save its location here
			// and emit in AndroidMkEntries.
			m.testConfig = m.FixTestConfig(ctx, provider.TestConfig())
			// ctx.InstallFile(installDir, m.testConfig.Base(), m.testConfig)
		}

		// out/soong/Android-aosp_shiba.mk:LOCAL_FULL_TEST_CONFIG := out/soong/.intermediates/frameworks/base/services/tests/servicestests/FrameworksServicesTests/android_common_FrameworksServicesOverride/c7b2e1d59c67dbe9379df594344f61da/test_config_fixer/AndroidTest.xml

	})
}

func TestRunFactory() android.Module {
	module := &testRunModule{}

	module.AddProperties(&module.tfProperties)
	//Do we need to add host and device properties, do overload modules do it?
	//android.InitAndroidModule(module)
	android.InitAndroidArchModule(module, android.DeviceSupported, android.MultilibCommon)
	android.InitDefaultableModule(module)

	return module
}

// Implements android.AndroidMkEntriesProvider
func (r *testRunModule) AndroidMkEntries() []android.AndroidMkEntries {
	return []android.AndroidMkEntries{android.AndroidMkEntries{
		Class:      "APPS",
		OutputFile: android.OptionalPathForPath(r.output),
		// TODO(ron): right include
		Include: "$(BUILD_SYSTEM)/soong_app_prebuilt.mk",
		// Include: "$(BUILD_SYSTEM)/phony_package.mk",
		// TODO(ron): Required: ?? field needed for base?
		// DistFiles:  android.MakeDefaultDistFiles(f.outputFile),
		ExtraEntries: []android.AndroidMkExtraEntriesFunc{
			func(ctx android.AndroidMkExtraEntriesContext, entries *android.AndroidMkEntries) {
				// TODO(ron): our name or base name here?
				// entries.SetString("LOCAL_MODULE", app.installApkName)
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
		ExtraFooters: []android.AndroidMkExtraFootersFunc{
			func(w io.Writer, name, prefix, moduleDir string) {
				fmt.Fprintln(w, r.Name()+"-target:", *r.Base+"-target")
			},
		},
	}}
}
