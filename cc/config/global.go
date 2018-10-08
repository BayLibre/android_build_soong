// Copyright 2016 Google Inc. All rights reserved.
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
	"runtime"
	"strings"

	"android/soong/android"
)

var (
	// Flags used by lots of devices.  Putting them in package static variables
	// will save bytes in build.ninja so they aren't repeated for every file
	commonGlobalCflags = []string{
		"-DANDROID",
		"-fmessage-length=0",
		"-W",
		"-Wall",
		"-Wno-unused",
		"-Winit-self",
		"-Wpointer-arith",

		// Make paths in deps files relative
		"-no-canonical-prefixes",

		"-DNDEBUG",
		"-UDEBUG",

		"-fno-exceptions",
		"-Wno-multichar",

		"-O2",
		"-g",

		"-fno-strict-aliasing",

		"-D__compiler_offsetof=__builtin_offsetof",

		// Help catch common 32/64-bit errors.
		"-Werror=int-conversion",

		// Disable overly aggressive warning for macros defined with a leading underscore
		// This happens in AndroidConfig.h, which is included nearly everywhere.
		// TODO: can we remove this now?
		"-Wno-reserved-id-macro",

		// Disable overly aggressive warning for format strings.
		// Bug: 20148343
		"-Wno-format-pedantic",

		// Workaround for ccache with clang.
		// See http://petereisentraut.blogspot.com/2011/05/ccache-and-clang.html.
		"-Wno-unused-command-line-argument",

		// Force clang to always output color diagnostics. Ninja will strip the ANSI
		// color codes if it is not running in a terminal.
		"-fcolor-diagnostics",

		// http://b/29823425 Disable -Wexpansion-to-defined for Clang update to r271374
		"-Wno-expansion-to-defined",

		// http://b/68236239 Allow 0/NULL instead of using nullptr everywhere.
		"-Wno-zero-as-null-pointer-constant",

		// Warnings from clang-7.0
		"-Wno-deprecated-register",
		"-Wno-sign-compare",
	}

	commonGlobalConlyflags = []string{}

	deviceGlobalCflags = []string{
		"-ffunction-sections",
		"-fdata-sections",
		"-fno-short-enums",
		"-funwind-tables",
		"-fstack-protector-strong",
		"-Wa,--noexecstack",
		"-D_FORTIFY_SOURCE=2",

		"-Wstrict-aliasing=2",

		"-Werror=return-type",
		"-Werror=non-virtual-dtor",
		"-Werror=address",
		"-Werror=sequence-point",
		"-Werror=date-time",
		"-Werror=format-security",

		"-nostdlibinc",
	}

	deviceGlobalCppflags = []string{
		"-fvisibility-inlines-hidden",
	}

	deviceGlobalLdflags = []string{
		"-Wl,-z,noexecstack",
		"-Wl,-z,relro",
		"-Wl,-z,now",
		"-Wl,--build-id=md5",
		"-Wl,--warn-shared-textrel",
		"-Wl,--fatal-warnings",
		"-Wl,--no-undefined-version",
	}

	deviceGlobalLldflags = append(ClangFilterUnknownLldflags(deviceGlobalLdflags),
		[]string{
			// TODO(b/109657296): needs --no-rosegment until Android
			// stack unwinder can handle the read-only segment.
			"-Wl,--no-rosegment",
			"-Wl,--pack-dyn-relocs=android",
			"-fuse-ld=lld",
		}...)

	hostGlobalCflags = []string{}

	hostGlobalCppflags = []string{}

	hostGlobalLdflags = []string{}

	hostGlobalLldflags = []string{"-fuse-ld=lld"}

	commonGlobalCppflags = []string{
		"-Wsign-promo",

		// Disable -Winconsistent-missing-override until we can clean up the existing
		// codebase for it.
		"-Wno-inconsistent-missing-override",

		// Bug: http://b/29823425 Disable -Wnull-dereference until the
		// new instances detected by this warning are fixed.
		"-Wno-null-dereference",

		// Enable clang's thread-safety annotations in libcxx.
		// Turn off -Wthread-safety-negative, to avoid breaking projects that use -Weverything.
		"-D_LIBCPP_ENABLE_THREAD_SAFETY_ANNOTATIONS",
		"-Wno-thread-safety-negative",

		// libc++'s math.h has an #include_next outside of system_headers.
		"-Wno-gnu-include-next",
	}

	noOverrideGlobalCflags = []string{
		"-Werror=int-to-pointer-cast",
		"-Werror=pointer-to-int-cast",

		"-Werror=address-of-temporary",
		// Bug: http://b/29823425 Disable -Wnull-dereference until the
		// new cases detected by this warning in Clang r271374 are
		// fixed.
		//"-Werror=null-dereference",
		"-Werror=return-type",

		// http://b/72331526 Disable -Wtautological-* until the instances detected by these
		// new warnings are fixed.
		"-Wno-tautological-constant-compare",
		"-Wno-tautological-type-limit-compare",
		"-Wno-tautological-unsigned-enum-zero-compare",
		"-Wno-tautological-unsigned-zero-compare",

		// http://b/72331524 Allow null pointer arithmetic until the instances detected by
		// this new warning are fixed.
		"-Wno-null-pointer-arithmetic",

		// Disable c++98-specific warning since Android is not concerned with C++98
		// compatibility.
		"-Wno-c++98-compat-extra-semi",

		// Disable this warning until we can fix all instances where it fails.
		"-Wno-self-assign-overloaded",

		// Disable this warning until we can fix all instances where it fails.
		"-Wno-constant-logical-operand",

		// Disable this warning because we don't care about behavior with older compilers.
		"-Wno-return-std-move-in-c++11",

		// Disable this warning until we can fix all instances where it fails.
		"-Wno-dangling-field",
	}

	IllegalFlags = []string{
		"-w",
	}

	CStdVersion               = "gnu99"
	CppStdVersion             = "gnu++14"
	ExperimentalCStdVersion   = "gnu11"
	ExperimentalCppStdVersion = "gnu++1z"

	NdkMaxPrebuiltVersionInt = 27

	// prebuilts/clang default settings.
	ClangDefaultBase         = "prebuilts/clang/host"
	ClangDefaultVersion      = "clang-r339409"
	ClangDefaultShortVersion = "8.0.1"

	// Directories with warnings from Android.bp files.
	WarningAllowedProjects = []string{
		"device/",
		"vendor/",
	}

	// Directories with warnings from Android.mk files.
	WarningAllowedOldProjects = []string{}
)

var pctx = android.NewPackageContext("android/soong/cc/config")

func init() {
	if android.BuildOs == android.Linux {
		commonGlobalCflags = append(commonGlobalCflags, "-fdebug-prefix-map=/proc/self/cwd=")
	}

	pctx.StaticVariable("CommonGlobalConlyflags", strings.Join(commonGlobalConlyflags, " "))
	pctx.StaticVariable("DeviceGlobalCppflags", strings.Join(deviceGlobalCppflags, " "))
	pctx.StaticVariable("DeviceGlobalLdflags", strings.Join(deviceGlobalLdflags, " "))
	pctx.StaticVariable("DeviceGlobalLldflags", strings.Join(deviceGlobalLldflags, " "))
	pctx.StaticVariable("HostGlobalCppflags", strings.Join(hostGlobalCppflags, " "))
	pctx.StaticVariable("HostGlobalLdflags", strings.Join(hostGlobalLdflags, " "))
	pctx.StaticVariable("HostGlobalLldflags", strings.Join(hostGlobalLldflags, " "))

	pctx.StaticVariable("CommonClangGlobalCflags", strings.Join(commonGlobalCflags, " "))
	pctx.StaticVariable("DeviceClangGlobalCflags", strings.Join(deviceGlobalCflags, " "))
	pctx.StaticVariable("HostClangGlobalCflags", strings.Join(hostGlobalCflags, " "))
	pctx.StaticVariable("NoOverrideClangGlobalCflags", strings.Join(noOverrideGlobalCflags, " "))

	pctx.StaticVariable("CommonClangGlobalCppflags", strings.Join(commonGlobalCppflags, " "))

	// Extra cflags for projects under external/ directory to disable warnings that are infeasible
	// to fix in all the external projects and their upstream repos.
	pctx.StaticVariable("ClangExternalCflags", strings.Join([]string{
		"-Wno-enum-compare",
		"-Wno-enum-compare-switch",
	}, " "))

	// Everything in these lists is a crime against abstraction and dependency tracking.
	// Do not add anything to this list.
	pctx.PrefixedExistentPathsForSourcesVariable("CommonGlobalIncludes", "-I",
		[]string{
			"system/core/include",
			"system/media/audio/include",
			"hardware/libhardware/include",
			"hardware/libhardware_legacy/include",
			"hardware/ril/include",
			"libnativehelper/include",
			"frameworks/native/include",
			"frameworks/native/opengl/include",
			"frameworks/av/include",
		})
	// This is used by non-NDK modules to get jni.h. export_include_dirs doesn't help
	// with this, since there is no associated library.
	pctx.PrefixedExistentPathsForSourcesVariable("CommonNativehelperInclude", "-I",
		[]string{"libnativehelper/include_jni"})

	pctx.SourcePathVariable("ClangDefaultBase", ClangDefaultBase)
	pctx.VariableFunc("ClangBase", func(ctx android.PackageVarContext) string {
		if override := ctx.Config().Getenv("LLVM_PREBUILTS_BASE"); override != "" {
			return override
		}
		return "${ClangDefaultBase}"
	})
	pctx.VariableFunc("ClangVersion", func(ctx android.PackageVarContext) string {
		if override := ctx.Config().Getenv("LLVM_PREBUILTS_VERSION"); override != "" {
			return override
		}
		return ClangDefaultVersion
	})
	pctx.StaticVariable("ClangPath", "${ClangBase}/${HostPrebuiltTag}/${ClangVersion}")
	pctx.StaticVariable("ClangBin", "${ClangPath}/bin")
	pctx.StaticVariable("ClangTidyShellPath", "build/soong/scripts/clang-tidy.sh")

	pctx.VariableFunc("ClangShortVersion", func(ctx android.PackageVarContext) string {
		if override := ctx.Config().Getenv("LLVM_RELEASE_VERSION"); override != "" {
			return override
		}
		return ClangDefaultShortVersion
	})
	pctx.StaticVariable("ClangAsanLibDir", "${ClangBase}/linux-x86/${ClangVersion}/lib64/clang/${ClangShortVersion}/lib/linux")
	if runtime.GOOS == "darwin" {
		pctx.StaticVariable("LLVMGoldPlugin", "${ClangPath}/lib64/LLVMgold.dylib")
	} else {
		pctx.StaticVariable("LLVMGoldPlugin", "${ClangPath}/lib64/LLVMgold.so")
	}

	// These are tied to the version of LLVM directly in external/llvm, so they might trail the host prebuilts
	// being used for the rest of the build process.
	pctx.SourcePathVariable("RSClangBase", "prebuilts/clang/host")
	pctx.SourcePathVariable("RSClangVersion", "clang-3289846")
	pctx.SourcePathVariable("RSReleaseVersion", "3.8")
	pctx.StaticVariable("RSLLVMPrebuiltsPath", "${RSClangBase}/${HostPrebuiltTag}/${RSClangVersion}/bin")
	pctx.StaticVariable("RSIncludePath", "${RSLLVMPrebuiltsPath}/../lib64/clang/${RSReleaseVersion}/include")

	pctx.PrefixedExistentPathsForSourcesVariable("RsGlobalIncludes", "-I",
		[]string{
			"external/clang/lib/Headers",
			"frameworks/rs/script_api/include",
		})

	pctx.VariableFunc("CcWrapper", func(ctx android.PackageVarContext) string {
		if override := ctx.Config().Getenv("CC_WRAPPER"); override != "" {
			return override + " "
		}
		return ""
	})
}

var HostPrebuiltTag = pctx.VariableConfigMethod("HostPrebuiltTag", android.Config.PrebuiltOS)

func bionicHeaders(kernelArch string) string {
	return strings.Join([]string{
		"-isystem bionic/libc/include",
		"-isystem bionic/libc/kernel/uapi",
		"-isystem bionic/libc/kernel/uapi/asm-" + kernelArch,
		"-isystem bionic/libc/kernel/android/scsi",
		"-isystem bionic/libc/kernel/android/uapi",
	}, " ")
}
