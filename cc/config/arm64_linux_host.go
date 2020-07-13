// Copyright 2020 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"android/soong/android"
	"fmt"
	"strings"
)

var (
	// These flags are copied from x86_linux_bionic_host with one
	// exception: --gcc-toolchain=${LinuxBionicGccRoot} which is set to x86
	// gcc toolchain.
	linuxCrossCflags = ClangFilterUnknownCflags([]string{
		"-fdiagnostics-color",

		"-Wa,--noexecstack",

		"-fPIC",

		"-U_FORTIFY_SOURCE",
		"-D_FORTIFY_SOURCE=2",
		"-fstack-protector-strong",

		// From x86_64_device
		"-ffunction-sections",
		"-finline-functions",
		"-finline-limit=300",
		"-fno-short-enums",
		"-funswitch-loops",
		"-funwind-tables",
		"-fno-canonical-system-headers",

		// This is normally in ClangExtraTargetCflags, but this is considered host
		"-nostdlibinc",
	})

	linuxCrossLdflags = ClangFilterUnknownCflags([]string{
		"-Wl,-z,noexecstack",
		"-Wl,-z,relro",
		"-Wl,-z,now",
		"-Wl,--build-id=md5",
		"-Wl,--warn-shared-textrel",
		"-Wl,--fatal-warnings",
		"-Wl,--hash-style=gnu",
		"-Wl,--no-undefined-version",
	})
)

func init() {
	pctx.StaticVariable("LinuxCrossCflags", strings.Join(linuxCrossCflags, " "))
	pctx.StaticVariable("LinuxCrossLdflags", strings.Join(linuxCrossLdflags, " "))
}

// toolchain config for ARM64 Linux CrossHost. Almost everything is the same as the ARM64 Android
// target. The overridden methods below show the differences.
type toolchainLinuxArm64 struct {
	toolchainArm64
}

func (toolchainLinuxArm64) ClangTriple() string {
	// Note the absence of "-android" suffix. The compiler won't define __ANDROID__
	return "aarch64-linux"
}

func (toolchainLinuxArm64) ClangCflags() string {
	// The inherited flags + extra flags
	return "${config.Arm64ClangCflags} ${config.LinuxCrossCflags}"
}

func linuxArm64ToolchainFactory(arch android.Arch) Toolchain {
	archVariant := "armv8-a" // default to armv8-a
	if arch.ArchVariant != "" {
		archVariant = arch.ArchVariant
	}
	switch archVariant {
	case "armv8-a":
	case "armv8-2a":
	default:
		panic(fmt.Sprintf("Unknown ARM architecture version: %q", archVariant))
	}

	toolchainClangCflags := []string{arm64ClangArchVariantCflagsVar[arch.ArchVariant]}

	// TODO(jiyong): uncomment below when CPU variants can be specified for CrossHost
	//toolchainClangCflags = append(toolchainClangCflags,
	//	variantOrDefault(arm64ClangCpuVariantCflagsVar, archVariant))

	// We don't specify CPU architecture for CrossHost. Conservatively assume
	// the host cross CPU needs the fix
	// TODO(jiyong): remove this when CPU variants can be specified for CrossHost
	extraLdflags := "-Wl,--fix-cortex-a53-843419"

	ret := toolchainLinuxArm64{}

	// add the extra ld and lld flags
	ret.toolchainArm64.ldflags = strings.Join([]string{
		"${config.Arm64Ldflags}",
		"${config.LinuxCrossLdflags}",
		extraLdflags,
	}, " ")
	ret.toolchainArm64.lldflags = strings.Join([]string{
		"${config.Arm64Lldflags}",
		"${config.LinuxCrossLdflags}",
		extraLdflags,
	}, " ")
	ret.toolchainArm64.toolchainClangCflags = strings.Join(toolchainClangCflags, " ")
	return &ret
}

func init() {
	registerToolchainFactory(android.LinuxCross, android.Arm64, linuxArm64ToolchainFactory)
}
