package android

import (
	"testing"

	"github.com/google/blueprint"
)

var licensesTests = []struct {
	name                string
	fs                  map[string][]byte
	expectedErrors      []string
	effectiveLicenses   map[string][]string
}{
	{
		name: "invalid module type without licenses property",
		fs: map[string][]byte{
			"top/Blueprints": []byte(`
				mock_bad_module {
					name: "libexample",
				}`),
		},
		expectedErrors: []string{`module type "mock_bad_module" must have an applicable licenses property`},
	},
	{
		name: "license must exist",
		fs: map[string][]byte{
			"top/Blueprints": []byte(`
				mock_library {
					name: "libexample",
					licenses: ["notice"],
				}`),
		},
		expectedErrors: []string{`"libexample" depends on undefined module "notice"`},
	},
	{
		name: "all good",
		fs: map[string][]byte{
			"top/Blueprints": []byte(`
				license_kind {
					name: "notice",
				}

				license {
					name: "top_Apache2",
					license_kinds: ["notice"],
				}

				mock_library {
					name: "libexample",
					licenses: ["top_Apache2"],
				}`),
			"top/nested/Blueprints": []byte(`
				mock_library {
					name: "libnested",
					licenses: ["top_Apache2"],
				}`),
			"other/Blueprints": []byte(`
				mock_library {
					name: "libother",
					licenses: ["top_Apache2"],
				}`),
		},
		effectiveLicenses: map[string][]string{
			"libexample": []string{"top_Apache2"},
			"libnested": []string{"top_Apache2"},
			"libother": []string{"top_Apache2"},
		},
	},

	// Defaults propagation tests
	{
		// Check that licenses is the union of the defaults modules.
		name: "defaults union, basic",
		fs: map[string][]byte{
			"top/Blueprints": []byte(`
				license_kind {
					name: "top_notice",
				}

				license {
					name: "top_other",
					license_kinds: ["top_notice"],
				}

				mock_defaults {
					name: "libexample_defaults",
					licenses: ["top_other"],
				}
				mock_library {
					name: "libexample",
					licenses: ["nested_other"],
					defaults: ["libexample_defaults"],
				}
				mock_library {
					name: "libsamepackage",
					deps: ["libexample"],
				}`),
			"top/nested/Blueprints": []byte(`
				license_kind {
					name: "nested_notice",
				}

				license {
					name: "nested_other",
					license_kinds: ["nested_notice"],
				}

				mock_library {
					name: "libnested",
					deps: ["libexample"],
				}`),
			"other/Blueprints": []byte(`
				mock_library {
					name: "libother",
					deps: ["libexample"],
				}`),
		},
		effectiveLicenses: map[string][]string{
			"libexample": []string{"nested_other", "top_other"},
			"libsamepackage": []string{"nested_other", "top_other"},
			"libnested": []string{"nested_other", "top_other"},
			"libother": []string{"nested_other", "top_other"},
		},
	},
	{
		name: "defaults union, multiple defaults",
		fs: map[string][]byte{
			"top/Blueprints": []byte(`
				license {
					name: "top",
				}
				mock_defaults {
					name: "libexample_defaults_1",
					licenses: ["other"],
				}
				mock_defaults {
					name: "libexample_defaults_2",
					licenses: ["top_nested"],
				}
				mock_library {
					name: "libexample",
					defaults: ["libexample_defaults_1", "libexample_defaults_2"],
				}
				mock_library {
					name: "libsamepackage",
					deps: ["libexample"],
				}`),
			"top/nested/Blueprints": []byte(`
				license {
					name: "top_nested",
				}
				mock_library {
					name: "libnested",
					deps: ["libexample"],
				}`),
			"other/Blueprints": []byte(`
				license {
					name: "other",
				}
				mock_library {
					name: "libother",
					deps: ["libexample"],
				}`),
			"outsider/Blueprints": []byte(`
				mock_library {
					name: "liboutsider",
					deps: ["libexample"],
				}`),
		},
		effectiveLicenses: map[string][]string{
			"libexample": []string{"other", "top_nested"},
			"libsamepackage": []string{"other", "top_nested"},
			"libnested": []string{"other", "top_nested"},
			"libother": []string{"other", "top_nested"},
			"liboutsider": []string{"other", "top_nested"},
		},
	},

	// Defaults module's defaults_licenses tests
	{
		name: "defaults_licenses invalid",
		fs: map[string][]byte{
			"top/Blueprints": []byte(`
				mock_defaults {
					name: "top_defaults",
					licenses: ["notice"],
				}`),
		},
		expectedErrors: []string{`"top_defaults" depends on undefined module "notice"`},
	},
	{
		name: "defaults_licenses overrides package default",
		fs: map[string][]byte{
			"top/Blueprints": []byte(`
				package {
					default_applicable_licenses: ["by_exception_only"],
				}
				license {
					name: "by_exception_only",
				}
				license {
					name: "notice",
				}
				mock_defaults {
					name: "top_defaults",
					licenses: ["notice"],
				}
				mock_library {
					name: "libexample",
				}
				mock_library {
					name: "libdefaults",
					defaults: ["top_defaults"],
				}`),
		},
		effectiveLicenses: map[string][]string{
			"libexample": []string{"by_exception_only"},
			"libdefaults": []string{"notice"},
		},
	},

	// Package default_applicable_licenses tests
	{
		name: "package default_applicable_licenses must exist",
		fs: map[string][]byte{
			"top/Blueprints": []byte(`
				package {
					default_applicable_licenses: ["notice"],
				}`),
		},
		expectedErrors: []string{`"//top" depends on undefined module "notice"`},
	},
	{
		// This test relies on the default licenses being legacy_public.
		name: "package default_applicable_licenses property used when no licenses specified",
		fs: map[string][]byte{
			"top/Blueprints": []byte(`
				package {
					default_applicable_licenses: ["top_notice"],
				}

				license {
					name: "top_notice",
				}
				mock_library {
					name: "libexample",
				}`),
			"outsider/Blueprints": []byte(`
				mock_library {
					name: "liboutsider",
					deps: ["libexample"],
				}`),
		},
		effectiveLicenses: map[string][]string{
			"libexample": []string{"top_notice"},
			"liboutsider": []string{"top_notice"},
		},
	},
	{
		name: "package default_applicable_licenses not inherited to subpackages",
		fs: map[string][]byte{
			"top/Blueprints": []byte(`
				package {
					default_applicable_licenses: ["top_notice"],
				}
				license {
					name: "top_notice",
				}
				mock_library {
					name: "libexample",
				}`),
			"top/nested/Blueprints": []byte(`
				package {
					default_applicable_licenses: ["outsider"],
				}

				mock_library {
					name: "libnested",
				}`),
			"top/other/Blueprints": []byte(`
				mock_library {
					name: "libother",
				}`),
			"outsider/Blueprints": []byte(`
				license {
					name: "outsider",
				}
				mock_library {
					name: "liboutsider",
					deps: ["libexample", "libother", "libnested"],
				}`),
		},
		effectiveLicenses: map[string][]string{
			"libexample": []string{"top_notice"},
			"libnested": []string{"outsider"},
			"libother": []string{},
			"liboutsider": []string{"top_notice", "outsider"},
		},
	},
	{
		name: "verify that prebuilt dependencies are included",
		fs: map[string][]byte{
			"prebuilts/Blueprints": []byte(`
				license {
					name: "prebuilt"
				}
				prebuilt {
					name: "module",
					licenses: ["prebuilt"],
				}`),
			"top/sources/source_file": nil,
			"top/sources/Blueprints": []byte(`
				license {
					name: "top_sources"
				}
				source {
					name: "module",
					licenses: ["top_sources"],
				}`),
			"top/other/source_file": nil,
			"top/other/Blueprints": []byte(`
				source {
					name: "other",
					deps: [":module"],
				}`),
		},
		effectiveLicenses: map[string][]string{
			"other": []string{"prebuilt", "top_sources"},
		},
	},
	{
		name: "verify that prebuilt dependencies are ignored for licenses reasons (preferred)",
		fs: map[string][]byte{
			"prebuilts/Blueprints": []byte(`
				license {
					name: "prebuilt"
				}
				prebuilt {
					name: "module",
					licenses: ["prebuilt"],
					prefer: true,
				}`),
			"top/sources/source_file": nil,
			"top/sources/Blueprints": []byte(`
				license {
					name: "top_sources"
				}
				source {
					name: "module",
					licenses: ["top_sources"],
				}`),
			"top/other/source_file": nil,
			"top/other/Blueprints": []byte(`
				source {
					name: "other",
					deps: [":module"],
				}`),
		},
		effectiveLicenses: map[string][]string{
			"module": []string{"prebuilt", "top_sources"},
			"other": []string{"prebuilt", "top_sources"},
		},
	},
}

func TestLicenses(t *testing.T) {
	for _, test := range licensesTests {
		t.Run(test.name, func(t *testing.T) {
			ctx, errs := testLicenses(buildDir, test.fs)

			CheckErrorsAgainstExpectations(t, errs, test.expectedErrors)

			if test.effectiveLicenses != nil {
				checkEffectiveLicenses(t, ctx, test.effectiveLicenses)
			}
		})
	}
}

func checkEffectiveLicenses(t *testing.T, ctx *TestContext, effectiveLicenses map[string][]string) {
	actualLicenses := make(map[string][]string)
	ctx.Context.Context.VisitAllModules(func(m blueprint.Module) {
		if _, ok := m.(*licenseModule); ok {
			return
		}
		if _, ok := m.(*licenseKindModule); ok {
			return
		}
		if _, ok := m.(*packageModule); ok {
			return
		}
		module, ok := m.(Module)
		if !ok {
			t.Errorf("%q not a module", m.Name())
			return
		}
		base := module.base()
		if base == nil {
			return
		}
		actualLicenses[m.Name()] = base.commonProperties.Effective_licenses
	})

	for moduleName, expectedLicenses := range effectiveLicenses {
		licenses, ok := actualLicenses[moduleName]
		if !ok {
			licenses = []string{}
		}
		if !compareLicenses(expectedLicenses, licenses) {
			t.Errorf("effective licenses mismatch for module %q: expected %q, found %q", moduleName, expectedLicenses, licenses)
		}
	}
}

func compareLicenses(expected, actual []string) bool {
	if len(expected) != len(actual) {
		return false
	}
	s := make(map[string]int)
	for _, v := range expected {
		s[v] += 1
	}
	for _, v := range actual {
		c, ok := s[v]
		if !ok {
			return false
		}
		if c < 1 {
			return false
		}
		s[v] -= 1
	}
	return true
}

func testLicenses(buildDir string, fs map[string][]byte) (*TestContext, []error) {

	// Create a new config per test as licenses information is stored in the config.
	config := TestArchConfig(buildDir, nil, "", fs)

	ctx := NewTestArchContext()
	ctx.RegisterModuleType("mock_bad_module", newMockLicensesBadModule)
	ctx.RegisterModuleType("mock_library", newMockLicensesLibraryModule)
	ctx.RegisterModuleType("mock_defaults", defaultsLicensesFactory)

	// Order of the following method calls is significant.
	RegisterPackageBuildComponents(ctx)
	registerTestPrebuiltBuildComponents(ctx)
	RegisterLicenseKindBuildComponents(ctx)
	RegisterLicenseBuildComponents(ctx)
	ctx.PreArchMutators(RegisterVisibilityRuleChecker)
	ctx.PreArchMutators(RegisterLicensesPackageMapper)
	ctx.PreArchMutators(RegisterDefaultsPreArchMutators)
	ctx.PreArchMutators(RegisterLicensesPropertyGatherer)
	ctx.PreArchMutators(RegisterVisibilityRuleGatherer)
	ctx.PostDepsMutators(RegisterVisibilityRuleEnforcer)
	ctx.PostDepsMutators(RegisterLicensesPropertyExpander)
	ctx.Register(config)

	_, errs := ctx.ParseBlueprintsFiles(".")
	if len(errs) > 0 {
		return ctx, errs
	}

	_, errs = ctx.PrepareBuildActions(config)
	return ctx, errs
}

type mockLicensesBadProperties struct {
	Visibility []string
}

type mockLicensesBadModule struct {
	ModuleBase
	DefaultableModuleBase
	properties mockLicensesBadProperties
}

func newMockLicensesBadModule() Module {
	m := &mockLicensesBadModule{}

	base := m.base()
	m.AddProperties(&base.nameProperties, &m.properties)

	base.generalProperties = m.GetProperties()
	base.customizableProperties = m.GetProperties()

	// The default_visibility property needs to be checked and parsed by the visibility module during
	// its checking and parsing phases so make it the primary visibility property.
	setPrimaryVisibilityProperty(m, "visibility", &m.properties.Visibility)

	initAndroidModuleBase(m)
	InitDefaultableModule(m)

	return m
}

func (m *mockLicensesBadModule) GenerateAndroidBuildActions(ModuleContext) {
}

type mockLicensesLibraryProperties struct {
	Deps []string
}

type mockLicensesLibraryModule struct {
	ModuleBase
	DefaultableModuleBase
	properties mockLicensesLibraryProperties
}

func newMockLicensesLibraryModule() Module {
	m := &mockLicensesLibraryModule{}
	m.AddProperties(&m.properties)
	InitAndroidArchModule(m, HostAndDeviceSupported, MultilibCommon)
	InitDefaultableModule(m)
	return m
}

type dependencyLicensesTag struct {
	blueprint.BaseDependencyTag
	name string
}

func (j *mockLicensesLibraryModule) DepsMutator(ctx BottomUpMutatorContext) {
	ctx.AddVariationDependencies(nil, dependencyLicensesTag{name: "mockdeps"}, j.properties.Deps...)
}

func (p *mockLicensesLibraryModule) GenerateAndroidBuildActions(ModuleContext) {
}

type mockLicensesDefaults struct {
	ModuleBase
	DefaultsModuleBase
}

func defaultsLicensesFactory() Module {
	m := &mockLicensesDefaults{}
	InitDefaultsModule(m)
	return m
}
