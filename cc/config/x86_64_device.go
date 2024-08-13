// Copyright 2015 Google Inc. All rights reserved.
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
)

var (
	x86_64Cflags = []string{
		// Help catch common 32/64-bit errors.
		"-Werror=implicit-function-declaration",
	}

	x86_64Cppflags = []string{}

	x86_64Ldflags = []string{
		"-Wl,-z,separate-loadable-segments",
	}

	X86_64Lldflags = x86_64Ldflags

	x86_64ArchVariantCflags = map[string][]string{
		"": []string{
			"-march=x86-64",
		},

		"broadwell": []string{
			"-march=broadwell",
		},
		"goldmont": []string{
			"-march=goldmont",
		},
		"goldmont-plus": []string{
			"-march=goldmont-plus",
		},
		"goldmont-without-sha-xsaves": []string{
			"-march=goldmont",
			"-mno-sha",
			"-mno-xsaves",
		},
		"haswell": []string{
			"-march=core-avx2",
		},
		"ivybridge": []string{
			"-march=core-avx-i",
		},
		"sandybridge": []string{
			"-march=corei7",
		},
		"silvermont": []string{
			"-march=slm",
		},
		"skylake": []string{
			"-march=skylake",
		},
		"stoneyridge": []string{
			"-march=bdver4",
		},
		"tremont": []string{
			"-march=tremont",
		},
	}

	x86_64ArchFeatureCflags = map[string][]string{
		"ssse3":  []string{"-mssse3"},
		"sse4":   []string{"-msse4"},
		"sse4_1": []string{"-msse4.1"},
		"sse4_2": []string{"-msse4.2"},

		// Not all cases there is performance gain by enabling -mavx -mavx2
		// flags so these flags are not enabled by default.
		// if there is performance gain in individual library components,
		// the compiler flags can be set in corresponding bp files.
		// "avx":    []string{"-mavx"},
		// "avx2":   []string{"-mavx2"},
		// "avx512": []string{"-mavx512"}

		"popcnt": []string{"-mpopcnt"},
		"aes_ni": []string{"-maes"},
	}
)

func init() {
	pctx.StaticVariable("X86_64ToolchainCflags", "-m64")
	pctx.StaticVariable("X86_64ToolchainLdflags", "-m64")

	pctx.StaticVariable("X86_64Ldflags", strings.Join(x86_64Ldflags, " "))
	pctx.VariableFunc("X86_64Lldflags", func(ctx android.PackageVarContext) string {
		maxPageSizeFlag := "-Wl,-z,max-page-size=" + ctx.Config().MaxPageSizeSupported()
		flags := append(X86_64Lldflags, maxPageSizeFlag)
		return strings.Join(flags, " ")
	})

	// Clang cflags
	pctx.VariableFunc("X86_64Cflags", func(ctx android.PackageVarContext) string {
		flags := x86_64Cflags
		if ctx.Config().NoBionicPageSizeMacro() {
			flags = append(flags, "-D__BIONIC_NO_PAGE_SIZE_MACRO")
		} else {
			flags = append(flags, "-D__BIONIC_DEPRECATED_PAGE_SIZE_MACRO")
		}
		return strings.Join(flags, " ")
	})

	pctx.StaticVariable("X86_64Cppflags", strings.Join(x86_64Cppflags, " "))

	// Yasm flags
	pctx.StaticVariable("X86_64YasmFlags", "-f elf64 -m amd64")

	// Architecture variant cflags
	for variant, cflags := range x86_64ArchVariantCflags {
		pctx.StaticVariable("X86_64"+variant+"VariantCflags", strings.Join(cflags, " "))
	}
}

type toolchainX86_64 struct {
	toolchainBionic
	toolchain64Bit
	toolchainCflags string
}

func (t *toolchainX86_64) Name() string {
	return "x86_64"
}

func (t *toolchainX86_64) IncludeFlags() string {
	return ""
}

func (t *toolchainX86_64) ClangTriple() string {
	return "x86_64-linux-android"
}

func (t *toolchainX86_64) ToolchainLdflags() string {
	return "${config.X86_64ToolchainLdflags}"
}

func (t *toolchainX86_64) ToolchainCflags() string {
	return t.toolchainCflags
}

func (t *toolchainX86_64) Cflags() string {
	return "${config.X86_64Cflags}"
}

func (t *toolchainX86_64) Cppflags() string {
	return "${config.X86_64Cppflags}"
}

func (t *toolchainX86_64) Ldflags() string {
	return "${config.X86_64Ldflags}"
}

func (t *toolchainX86_64) Lldflags() string {
	return "${config.X86_64Lldflags}"
}

func (t *toolchainX86_64) YasmFlags() string {
	return "${config.X86_64YasmFlags}"
}

func (toolchainX86_64) LibclangRuntimeLibraryArch() string {
	return "x86_64"
}

func x86_64ToolchainFactory(arch android.Arch) Toolchain {
	// Error now rather than having a confusing Ninja error
	if _, ok := x86_64ArchVariantCflags[arch.ArchVariant]; !ok {
		panic(fmt.Sprintf("Unknown x86_64 architecture version: %q", arch.ArchVariant))
	}

	toolchainCflags := []string{
		"${config.X86_64ToolchainCflags}",
		"${config.X86_64" + arch.ArchVariant + "VariantCflags}",
	}

	for _, feature := range arch.ArchFeatures {
		toolchainCflags = append(toolchainCflags, x86_64ArchFeatureCflags[feature]...)
	}

	return &toolchainX86_64{
		toolchainCflags: strings.Join(toolchainCflags, " "),
	}
}

type toolchainTrustyX86_64 struct {
	toolchainBase
	toolchainNoCrt
	toolchain64Bit

	toolchainCflags string
}

func (t *toolchainTrustyX86_64) Name() string {
	return "x86_64-trusty"
}

func (t *toolchainTrustyX86_64) IncludeFlags() string {
	return ""
}

func (t *toolchainTrustyX86_64) ClangTriple() string {
	// from external/trusty/lk/arch/x86/toolchain.mk
	return "x86_64-linux-gnu"
}

func (t *toolchainTrustyX86_64) Cflags() string {
	sharedFlags := []string{
		"-O2",
		// from external/trusty/lk/engine.mk
		"-glldb",
		"-fdebug-macro",
		// HACK - need to port config.h to Soong
		"-Werror",
		"-Wall",
		"-Wsign-compare",
		"-Wno-multichar",
		"-Wno-unused-function",
		"-Wno-unused-label",
		"-fno-short-enums",
		"-fno-common",
		"-fno-omit-frame-pointer",
		"-Wstrict-prototypes",
		"-Wwrite-strings",
		"-Wimplicit-fallthrough",
		// VLAs can have subtle security bugs and assist exploits, so ban them.
		"-Wvla",
		// use linker garbage collection
		"-ffunction-sections",
		"-fdata-sections",
		// We are not Linux, and some libraries check this macro
		// and incorrectly target the wrong OS
		// TODO(b/224064243): remove this when we have a proper triple
		"-U__linux__",

		// from external/trusty/lk/make/module.mk
		// Initialize all automatic var to 0 if not initialized
		"-ftrivial-auto-var-init=zero",
	}

	// These flags are from config.h of the general_x86_64 target.
	// TODO: migrate config.h to Soong, rather than hard-coding here.
	globalDefines := []string{
		"-DLK=1",
		"-D__TRUSTY__=1",
		"-DHEAP_GROW_SIZE=8192",
		"-DMEMBASE=0X00200000",
		"-DMEMSIZE=0X0FE00000",
		"-DIS_64BIT=1",
		"-DARCH_X86_64=1",
		"-DMEMBASE=0X00200000",
		"-DKERNEL_BASE=0XFFFFFFFF80000000",
		"-DKERNEL_LOAD_OFFSET=0",
		"-DKERNEL_ASPACE_BASE=0XFFFFFF8000000000UL",
		"-DKERNEL_ASPACE_SIZE=0X0000008000000000UL",
		"-DUSER_ASPACE_BASE=0X0000000000001000UL",
		"-DUSER_ASPACE_SIZE=0X00007FFFFFFFE000UL",
		"-DSMP_MAX_CPUS=1",
		"-DX86_WITH_FPU=1",
		"-DPLATFORM_HAS_DYNAMIC_TIMER=1",
		"-DLK_LIBC_IMPLEMENTATION_IS_LK=1",
		"-DWITH_LIB_TRUSTY=1",
		"-DWITH_TRUSTY_IPC=1",
		"-DWITH_WAIT_ANY_SUPPORT=1",
		"-DWITH_SYSCALL_TABLE=1",
		"-DLK_HEAP_IMPLEMENTATION=MINIHEAP",
		"-DPROJECT_GENERIC_X86_64=1",
		"-DPROJECT=GENERIC_X86_64",
		"-DTARGET_GENERIC_X86_64=1",
		"-DTARGET=GENERIC_X86_64",
		"-DPLATFORM_GENERIC_X86_64=1",
		"-DPLATFORM=GENERIC_X86_64",
		"-DARCH_X86=1",
		"-DARCH=X86",
		"-DWITH_APP=1",
		"-DWITH_DEV=1",
		"-DWITH_DEV_INTERRUPT_X86_LAPIC=1",
		"-DWITH_DEV_TIMER_X86_GENERIC=1",
		"-DWITH_DEV_VIRTIO_VSOCK_RUST=1",
		"-DWITH_KERNEL=1",
		"-DWITH_KERNEL_VM=1",
		"-DWITH_LIB_BINARY_SEARCH_TREE=1",
		"-DWITH_LIB_CBUF=1",
		"-DWITH_LIB_DEBUG=1",
		"-DWITH_LIB_FIXED_POINT=1",
		"-DWITH_LIB_HEAP=1",
		"-DWITH_LIB_HEAP_MINIHEAP=1",
		"-DWITH_LIB_IO=1",
		"-DWITH_LIB_LIBC=1",
		"-DWITH_LIB_LIBC_RAND=1",
		"-DWITH_LIB_SYSCALL=1",
		"-DWITH_PLATFORM=1",
		"-DWITH_TARGET=1",
		"-DLK_DEBUGLEVEL=2",
		"-DLK_LOGLEVEL=2",
		"-DTLOG_LVL_DEFAULT=4",
		"-DIPC_MAX_HANDLES=64",
		"-DRELEASE_BUILD=1",
		"-DUSER_SCS_SUPPORTED=1",
	}

	return strings.Join(append(sharedFlags, globalDefines...), " ")
}

func (t *toolchainTrustyX86_64) Cppflags() string {
	sharedFlags := []string{
		// from external/trusty/lk/engine.mk
		"-fno-exceptions",
		"-fno-rtti",
		"-fno-threadsafe-statics",
		// c99 array designators are not part of C++, but they are convenient and help avoid errors.
		"-Wno-c99-designator",

		// from trusty/user/base/lib/libstdc++-trusty/rules.mk
		"-D_LIBCPP_BUILD_STATIC",
		"-D_LIBCPP_HAS_MUSL_LIBC",
		"-D_LIBCPP_HAS_QUICK_EXIT",
		"-D_LIBCPP_HAS_TIMESPEC_GET",
		"-D_LIBCPP_HAS_C11_FEATURES",
		"-D_LIBCPP_HAS_THREAD_API_PTHREAD",
	}
	return strings.Join(sharedFlags, " ")
}

func (toolchainTrustyX86_64) Ldflags() string {
	return ""
}

func (toolchainTrustyX86_64) Lldflags() string {
	return ""
}

func (toolchainTrustyX86_64) Asflags() string {
	sharedFlags := []string{
		// from external/trusty/lk/engine.mk
		"-DASSEMBLY",
	}
	return strings.Join(sharedFlags, " ")
}

func (t *toolchainTrustyX86_64) ToolchainCflags() string {
	return t.toolchainCflags
}

func (toolchainTrustyX86_64) AvailableLibraries() []string {
	return nil
}

func (toolchainTrustyX86_64) LibclangRuntimeLibraryArch() string {
	return "x86_64"
}

func (toolchainTrustyX86_64) ExecutableSuffix() string {
	return ""
}

func (toolchainTrustyX86_64) ShlibSuffix() string {
	return ""
}

func (toolchainTrustyX86_64) CrtBeginStaticBinary() []string {
	return []string{"trusty-libc-crtbegin"}
}

func x86_64TrustyToolchainFactory(arch android.Arch) Toolchain {
	// Error now rather than having a confusing Ninja error
	if _, ok := x86_64ArchVariantCflags[arch.ArchVariant]; !ok {
		panic(fmt.Sprintf("Unknown x86_64 architecture version: %q", arch.ArchVariant))
	}

	toolchainCflags := []string{
		"${config.X86_64ToolchainCflags}",
		"${config.X86_64" + arch.ArchVariant + "VariantCflags}",
	}

	for _, feature := range arch.ArchFeatures {
		toolchainCflags = append(toolchainCflags, x86_64ArchFeatureCflags[feature]...)
	}

	return &toolchainTrustyX86_64{
		toolchainCflags: strings.Join(toolchainCflags, " "),
	}
}

func init() {
	registerToolchainFactory(android.Android, android.X86_64, x86_64ToolchainFactory)
	registerToolchainFactory(android.Trusty, android.X86_64, x86_64TrustyToolchainFactory)
}
