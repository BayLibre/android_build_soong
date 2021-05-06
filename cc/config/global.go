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
	"fmt"
	"strings"

	"android/soong/android"
	"android/soong/remoteexec"
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
		"-Wunreachable-code-loop-increment",

		// Make paths in deps files relative
		"-no-canonical-prefixes",
		"-fno-canonical-system-headers",

		"-DNDEBUG",
		"-UDEBUG",

		"-fno-exceptions",
		"-Wno-multichar",

		"-O2",
		"-g",
		"-fdebug-info-for-profiling",

		"-fno-strict-aliasing",

		"-Werror=date-time",
		"-Werror=pragma-pack",
		"-Werror=pragma-pack-suspicious-include",
		"-Werror=string-plus-int",
		"-Werror=unreachable-code-loop-increment",
	}

	commonGlobalConlyflags = []string{}

	deviceGlobalCflags = []string{
		"-fdiagnostics-color",

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
		"-Werror=format-security",
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
		// TODO: Eventually we should link against a libunwind.a with hidden symbols, and then these
		// --exclude-libs arguments can be removed.
		"-Wl,--exclude-libs,libgcc.a",
		"-Wl,--exclude-libs,libgcc_stripped.a",
		"-Wl,--exclude-libs,libunwind_llvm.a",
		"-Wl,--exclude-libs,libunwind.a",
		"-Wl,--icf=safe",
	}

	deviceGlobalLldflags = append(ClangFilterUnknownLldflags(deviceGlobalLdflags),
		[]string{
			"-fuse-ld=lld",
		}...)

	hostGlobalCflags = []string{}

	hostGlobalCppflags = []string{}

	hostGlobalLdflags = []string{}

	hostGlobalLldflags = []string{"-fuse-ld=lld"}

	commonGlobalCppflags = []string{
		"-Wsign-promo",
	}

	noOverrideGlobalCflags = []string{
		"-Werror=bool-operation",
		"-Werror=implicit-int-float-conversion",
		"-Werror=int-in-bool-context",
		"-Werror=int-to-pointer-cast",
		"-Werror=pointer-to-int-cast",
		"-Werror=string-compare",
		"-Werror=xor-used-as-pow",
		// http://b/161386391 for -Wno-void-pointer-to-enum-cast
		"-Wno-void-pointer-to-enum-cast",
		// http://b/161386391 for -Wno-void-pointer-to-int-cast
		"-Wno-void-pointer-to-int-cast",
		// http://b/161386391 for -Wno-pointer-to-int-cast
		"-Wno-pointer-to-int-cast",
		"-Werror=fortify-source",
	}

	IllegalFlags = []string{
		"-w",
	}

	CStdVersion               = "gnu99"
	CppStdVersion             = "gnu++17"
	ExperimentalCStdVersion   = "gnu11"
	ExperimentalCppStdVersion = "gnu++2a"

	// prebuilts/clang default settings.
	ClangDefaultBase         = "prebuilts/clang/host"
	ClangDefaultVersion      = "clang-r416183b"
	ClangDefaultShortVersion = "12.0.5"

	// Directories with warnings from Android.bp files.
	WarningAllowedProjects = []string{
		"device/",
		"vendor/",
	}

	// Directories with warnings from Android.mk files.
	WarningAllowedOldProjects = []string{}

	// Exported for bp2build for generating a cc_toolchain.
	exportedVars = map[string]map[string]string{}
)

var pctx = android.NewPackageContext("android/soong/cc/config")

func init() {
	if android.BuildOs == android.Linux {
		commonGlobalCflags = append(commonGlobalCflags, "-fdebug-prefix-map=/proc/self/cwd=")
	}

	staticVariable("CommonGlobalConlyflags", strings.Join(commonGlobalConlyflags, " "))
	staticVariable("DeviceGlobalCppflags", strings.Join(deviceGlobalCppflags, " "))
	staticVariable("DeviceGlobalLdflags", strings.Join(deviceGlobalLdflags, " "))
	staticVariable("DeviceGlobalLldflags", strings.Join(deviceGlobalLldflags, " "))
	staticVariable("HostGlobalCppflags", strings.Join(hostGlobalCppflags, " "))
	staticVariable("HostGlobalLdflags", strings.Join(hostGlobalLdflags, " "))
	staticVariable("HostGlobalLldflags", strings.Join(hostGlobalLldflags, " "))

	// CommonClangGlobalCflags
	commonClangGlobalCflags := ClangFilterUnknownCflags(commonGlobalCflags)
	commonClangGlobalCflags = append(commonClangGlobalCflags, "${ClangExtraCflags}")

	autoZeroInitializeFlags := strings.Join(
		append(
			commonClangGlobalCflags,
			"-ftrivial-auto-var-init=zero -enable-trivial-auto-var-init-zero-knowing-it-will-be-removed-from-clang"), " ")
	setBazelCcEnvDependentVariable(exportedVars, "CommonClangGlobalCflags", autoZeroInitializeFlags, "AUTO_ZERO_INITIALIZE")

	autoPatternInitializeFlags := strings.Join(append(commonClangGlobalCflags, "-ftrivial-auto-var-init=pattern"), " ")
	setBazelCcEnvDependentVariable(exportedVars, "CommonClangGlobalCflags", autoPatternInitializeFlags, "AUTO_PATTERN_INITIALIZE")

	autoUninitializeFlags := strings.Join(append(commonClangGlobalCflags, "-ftrivial-auto-var-init=uninitialized"), " ")
	setBazelCcEnvDependentVariable(exportedVars, "CommonClangGlobalCflags", autoUninitializeFlags, "AUTO_UNINITIALIZE")

	zeroInitializationFlags := strings.Join(
		append(
			commonClangGlobalCflags,
			"-ftrivial-auto-var-init=zero -enable-trivial-auto-var-init-zero-knowing-it-will-be-removed-from-clang"), " ")
	setBazelCcVariable(exportedVars, "CommonClangGlobalCflags", zeroInitializationFlags)

	pctx.VariableFunc("CommonClangGlobalCflags", func(ctx android.PackageVarContext) string {
		// http://b/131390872
		// Automatically initialize any uninitialized stack variables.
		// Prefer zero-init if multiple options are set.
		if ctx.Config().IsEnvTrue("AUTO_ZERO_INITIALIZE") {
			return autoZeroInitializeFlags
		} else if ctx.Config().IsEnvTrue("AUTO_PATTERN_INITIALIZE") {
			return autoPatternInitializeFlags
		} else if ctx.Config().IsEnvTrue("AUTO_UNINITIALIZE") {
			return autoUninitializeFlags
		} else {
			// Default to zero initialization.
			return zeroInitializationFlags
		}
	})

	// DeviceClangGlobalCflags
	deviceClangGlobalCflagsFuschia := strings.Join(ClangFilterUnknownCflags(deviceGlobalCflags), " ")
	// Fuschia flags aren't being exported since it's not needed yet.
	deviceClangGlobalCflags := strings.Join(append(ClangFilterUnknownCflags(deviceGlobalCflags), "${ClangExtraTargetCflags}"), " ")
	setBazelCcVariable(exportedVars, "DeviceClangGlobalCflags", deviceClangGlobalCflags)
	pctx.VariableFunc("DeviceClangGlobalCflags", func(ctx android.PackageVarContext) string {
		if ctx.Config().Fuchsia() {
			return deviceClangGlobalCflagsFuschia
		} else {
			return deviceClangGlobalCflags
		}
	})

	staticVariable("HostClangGlobalCflags",
		strings.Join(ClangFilterUnknownCflags(hostGlobalCflags), " "))
	staticVariable("NoOverrideClangGlobalCflags",
		strings.Join(append(ClangFilterUnknownCflags(noOverrideGlobalCflags), "${ClangExtraNoOverrideCflags}"), " "))

	staticVariable("CommonClangGlobalCppflags",
		strings.Join(append(ClangFilterUnknownCflags(commonGlobalCppflags), "${ClangExtraCppflags}"), " "))

	staticVariable("ClangExternalCflags", "${ClangExtraExternalCflags}")

	// Everything in these lists is a crime against abstraction and dependency tracking.
	// Do not add anything to this list.
	pctx.PrefixedExistentPathsForSourcesVariable("CommonGlobalIncludes", "-I",
		[]string{
			"system/core/include",
			"system/logging/liblog/include",
			"system/media/audio/include",
			"hardware/libhardware/include",
			"hardware/libhardware_legacy/include",
			"hardware/ril/include",
			"frameworks/native/include",
			"frameworks/native/opengl/include",
			"frameworks/av/include",
		})

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

	pctx.VariableFunc("ClangShortVersion", func(ctx android.PackageVarContext) string {
		if override := ctx.Config().Getenv("LLVM_RELEASE_VERSION"); override != "" {
			return override
		}
		return ClangDefaultShortVersion
	})
	pctx.StaticVariable("ClangAsanLibDir", "${ClangBase}/linux-x86/${ClangVersion}/lib64/clang/${ClangShortVersion}/lib/linux")

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

	pctx.StaticVariableWithEnvOverride("RECXXPool", "RBE_CXX_POOL", remoteexec.DefaultPool)
	pctx.StaticVariableWithEnvOverride("RECXXLinksPool", "RBE_CXX_LINKS_POOL", remoteexec.DefaultPool)
	pctx.StaticVariableWithEnvOverride("REClangTidyPool", "RBE_CLANG_TIDY_POOL", remoteexec.DefaultPool)
	pctx.StaticVariableWithEnvOverride("RECXXLinksExecStrategy", "RBE_CXX_LINKS_EXEC_STRATEGY", remoteexec.LocalExecStrategy)
	pctx.StaticVariableWithEnvOverride("REClangTidyExecStrategy", "RBE_CLANG_TIDY_EXEC_STRATEGY", remoteexec.LocalExecStrategy)
	pctx.StaticVariableWithEnvOverride("REAbiDumperExecStrategy", "RBE_ABI_DUMPER_EXEC_STRATEGY", remoteexec.LocalExecStrategy)
	pctx.StaticVariableWithEnvOverride("REAbiLinkerExecStrategy", "RBE_ABI_LINKER_EXEC_STRATEGY", remoteexec.LocalExecStrategy)
}

var HostPrebuiltTag = pctx.VariableConfigMethod("HostPrebuiltTag", android.Config.PrebuiltOS)

func envOverrideFunc(envVar, defaultVal string) func(ctx android.PackageVarContext) string {
	return func(ctx android.PackageVarContext) string {
		if override := ctx.Config().Getenv(envVar); override != "" {
			return override
		}
		return defaultVal
	}
}

// Convenience function to declare a static variable and export it to Bazel's cc_toolchain.
func staticVariable(name, value string) {
	pctx.StaticVariable(name, value)
	setBazelCcVariable(exportedVars, name, value)
}

func setBazelCcVariable(store map[string]map[string]string, key, value string) {
	setBazelCcEnvDependentVariable(store, key, value, "__default__")
}

func setBazelCcEnvDependentVariable(store map[string]map[string]string, key, value, envVar string) {
	if store[key] == nil {
		store[key] = map[string]string{}
	}
	store[key][envVar] = value
}

// BazelCcToolchainVars generates a list of string/dict variables containing
// configuration for Bazel's cc_toolchain configuration.
func BazelCcToolchainVars() string {
	// Replace with a regex?
	findOnlyVar := func(s string) string {
		i := strings.Index(s, "{")
		if i >= 0 {
			j := strings.Index(s, "}")
			if j >= 0 {
				return s[i+1 : j]

			}
		}
		return ""
	}
	ret := "# GENERATED FOR BAZEL FROM SOONG. DO NOT EDIT.\n\n"
	for _, k := range android.SortedStringKeys(exportedVars) {
		vs := exportedVars[k]
		if len(vs) > 1 {
			ret += fmt.Sprintf("_%s = {}\n", k)
		}
		for envVar, v := range vs {
			// Assume that there is only _at most 1_ interpolated variable.
			if strings.Contains(v, "${") {
				interpolatedVar := findOnlyVar(v)
				v = strings.Replace(
					v,
					fmt.Sprintf("${%s}", interpolatedVar),
					exportedVars[interpolatedVar]["__default__"], 1)
			}
			list := "["
			for _, flag := range strings.Split(v, " ") {
				if flag != "" {
					list += "\n    \"" + flag + "\","
				}
			}
			list += "\n]"
			if len(vs) == 1 {
				ret += fmt.Sprintf("_%s = %s\n", k, list)
			} else {
				ret += fmt.Sprintf("_%s[\"%s\"] = %s\n", k, envVar, list)
			}
			ret += "\n"
		}
	}

	ret += "constants = struct(\n"
	for _, k := range android.SortedStringKeys(exportedVars) {
		ret += fmt.Sprintf("    %s = _%s,\n", k, k)
	}
	ret += ")"
	return ret
}
