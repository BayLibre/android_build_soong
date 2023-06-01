// Copyright 2019 The Android Open Source Project
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
	"strings"

	"android/soong/android"
	_ "android/soong/cc/config"
)

var pctx = android.NewPackageContext("android/soong/rust/config")

var (
	RustDefaultVersion = "1.69.0"
	RustDefaultBase    = "prebuilts/rust/"
	DefaultEdition     = "2021"
	Stdlibs            = []string{
		"libstd",
	}

	// Mapping between Soong internal arch types and std::env constants.
	// Required as Rust uses aarch64 when Soong uses arm64.
	StdEnvArch = map[android.ArchType]string{
		android.Arm:    "arm",
		android.Arm64:  "aarch64",
		android.X86:    "x86",
		android.X86_64: "x86_64",
	}

	GlobalRustFlags = []string{
		"-Z remap-cwd-prefix=.",
		"-C codegen-units=1",
		"-C debuginfo=2",
		"-C opt-level=3",
		"-C relocation-model=pic",
		"-C overflow-checks=on",
		"-C force-unwind-tables=yes",
		// Use v0 mangling to distinguish from C++ symbols
		"-C symbol-mangling-version=v0",
		"--color always",
		// TODO (b/267698452): Temporary workaround until the "no unstable
		// features" policy is enforced.
		"-A stable-features",
		"-Zdylib-lto",
	}

	RustLinkerArgs = strings.Join([]string{
		"-Wl,--as-needed",
	}, " ")

	deviceGlobalRustFlags = []string{
		"-C panic=abort",
		"-Z link-native-libraries=no",
		// Generate additional debug info for AutoFDO
		"-Z debug-info-for-profiling",
	}

	deviceGlobalLinkFlags = []string{
		// Prepend the lld flags from cc_config so we stay in sync with cc
		"${cc_config.DeviceGlobalLldflags}",

		// Override cc's --no-undefined-version to allow rustc's generated alloc functions
		"-Wl,--undefined-version",

		"-Wl,-Bdynamic",
		"-nostdlib",
		"-Wl,--pack-dyn-relocs=android+relr",
		"-Wl,--use-android-relr-tags",
		"-Wl,--no-undefined",
		"-B${cc_config.ClangBin}",
	}
)

func init() {
	pctx.SourcePathVariable("RustDefaultBase", RustDefaultBase)
	pctx.VariableConfigMethod("HostPrebuiltTag", func(config android.Config) string {
		return getHostPrebuiltTag(config)
	})
	pctx.VariableFunc("RustBase", func(ctx android.PackageVarContext) string {
		return rustPath(ctx).String()
	})
	pctx.VariableFunc("RustPath", func(ctx android.PackageVarContext) string {
		return rustPath(ctx).String()
	})
	pctx.VariableFunc("RustVersion", getRustVersionPctx)
	pctx.StaticVariable("RustBin", "${RustPath}/bin")

	pctx.ImportAs("cc_config", "android/soong/cc/config")
	pctx.StaticVariable("RustLinker", "${cc_config.ClangBin}/clang++")
	pctx.StaticVariable("RustLinkerArgs", RustLinkerArgs)

	pctx.StaticVariable("DeviceGlobalLinkFlags", strings.Join(deviceGlobalLinkFlags, " "))
}

func getRustVersionPctx(ctx android.PackageVarContext) string {
	return GetRustVersion(ctx)
}

func GetRustVersion(ctx android.PathContext) string {
	if override := ctx.Config().Getenv("RUST_PREBUILTS_VERSION"); override != "" {
		return override
	}
	return RustDefaultVersion
}

func getHostPrebuiltTag(config android.Config) string {
	if config.UseHostMusl() {
		return "linux-musl-x86"
	} else {
		return config.PrebuiltOS()
	}
}

func getRustBase(ctx android.PathContext) string {
	rustBase := RustDefaultBase
	if override := ctx.Config().Getenv("RUST_PREBUILTS_BASE"); override != "" {
		rustBase = override
	}
	return rustBase
}

func RustPath(ctx android.PathContext, file string) android.SourcePath {
	type rustToolKey string

	key := android.NewCustomOnceKey(rustToolKey(file))

	return ctx.Config().OnceSourcePath(key, func() android.SourcePath {
		return rustPath(ctx).Join(ctx, file)
	})
}

var rustPathKey = android.NewOnceKey("rustPath")

func rustPath(ctx android.PathContext) android.SourcePath {
	return ctx.Config().OnceSourcePath(rustPathKey, func() android.SourcePath {
		return android.PathForSource(ctx, getRustBase(ctx), getHostPrebuiltTag(ctx.Config()), GetRustVersion(ctx))
	})
}

func RustToolchainComponentsPaths(ctx android.PathContext) android.Paths {
	return android.Paths{
		RustPath(ctx, "lib/librustc_driver-538952ddf0f7d59a.so"),
		RustPath(ctx, "lib/libstd-e4d585b827a2ecd8.so"),
		RustPath(ctx, "lib/libLLVM-15-rust-dev.so"),
		RustPath(ctx, "lib64/libc++.so.1"),
	}
}
