package arm64

import (
	"strings"

	"blueprint"

	_ "android/soong/cc/common"
)

type Config interface {
	SrcDir() string
	PrebuiltOS() string
}

func init() {
	pctx.Import("android/soong/cc/common")
}

var (
	pctx = blueprint.NewPackageContext("android/soong/cc/target_arm64")

	gccVersion = pctx.StaticVariable("gccVersion", "4.9")

	toolchainRoot = pctx.StaticVariable("toolchainRoot",
		"${common.SrcDir}/prebuilts/gcc/${common.HostPrebuiltTag}/aarch64/aarch64-linux-android-${gccVersion}")

	ToolPrefix = pctx.StaticVariable("ToolPrefix", "${toolchainRoot}/bin/aarch64-linux-android-")

	Cflags = pctx.StaticVariable("Cflags", strings.Join([]string{
		"-fno-exceptions", // from build/core/combo/select.mk
		"-Wno-multichar",  // from build/core/combo/select.mk
		"-fno-strict-aliasing",
		"-fstack-protector",
		"-ffunction-sections",
		"-fdata-sections",
		"-funwind-tables",
		"-Wa,--noexecstack",
		"-Werror=format-security",
		"-D_FORTIFY_SOURCE=2",
		"-fno-short-enums",
		"-no-canonical-prefixes",
		"-fno-canonical-system-headers",
		"-include ${common.SrcDir}/build/core/combo/include/arch/linux-arm64/AndroidConfig.h",

		// Help catch common 32/64-bit errors.
		"-Werror=pointer-to-int-cast",
		"-Werror=int-to-pointer-cast",

		"-fno-strict-volatile-bitfields",

		// TARGET_RELEASE_CFLAGS
		"-DNDEBUG",
		"-O2 -g",
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
		"-Wl,-maarch64linux",
		"-Wl,--hash-style=gnu",

		// Disable transitive dependency library symbol resolving.
		"-Wl,--allow-shlib-undefined",
	}, " "))

	CppFlags = pctx.StaticVariable("Cppflags", strings.Join([]string{
		"-fvisibility-inlines-hidden",
	}, " "))

	IncludeFlags = pctx.StaticVariable("IncludeFlags", strings.Join([]string{
		"-isystem ${common.LibcRoot}/arch-arm64/include",
		"-isystem ${common.LibcRoot}/include",
		"-isystem ${common.LibcRoot}/kernel/uapi",
		"-isystem ${common.LibcRoot}/kernel/uapi/asm-arm64",
		"-isystem ${common.LibmRoot}/include",
		"-isystem ${common.LibmRoot}/include/arm64",
	}, " "))

	CrtBeginSo = pctx.StaticVariable("CrtBeginSo", "target_arm64/crtbegin_so.o")
	CrtEndSo   = pctx.StaticVariable("CrtEndSo", "target_arm64/crtend_so.o")
)
