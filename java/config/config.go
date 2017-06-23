package config

import "android/soong/android"

var (
	DefaultLibraries = []string{"core-oj", "core-libart", "ext", "framework", "okhttp"}
)

var pctx = android.NewPackageContext("android/soong/javac/config")
