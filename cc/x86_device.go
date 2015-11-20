package cc

import (
	"strings"

	"android/soong/common"
)

var (
	x86Cflags = []string{
		"-fno-exceptions", // from build/core/combo/select.mk
		"-Wno-multichar",  // from build/core/combo/select.mk
		"-O2",
		"-Wa,--noexecstack",
		"-Werror=format-security",
		"-D_FORTIFY_SOURCE=2",
		"-Wstrict-aliasing=2",
		"-ffunction-sections",
		"-finline-functions",
		"-finline-limit=300",
		"-fno-short-enums",
		"-fstrict-aliasing",
		"-funswitch-loops",
		"-funwind-tables",
		"-fstack-protector",
		"-m32",
		"-no-canonical-prefixes",
		"-fno-canonical-system-headers",

		// TARGET_RELEASE_CFLAGS from build/core/combo/select.mk
		"-O2",
		"-g",
		"-fno-strict-aliasing",
	}

	x86Cppflags = []string{}

	x86Ldflags = []string{
		"-m32",
		"-Wl,-z,noexecstack",
		"-Wl,-z,relro",
		"-Wl,-z,now",
		"-Wl,--build-id=md5",
		"-Wl,--warn-shared-textrel",
		"-Wl,--fatal-warnings",
		"-Wl,--gc-sections",
		"-Wl,--hash-style=gnu",
	}

	x86ArchVariantCflags = map[string][]string{
		"": []string{
			"-march=prescott",
		},
		"atom": []string{
			"-march=atom",
			"-mfpmath=sse",
		},
		"haswell": []string{
			"-march=core-avx2",
			"-mfpmath=sse",
		},
		"ivybridge": []string{
			"-march=core-avx-i",
			"-mfpmath=sse",
		},
		"sandybridge": []string{
			"-march=corei7-avx",
			"-mfpmath=sse",
		},
		"silvermont": []string{
			"-march=slm",
			"-mfpmath=sse",
		},
	}

	x86ArchFeatureCflags = map[string][]string{
		"ssse3":  []string{"-DUSE_SSSE3", "-mssse3"},
		"sse4":   []string{"-msse4"},
		"sse4_1": []string{"-msse4.1"},
		"sse4_2": []string{"-msse4.2"},
		"avx":    []string{"-mavx"},
		"aes_ni": []string{"-maes"},
	}
)

func init() {
	common.RegisterArchFeatures(common.X86, "atom",
		"ssse3",
		"movbe")
	common.RegisterArchFeatures(common.X86, "haswell",
		"ssse3",
		"sse4",
		"sse4_1",
		"sse4_2",
		"aes_ni",
		"avx",
		"popcnt",
		"movbe")
	common.RegisterArchFeatures(common.X86, "ivybridge",
		"ssse3",
		"sse4",
		"sse4_1",
		"sse4_2",
		"aes_ni",
		"avx",
		"popcnt")
	common.RegisterArchFeatures(common.X86, "sandybridge",
		"ssse3",
		"sse4",
		"sse4_1",
		"sse4_2",
		"aes_ni",
		"avx",
		"popcnt")
	common.RegisterArchFeatures(common.X86, "silvermont",
		"ssse3",
		"sse4",
		"sse4_1",
		"sse4_2",
		"aes_ni",
		"popcnt",
		"movbe")

	pctx.StaticVariable("x86GccVersion", "4.9")

	pctx.StaticVariable("x86GccRoot",
		"prebuilts/gcc/${HostPrebuiltTag}/x86/x86_64-linux-android-${armGccVersion}")

	pctx.StaticVariable("x86GccTriple", "x86_64-linux-android")

	pctx.StaticVariable("x86Cflags", strings.Join(x86Cflags, " "))
	pctx.StaticVariable("x86Ldflags", strings.Join(x86Ldflags, " "))
	pctx.StaticVariable("x86Cppflags", strings.Join(x86Cppflags, " "))
	pctx.StaticVariable("x86IncludeFlags", strings.Join([]string{
		"-isystem ${LibcRoot}/arch-x86/include",
		"-isystem ${LibcRoot}/include",
		"-isystem ${LibcRoot}/kernel/uapi",
		"-isystem ${LibcRoot}/kernel/uapi/asm-x86",
		"-isystem ${LibmRoot}/include",
		"-isystem ${LibmRoot}/include/i387",
	}, " "))

	// Clang cflags
	pctx.StaticVariable("x86ClangCflags", strings.Join(clangFilterUnknownCflags(x86Cflags), " "))
	pctx.StaticVariable("x86ClangLdflags", strings.Join(clangFilterUnknownCflags(x86Ldflags), " "))
	pctx.StaticVariable("x86ClangCppflags", strings.Join(clangFilterUnknownCflags(x86Cppflags), " "))

	// Extended cflags

	// Architecture variant cflags
	for variant, cflags := range x86ArchVariantCflags {
		pctx.StaticVariable("x86"+variant+"VariantCflags", strings.Join(cflags, " "))
		pctx.StaticVariable("x86"+variant+"VariantClangCflags",
			strings.Join(clangFilterUnknownCflags(cflags), " "))
	}
}

type toolchainX86 struct {
	toolchain32Bit
	cflags, ldflags, clangCflags string
}

func (t *toolchainX86) Name() string {
	return "x86"
}

func (t *toolchainX86) GccRoot() string {
	return "${x86GccRoot}"
}

func (t *toolchainX86) GccTriple() string {
	return "${x86GccTriple}"
}

func (t *toolchainX86) GccVersion() string {
	return "${x86GccVersion}"
}

func (t *toolchainX86) Cflags() string {
	return t.cflags
}

func (t *toolchainX86) Cppflags() string {
	return "${x86Cppflags}"
}

func (t *toolchainX86) Ldflags() string {
	return t.ldflags
}

func (t *toolchainX86) IncludeFlags() string {
	return "${x86IncludeFlags}"
}

func (t *toolchainX86) ClangTriple() string {
	return "${x86GccTriple}"
}

func (t *toolchainX86) ClangCflags() string {
	return t.clangCflags
}

func (t *toolchainX86) ClangCppflags() string {
	return "${x86ClangCppflags}"
}

func (t *toolchainX86) ClangLdflags() string {
	return t.ldflags
}

func x86ToolchainFactory(arch common.Arch) Toolchain {
	cflags := []string{
		"${x86Cflags}",
		"${x86" + arch.ArchVariant + "VariantCflags}",
	}

	clangCflags := []string{
		"${x86ClangCflags}",
		"${x86" + arch.ArchVariant + "VariantClangCflags}",
	}

	for _, feature := range arch.ArchFeatures {
		cflags = append(cflags, x86ArchFeatureCflags[feature]...)
		clangCflags = append(clangCflags, x86ArchFeatureCflags[feature]...)
	}

	return &toolchainX86{
		cflags: strings.Join(cflags, " "),
		ldflags: strings.Join([]string{
			"${x86Ldflags}",
		}, " "),
		clangCflags: strings.Join(clangCflags, " "),
	}
}

func init() {
	registerToolchainFactory(common.Device, common.X86, x86ToolchainFactory)
}
