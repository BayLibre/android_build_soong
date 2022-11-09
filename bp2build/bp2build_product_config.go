package bp2build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func CreateProductConfigFiles(
	ctx *CodegenContext) ([]BazelFile, error) {
	cfg := &ctx.config
	targetProduct := cfg.DeviceProduct()
	targetBuildVariant := "user"
	if cfg.Eng() {
		targetBuildVariant = "eng"
	} else if cfg.Debuggable() {
		targetBuildVariant = "userdebug"
	}

	bytes, err := os.ReadFile(filepath.Join(ctx.topDir, cfg.ProductVariablesFileName))
	if err != nil {
		return nil, err
	}

	currentProductFolder := fmt.Sprintf("build/bazel/product_config/generated/products/%s-%s", targetProduct, targetBuildVariant)

	productReplacer := strings.NewReplacer(
		"{PRODUCT}", targetProduct,
		"{VARIANT}", targetBuildVariant,
		"{PRODUCT_FOLDER}", currentProductFolder)

	result := []BazelFile{
		newFile(
			currentProductFolder,
			"soong.variables.bzl",
			`variables = json.decode("""`+strings.ReplaceAll(string(bytes), "\\", "\\\\")+`""")`),
		newFile(
			currentProductFolder,
			"BUILD",
			productReplacer.Replace(`
package(default_visibility=["//build/bazel/product_config:__subpackages__"])
load(":soong.variables.bzl", _soong_variables = "variables")
load("//build/bazel/product_config:utils.bzl", "android_product")

android_product(
    name = "{PRODUCT}-{VARIANT}",
    soong_variables = _soong_variables,
)

# def _default_android_transition_impl(settings, attr):
#     # Ensure that this target is always built for the android target platform.
#     #
#     # For example, there is currently no support for building an APEX for the
#     # host (e.g. linux_x86_64) or other platforms like darwin or windows.
#     #
#     # This is further enforced by the toolchains for these types (apex,
#     # partition) being compatible with only //build/bazel/platforms/os:android
#     # for their target platform.  If we don't do this, an apex can be
#     # accidentally requested for a non-android target platform, resulting in
#     # toolchain resolution failures.
# 
#     return {
#         "//command_line_option:platforms": ["//{PRODUCT_FOLDER}:{PRODUCT}-{VARIANT}"],
#     }
# 
# # A transition to always enforce an android target platform. Useful for targets
# # that only has toolchains for building against android (and nothing else), like
# # APEXes.
# default_android_transition = transition(
#     implementation = _default_android_transition_impl,
#     outputs = ["//command_line_option:platforms"],
# )
`)),
		newFile(
			"build/bazel/product_config/generated",
			"BUILD.bazel",
			productReplacer.Replace(`
alias(
	name = "lunched_android_platform",
	actual = "//{PRODUCT_FOLDER}:{PRODUCT}-{VARIANT}"
)

alias(
	name = "lunched_android_platform_linux_x86_64",
	actual = "//{PRODUCT_FOLDER}:{PRODUCT}-{VARIANT}_linux_x86_64"
)

alias(
	name = "lunched_android_platform_darwin_x86_64",
	actual = "//{PRODUCT_FOLDER}:{PRODUCT}-{VARIANT}_darwin_x86_64"
)

alias(
	name = "product_vars",
	actual = select({
		# TODO: When we start generating the platforms for more than just the
		# currently lunched, product, this select should have an arm for each product.
		"//{PRODUCT_FOLDER}:{PRODUCT}-{VARIANT}_constraint_value": "//{PRODUCT_FOLDER}:{PRODUCT}-{VARIANT}_product_vars",
	}),
	visibility = ["//build/bazel/product_config:__subpackages__"],
)
`)),
		newFile(
			"build/bazel/product_config/generated",
			"common.bazelrc",
			productReplacer.Replace(`
build --platform_mappings=build/bazel/product_config/generated/platform_mappings
build --platforms //{PRODUCT_FOLDER}:{PRODUCT}-{VARIANT}_linux_x86_64

build:android --platforms=//{PRODUCT_FOLDER}:{PRODUCT}-{VARIANT}
build:linux_x86_64 --platforms=//{PRODUCT_FOLDER}:{PRODUCT}-{VARIANT}_linux_x86_64
`)),
		newFile(
			"build/bazel/product_config/generated",
			"linux.bazelrc",
			productReplacer.Replace(`
build --host_platform //{PRODUCT_FOLDER}:{PRODUCT}-{VARIANT}_linux_x86_64
`)),
		newFile(
			"build/bazel/product_config/generated",
			"darwin.bazelrc",
			productReplacer.Replace(`
build --host_platform //{PRODUCT_FOLDER}:{PRODUCT}-{VARIANT}_darwin_x86_64
`)),
		newFile(
			"build/bazel/product_config/generated",
			"platform_mappings",
			productReplacer.Replace(`
flags:
  --cpu=k8
    //{PRODUCT_FOLDER}:{PRODUCT}-{VARIANT}
`)),
	}

	return result, nil
}
