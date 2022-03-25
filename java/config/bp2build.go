package config

import (
	"android/soong/android"
	cc_config "android/soong/cc/config"
)

var exportedVars = cc_config.NewExportedVariables()

func BazelJavaToolchainVars(config android.Config) string {
	return cc_config.BazelToolchainVars(
		config,
		exportedVars,
	)
}
