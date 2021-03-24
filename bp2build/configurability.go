package bp2build

import "android/soong/android"

// Configurability support for bp2build.

var (
	// A map of architectures to the Bazel label of the constraint_value
	// for the @platforms//cpu:cpu constraint_setting
	platformArchMap = map[android.ArchType]string{
		android.Arm:    "@bazel_tools//platforms:arm",
		android.Arm64:  "@bazel_tools//platforms:aarch64",
		android.X86:    "@bazel_tools//platforms:x86_32",
		android.X86_64: "@bazel_tools//platforms:x86_64",
	}

	// A map of target operating systems to the Bazel label of the
	// constraint_value for the @platforms//os:os constraint_setting
	platformOsMap = map[android.OsType]string{
		android.Android:     "@bazel_tools//platforms:android",
		android.CommonOS:    "@bazel_tools//platforms:android",
		android.Darwin:      "@bazel_tools//platforms:osx",
		android.Linux:       "@bazel_tools//platforms:linux",
		android.LinuxBionic: "@bazel_tools//platforms:linux",
		android.Windows:     "@bazel_tools//platforms:windows",
		// Not listed: Fuchsia, NoOsType
	}
)
