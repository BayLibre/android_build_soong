package arm

import (
	"strings"

	"blueprint"

	_ "android/soong/cc/common"

	"android/soong/common"
)

type Config interface {
	SrcDir() string
	PrebuiltOS() string
	ArchVariant(common.Arch) string
	CpuVariant(common.Arch) string
}

func init() {
	pctx.Import("android/soong/cc/common")
}

var (
	pctx = blueprint.NewPackageContext("android/soong/cc/target_arm")

	gccVersion = pctx.StaticVariable("gccVersion", "4.9")

	toolchainRoot = pctx.StaticVariable("toolchainRoot",
		"${common.SrcDir}/prebuilts/gcc/${common.HostPrebuiltTag}/arm/arm-linux-androideabi-${gccVersion}")

	ToolPrefix = pctx.StaticVariable("ToolPrefix", "${toolchainRoot}/bin/arm-linux-androideabi-")

	Cflags = pctx.StaticVariable("Cflags", strings.Join([]string{
		"-fno-exceptions", // from build/core/combo/select.mk
		"-Wno-multichar",  // from build/core/combo/select.mk
		"-fno-strict-aliasing",
		"-fstack-protector",
		"-ffunction-sections",
		"-fdata-sections",
		"-funwind-tables",
		"-fstack-protector",
		"-Wa,--noexecstack",
		"-Werror=format-security",
		"-D_FORTIFY_SOURCE=2",
		"-fno-short-enums",
		"-no-canonical-prefixes",
		"-fno-canonical-system-headers",
		"-include ${common.SrcDir}/build/core/combo/include/arch/linux-arm/AndroidConfig.h",

		"-fno-builtin-sin",
		"-fno-strict-volatile-bitfields",

		// TARGET_RELEASE_CFLAGS
		"-DNDEBUG",
		"-g",
		"-Wstrict-aliasing=2",
		"-fgcse-after-reload",
		"-frerun-cse-after-loop",
		"-frename-registers",
	}, " "))

	Ldflags = pctx.StaticVariable("Ldflags", strings.Join([]string{
		"-Wl,-z,noexecstack",
		"-Wl,-z,relro",
		"-Wl,-z,now",
		"-Wl,--build-id=md5",
		"-Wl,--warn-shared-textrel",
		"-Wl,--fatal-warnings",
		"-Wl,-icf=safe",
		"-Wl,--hash-style=gnu",

		// Disable transitive dependency library symbol resolving.
		"-Wl,--allow-shlib-undefined",
	}, " "))

	CppFlags = pctx.StaticVariable("Cppflags", strings.Join([]string{
		"-fvisibility-inlines-hidden",
	}, " "))

	IncludeFlags = pctx.StaticVariable("IncludeFlags", strings.Join([]string{
		"-isystem ${common.LibcRoot}/arch-arm/include",
		"-isystem ${common.LibcRoot}/include",
		"-isystem ${common.LibcRoot}/kernel/uapi",
		"-isystem ${common.LibcRoot}/kernel/uapi/asm-arm",
		"-isystem ${common.LibmRoot}/include",
		"-isystem ${common.LibmRoot}/include/arm",
	}, " "))

	CrtBeginSo = pctx.StaticVariable("CrtBeginSo", "target_arm/crtbegin_so.o")
	CrtEndSo   = pctx.StaticVariable("CrtEndSo", "target_arm/crtend_so.o")

	// Extended cflags

	// ARM mode vs. Thumb mode
	ArmCflags = pctx.StaticVariable("ArmCflags", strings.Join([]string{
		"-O2",
		"-fomit-frame-pointer",
		"-fstrict-aliasing",
		"-funswitch-loops",
	}, " "))

	ThumbClfags = pctx.StaticVariable("ThumbCflags", strings.Join([]string{
		"-mthumb",
		"-Os",
		"-fomit-frame-pointer",
		"-fno-strict-aliasing",
	}, " "))

	// armv5te variant
	Armv5TECflags = pctx.StaticVariable("Armv5TECflags", strings.Join([]string{
		"-march=armv5te",
		"-mtune=xscale",
		"-D__ARM_ARCH_5__",
		"-D__ARM_ARCH_5T__",
		"-D__ARM_ARCH_5E__",
		"-D__ARM_ARCH_5TE__",
	}, " "))

	Armv7ACflags = pctx.StaticVariable("Armv7ACflags", strings.Join([]string{
		"-march=armv7-a",
		"-mfloat-abi=softfp",
		"-mfpu=vfpv3-d16",
	}, " "))

	Armv7ALdflags = pctx.StaticVariable("Armv7ALdflags", strings.Join([]string{
		"-Wl,--fix-cortex-a8",
	}, " "))

	Armv7ANeonCflags = pctx.StaticVariable("Armv7ANeonCflags", strings.Join([]string{
		"-mfloat-abi=softfp",
		"-mfpu=neon",
	}, " "))

	ArchVariantCflags = map[string]string{
		"armv5te":      "${target_arm.Armv5TECflags}",
		"armv7-a":      "${target_arm.Armv7ACflags}",
		"armv7-a-neon": "${target_arm.Armv7ANeonCflags}",
	}

	ArchVariantLdflags = map[string]string{
		"armv7-a":      "${target_arm.Armv7ALdflags}",
		"armv7-a-neon": "${target_arm.Armv7ALdflags}",
	}

	CortexA7Cflags = pctx.StaticVariable("CortexA7Cflags", strings.Join([]string{
		"-mcpu=cortex-a7",
	}, " "))

	CortexA8Cflags = pctx.StaticVariable("CortexA8Cflags", strings.Join([]string{
		"-mcpu=cortex-a8",
	}, " "))

	CortexA15Cflags = pctx.StaticVariable("CortexA15Cflags", strings.Join([]string{
		"-mcpu=cortex-a15",
		// Fake an ARM compiler flag as these processors support LPAE which GCC/clang
		// don't advertise.
		"-D__ARM_FEATURE_LPAE=1",
	}, " "))

	CpuVariantCflags = map[string]string{
		"":           "${target_arm.DefaultCpuVariantCflags}",
		"cortex-a7":  "${target_arm.CortexA7Cflags}",
		"cortex-a8":  "${target_arm.CortexA8Cflags}",
		"cortex-a15": "${target_arm.CortexA15Cflags}",
		"krait":      "${target_arm.CortexA15Cflags}",
		"denver":     "${target_arm.CortexA15Cflags}",
	}
)
