package bp2build

import (
	"android/soong/android"
	"android/soong/cc"
	"testing"
)

func runSoongConfigModuleTypeTest(t *testing.T, tc bp2buildTestCase) {
	t.Helper()
	runBp2BuildTestCase(t, registerSoongConfigModuleTypes, tc)
}

type soongConfigTestDefaultsModuleProperties struct {
}

type soongConfigTestDefaultsModule struct {
	android.ModuleBase
	android.DefaultsModuleBase
}

func soongConfigTestDefaultsModuleFactory() android.Module {
	m := &soongConfigTestDefaultsModule{}
	m.AddProperties(&soongConfigTestModuleProperties{})
	android.InitDefaultsModule(m)
	return m
}

type soongConfigTestModule struct {
	android.ModuleBase
	android.DefaultableModuleBase
	props soongConfigTestModuleProperties
}

type soongConfigTestModuleProperties struct {
	Cflags []string
}

func soongConfigTestModuleFactory() android.Module {
	m := &soongConfigTestModule{}
	m.AddProperties(&m.props)
	android.InitAndroidModule(m)
	android.InitDefaultableModule(m)
	return m
}

func (t soongConfigTestModule) GenerateAndroidBuildActions(android.ModuleContext) {}

func registerSoongConfigModuleTypes(ctx android.RegistrationContext) {
	cc.RegisterCCBuildComponents(ctx)

	ctx.RegisterModuleType("soong_config_module_type_import", android.SoongConfigModuleTypeImportFactory)
	ctx.RegisterModuleType("soong_config_module_type", android.SoongConfigModuleTypeFactory)
	ctx.RegisterModuleType("soong_config_string_variable", android.SoongConfigStringVariableDummyFactory)
	ctx.RegisterModuleType("soong_config_bool_variable", android.SoongConfigBoolVariableDummyFactory)
	ctx.RegisterModuleType("test_defaults", soongConfigTestDefaultsModuleFactory)
	ctx.RegisterModuleType("test", soongConfigTestModuleFactory)
}

func TestSoongConfigModuleType_WithConditionsDefault(t *testing.T) {
	configBp := `
		soong_config_module_type {
			name: "acme_test",
			module_type: "cc_defaults",
			config_namespace: "acme",
			variables: ["board", "feature1", "FEATURE3", "unused_string_var"],
			bool_variables: ["feature2", "unused_feature"],
			value_variables: ["size", "unused_size"],
			properties: ["cflags", "srcs", "defaults"],
		}

		soong_config_string_variable {
			name: "board",
			values: ["soc_a", "soc_b", "soc_c", "soc_d"],
		}

		soong_config_string_variable {
			name: "unused_string_var",
			values: ["a", "b"],
		}

		soong_config_bool_variable {
			name: "feature1",
		}

		soong_config_bool_variable {
			name: "FEATURE3",
		}
	`

	bp := `
acme_test {
	name: "foo",
	cflags: ["-DGENERIC"],
	soong_config_variables: {
		board: {
			soc_a: {
				cflags: ["-DSOC_A"],
			},
			soc_b: {
				cflags: ["-DSOC_B"],
			},
			soc_c: {},
			conditions_default: {
				cflags: ["-DSOC_CONDITIONS_DEFAULT"],
			},
		},
		size: {
			cflags: ["-DSIZE=%s"],
			conditions_default: {
				cflags: ["-DSIZE=CONDITIONS_DEFAULT"],
			},
		},
		feature1: {
			conditions_default: {
				cflags: ["-DF1_CONDITIONS_DEFAULT"],
			},
			cflags: ["-DFEATURE1"],
		},
		feature2: {
			cflags: ["-DFEATURE2"],
			conditions_default: {
				cflags: ["-DF2_CONDITIONS_DEFAULT"],
			},
		},
		FEATURE3: {
			cflags: ["-DFEATURE3"],
		},
	},
}

cc_library_static {
    name: "foo_library",
    defaults: ["foo"],
    bazel_module: { bp2build_available: true },
}
`

	runSoongConfigModuleTypeTest(t, bp2buildTestCase{
		description:                        "soong config variables",
		moduleTypeUnderTest:                "cc_library_static",
		moduleTypeUnderTestFactory:         cc.LibraryStaticFactory,
		moduleTypeUnderTestBp2BuildMutator: cc.CcLibraryStaticBp2Build,
		filesystem:                         map[string]string{},
		blueprint:                          configBp + bp,
		expectedBazelTargets: []string{`cc_library_static(
    name = "foo_library",
    copts = ["-DGENERIC"] + select({
        "//build/bazel/product_variables/vendor/namespace:conditions_default": ["-DSOC_CONDITIONS_DEFAULT"],
        "//build/bazel/product_variables/vendor/namespace:soc_a": ["-DSOC_A"],
        "//build/bazel/product_variables/vendor/namespace:soc_b": ["-DSOC_B"],
        "//conditions:default": [],
    }) + select({
        "//build/bazel/product_variables/vendor/namespace:feature3": ["-DFEATURE3"],
        "//conditions:default": [],
    }) + select({
        "//build/bazel/product_variables/vendor/namespace:feature1": ["-DFEATURE1"],
        "//conditions:default": ["-DF1_CONDITIONS_DEFAULT"],
    }) + select({
        "//build/bazel/product_variables/vendor/namespace:feature2": ["-DFEATURE2"],
        "//conditions:default": [],
    }) + select({
        "//build/bazel/product_variables/vendor/namespace:size": ["-DSIZE=$(Size)"],
        "//conditions:default": [],
    }),
    local_includes = ["."],
)`}})
}

func TestSoongConfigModuleType_LoadFromFile(t *testing.T) {
	configBp := `
		soong_config_module_type {
			name: "acme_test",
			module_type: "cc_defaults",
			config_namespace: "acme",
			variables: ["board", "feature1", "FEATURE3", "unused_string_var"],
			bool_variables: ["feature2", "unused_feature"],
			value_variables: ["size", "unused_size"],
			properties: ["cflags", "srcs", "defaults"],
		}

		soong_config_string_variable {
			name: "board",
			values: ["soc_a", "soc_b", "soc_c", "soc_d"],
		}

		soong_config_string_variable {
			name: "unused_string_var",
			values: ["a", "b"],
		}

		soong_config_bool_variable {
			name: "feature1",
		}

		soong_config_bool_variable {
			name: "FEATURE3",
		}
	`

	importBp := `
		soong_config_module_type_import {
			from: "path/to/SoongConfig.bp",
			module_types: ["acme_test"],
		}
	`

	bp := `
acme_test {
	name: "foo",
	cflags: ["-DGENERIC"],
	soong_config_variables: {
		board: {
			soc_a: {
				cflags: ["-DSOC_A"],
			},
			soc_b: {
				cflags: ["-DSOC_B"],
			},
			soc_c: {},
			conditions_default: {
				cflags: ["-DSOC_CONDITIONS_DEFAULT"],
			},
		},
		size: {
			cflags: ["-DSIZE=%s"],
			conditions_default: {
				cflags: ["-DSIZE=CONDITIONS_DEFAULT"],
			},
		},
		feature1: {
			conditions_default: {
				cflags: ["-DF1_CONDITIONS_DEFAULT"],
			},
			cflags: ["-DFEATURE1"],
		},
		feature2: {
			cflags: ["-DFEATURE2"],
			conditions_default: {
				cflags: ["-DF2_CONDITIONS_DEFAULT"],
			},
		},
		FEATURE3: {
			cflags: ["-DFEATURE3"],
		},
	},
}

cc_library_static {
    name: "foo_library",
    defaults: ["foo"],
    bazel_module: { bp2build_available: true },
}
`

	runSoongConfigModuleTypeTest(t, bp2buildTestCase{
		description:                        "soong config variables - import module type from another file",
		moduleTypeUnderTest:                "cc_library_static",
		moduleTypeUnderTestFactory:         cc.LibraryStaticFactory,
		moduleTypeUnderTestBp2BuildMutator: cc.CcLibraryStaticBp2Build,
		filesystem: map[string]string{
			"path/to/SoongConfig.bp": configBp,
		},
		blueprint: importBp + bp,
		expectedBazelTargets: []string{`cc_library_static(
    name = "foo_library",
    copts = ["-DGENERIC"] + select({
        "//build/bazel/product_variables/vendor/namespace:conditions_default": ["-DSOC_CONDITIONS_DEFAULT"],
        "//build/bazel/product_variables/vendor/namespace:soc_a": ["-DSOC_A"],
        "//build/bazel/product_variables/vendor/namespace:soc_b": ["-DSOC_B"],
        "//conditions:default": [],
    }) + select({
        "//build/bazel/product_variables/vendor/namespace:feature3": ["-DFEATURE3"],
        "//conditions:default": [],
    }) + select({
        "//build/bazel/product_variables/vendor/namespace:feature1": ["-DFEATURE1"],
        "//conditions:default": ["-DF1_CONDITIONS_DEFAULT"],
    }) + select({
        "//build/bazel/product_variables/vendor/namespace:feature2": ["-DFEATURE2"],
        "//conditions:default": [],
    }) + select({
        "//build/bazel/product_variables/vendor/namespace:size": ["-DSIZE=$(Size)"],
        "//conditions:default": [],
    }),
    local_includes = ["."],
)`}})
}

func TestSoongConfigModuleType_WithDefaults(t *testing.T) {
	t.Skip("Skipped - this is not implemented yet. Tracking in b/204281595")

	configBp := `
		soong_config_module_type {
			name: "acme_test",
			module_type: "cc_defaults",
			config_namespace: "acme",
			variables: ["board", "feature1", "FEATURE3", "unused_string_var"],
			bool_variables: ["feature2", "unused_feature"],
			value_variables: ["size", "unused_size"],
			properties: ["cflags", "srcs", "defaults"],
		}

		soong_config_string_variable {
			name: "board",
			values: ["soc_a", "soc_b", "soc_c", "soc_d"],
		}

		soong_config_string_variable {
			name: "unused_string_var",
			values: ["a", "b"],
		}

		soong_config_bool_variable {
			name: "feature1",
		}

		soong_config_bool_variable {
			name: "FEATURE3",
		}
	`

	bp := `
test_defaults {
	name: "foo_defaults",
	cflags: ["DEFAULT"],
}

test_defaults {
	name: "foo_defaults_a",
	cflags: ["DEFAULT_A"],
}

test_defaults {
	name: "foo_defaults_b",
	cflags: ["DEFAULT_B"],
}

acme_test {
	name: "foo_with_defaults",
	cflags: ["-DGENERIC"],
	defaults: ["foo_defaults"],
	soong_config_variables: {
		board: {
			soc_a: {
				cflags: ["-DSOC_A"],
				defaults: ["foo_defaults_a"],
			},
			soc_b: {
				cflags: ["-DSOC_B"],
				defaults: ["foo_defaults_b"],
			},
			soc_c: {},
		},
		size: {
			cflags: ["-DSIZE=%s"],
		},
		feature1: {
			cflags: ["-DFEATURE1"],
		},
		feature2: {
			cflags: ["-DFEATURE2"],
		},
		FEATURE3: {
			cflags: ["-DFEATURE3"],
		},
	},
}

cc_library_static {
    name: "foo_library_with_defaults",
    defaults: ["foo_with_defaults"],
    bazel_module: { bp2build_available: true },
}
`
	runSoongConfigModuleTypeTest(t, bp2buildTestCase{
		description:                        "soong config variables with defaults",
		moduleTypeUnderTest:                "cc_library_static",
		moduleTypeUnderTestFactory:         cc.LibraryStaticFactory,
		moduleTypeUnderTestBp2BuildMutator: cc.CcLibraryStaticBp2Build,
		filesystem:                         map[string]string{},
		blueprint:                          configBp + bp,
		expectedBazelTargets:               []string{
			// TODO(b/204281595)
		}})
}
