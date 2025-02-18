// Copyright 2025 Google Inc. All rights reserved.
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
	"strings"
)

var (
	fuchsiaArm64SysRoot string = "prebuilts/fuchsia_sdk/fuchsia_bazel_rules/arch/arm64/sysroot"

	fuchsiaArm64Dist string = "prebuilts/fuchsia_sdk/fuchsia_bazel_rules/arch/arm64/dist"

	fuchsiaArm64Lib string = "prebuilts/fuchsia_sdk/fuchsia_bazel_rules/arch/arm64/lib"

	fuchsiaArm64Cflags = []string{
		"-fPIC",
		"-std=c++20",
		"-no-canonical-prefixes",
		"-fno-exceptions",
		"-fno-rtti",
		"-ffunction-sections",
		"-fdata-sections",
		"--target=aarch64-unknown-fuchsia",
		"-Wall",
		"-Wextra",
		"-Werror",
		"--sysroot=" + fuchsiaArm64SysRoot,
	}

	fuchsiaArm64Ldflags = []string{
		"-fPIC",
		"-std=c++20",
		"--driver-mode=g++",
		"--target=aarch64-unknown-fuchsia",
		"-Wl,--gc-sections",
		"-Wl,--no-undefined",
		"--sysroot=" + fuchsiaArm64SysRoot,
		" -L" + fuchsiaArm64Dist,
		" -L" + fuchsiaArm64Lib,
	}

	fuchsiaArm64Lldflags = fuchsiaArm64Ldflags
)

type toolchainFuchsiaArm64 struct {
	toolchainBionic
	toolchain64Bit
}

func (t *toolchainFuchsiaArm64) AvailableLibraries() []string { return nil }

func (t *toolchainFuchsiaArm64) Name() string {
	return "arm64"
}

func (t *toolchainFuchsiaArm64) Cflags() string {
	return strings.Join(fuchsiaArm64Cflags, " ")
}

func (t *toolchainFuchsiaArm64) Cppflags() string {
	return ""
}

func (t *toolchainFuchsiaArm64) Ldflags() string {
	return strings.Join(fuchsiaArm64Ldflags, " ")
}

func (t *toolchainFuchsiaArm64) Lldflags() string {
	return strings.Join(fuchsiaArm64Lldflags, " ")
}

func (t *toolchainFuchsiaArm64) GccTriple() string {
	return "aarch64-unknown-fuchsia"
}

func (t *toolchainFuchsiaArm64) IncludeFlags() string {
	return ""
}

func (t *toolchainFuchsiaArm64) ClangTriple() string {
	return "aarch64-unknown-fuchsia"
}

func (t *toolchainFuchsiaArm64) ClangCppflags() string {
	return strings.Join(fuchsiaArm64Cflags, " ")
}

func (t *toolchainFuchsiaArm64) ClangLdflags() string {
	return strings.Join(fuchsiaArm64Ldflags, " ")
}

func (t *toolchainFuchsiaArm64) ClangLldflags() string {
	return strings.Join(fuchsiaArm64Lldflags, " ")
}

func (t *toolchainFuchsiaArm64) ClangCflags() string {
	return strings.Join(fuchsiaArm64Cflags, " ")
}

func (t *toolchainFuchsiaArm64) Bionic() bool {
	return false
}

func (t *toolchainFuchsiaArm64) ToolchainClangCflags() string {
	return ""
}

func (toolchainFuchsiaArm64) LibclangRuntimeLibraryArch() string {
	return "aarch64"
}

var toolchainArm64FuchsiaSingleton Toolchain = &toolchainFuchsiaArm64{}

func arm64FuchsiaToolchainFactory(arch android.Arch) Toolchain {
	return toolchainArm64FuchsiaSingleton
}

func init() {
	registerToolchainFactory(android.Fuchsia, android.Arm64, arm64FuchsiaToolchainFactory)
}
