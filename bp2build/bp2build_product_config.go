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
	targetProduct := "unknown"
	if cfg.HasDeviceProduct() {
		targetProduct = cfg.DeviceProduct()
	}
	targetBuildVariant := "user"
	if cfg.Eng() {
		targetBuildVariant = "eng"
	} else if cfg.Debuggable() {
		targetBuildVariant = "userdebug"
	}

	productVariablesFileName := cfg.ProductVariablesFileName
	if !strings.HasPrefix(productVariablesFileName, "/") {
		productVariablesFileName = filepath.Join(ctx.topDir, productVariablesFileName)
	}
	bytes, err := os.ReadFile(productVariablesFileName)
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
package(default_visibility=[
    "@soong_injection//build/bazel/product_config:__subpackages__",
    "@//build/bazel/product_config:__subpackages__",
])
load(":soong.variables.bzl", _soong_variables = "variables")
load("@//build/bazel/product_config:utils.bzl", "android_product")

android_product(
    name = "{PRODUCT}-{VARIANT}",
    soong_variables = _soong_variables,
)
`)),
		newFile(
			"build/bazel/product_config/generated",
			"BUILD.bazel",
			productReplacer.Replace(`
package(default_visibility = [
	"@//build/bazel/product_config:__subpackages__",
	"@soong_injection//build/bazel/product_config:__subpackages__",
])

alias(
	name = "current_android_platform",
	actual = select({
		# TODO: When we start generating the platforms for more than just the
		# currently lunched, product, this select should have an arm for each product.
		"@soong_injection//{PRODUCT_FOLDER}:{PRODUCT}-{VARIANT}_constraint_value": "@soong_injection//{PRODUCT_FOLDER}:{PRODUCT}-{VARIANT}",
	}),
)

alias(
	name = "current_linux_x86_64_platform",
	actual = select({
		# TODO: When we start generating the platforms for more than just the
		# currently lunched, product, this select should have an arm for each product.
		"@soong_injection//{PRODUCT_FOLDER}:{PRODUCT}-{VARIANT}_constraint_value": "@soong_injection//{PRODUCT_FOLDER}:{PRODUCT}-{VARIANT}_linux_x86_64",
	}),
)

alias(
	name = "product_vars",
	actual = select({
		# TODO: When we start generating the platforms for more than just the
		# currently lunched, product, this select should have an arm for each product.
		"@soong_injection//{PRODUCT_FOLDER}:{PRODUCT}-{VARIANT}_constraint_value": "@soong_injection//{PRODUCT_FOLDER}:{PRODUCT}-{VARIANT}_product_vars",
	}),
)
`)),
		newFile(
			"build/bazel/product_config/generated",
			"common.bazelrc",
			productReplacer.Replace(`
build --platforms @soong_injection//{PRODUCT_FOLDER}:{PRODUCT}-{VARIANT}_linux_x86_64

build:android --platforms=@soong_injection//{PRODUCT_FOLDER}:{PRODUCT}-{VARIANT}
build:linux_x86_64 --platforms=@soong_injection//{PRODUCT_FOLDER}:{PRODUCT}-{VARIANT}_linux_x86_64
`)),
		newFile(
			"build/bazel/product_config/generated",
			"linux.bazelrc",
			productReplacer.Replace(`
build --host_platform @soong_injection//{PRODUCT_FOLDER}:{PRODUCT}-{VARIANT}_linux_x86_64
`)),
		newFile(
			"build/bazel/product_config/generated",
			"darwin.bazelrc",
			productReplacer.Replace(`
build --host_platform @soong_injection//{PRODUCT_FOLDER}:{PRODUCT}-{VARIANT}_darwin_x86_64
`)),
		newFile(
			"build/bazel/product_config/generated",
			"platform_mappings",
			productReplacer.Replace(`
flags:
  --cpu=k8
    @soong_injection//{PRODUCT_FOLDER}:{PRODUCT}-{VARIANT}
`)),
	}

	return result, nil
}
