package config

import (
	"strings"

	"android/soong/android"
	cc_config "android/soong/cc/config"
)

// Helpers for exporting cc configuration information to Bazel.
var (
	// Maps containing toolchain variables that are independent of the
	// environment variables of the build.
	exportedStringVars          = cc_config.ExportedStringVariables{}
	exportedStringListVars      = cc_config.ExportedStringListVariables{}
	exportedConfigDependingVars = cc_config.ExportedConfigDependingVariables{}
)

func exportStringStaticVariable(name string, value string) {
	pctx.StaticVariable(name, value)
	exportedStringVars.Set(name, value)
}

// Convenience function to declare a static variable and export it to Bazel's cc_toolchain.
func exportStringListStaticVariable(name string, value []string) {
	pctx.StaticVariable(name, strings.Join(value, " "))
	exportedStringListVars.Set(name, value)
}

func BazelJavaToolchainVars(config android.Config) string {
	return cc_config.BazelToolchainVars(
		config,
		exportedStringVars,
		exportedStringListVars,
		exportedConfigDependingVars,
	)
}
