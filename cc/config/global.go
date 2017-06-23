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

		// COMMON_RELEASE_CFLAGS
		"-DNDEBUG",
		"-UDEBUG",
	}

	commonGlobalConlyflags = []string{}

	deviceGlobalCflags = []string{
		"-fdiagnostics-color",

		// TARGET_ERROR_FLAGS
		"-Werror=return-type",
		"-Werror=non-virtual-dtor",
		"-Werror=address",
		"-Werror=sequence-point",
		"-Werror=date-time",
	}

	hostGlobalCflags = []string{}

	commonGlobalCppflags = []string{
		"-Wsign-promo",
	}

	noOverrideGlobalCflags = []string{
		"-Werror=int-to-pointer-cast",
		"-Werror=pointer-to-int-cast",
	}

	IllegalFlags = []string{
		"-w",
	}

	CStdVersion               = "gnu99"
	CppStdVersion             = "gnu++14"
	GccCppStdVersion          = "gnu++11"
	ExperimentalCStdVersion   = "gnu11"
	ExperimentalCppStdVersion = "gnu++1z"

	NdkMaxPrebuiltVersionInt = 24

	// prebuilts/clang default settings.
	ClangDefaultBase         = "prebuilts/clang/host"
	ClangDefaultVersion      = "clang-4053586"
	ClangDefaultShortVersion = "5.0"
)

var pctx = android.NewPackageContext("android/soong/cc/config")

func init() {
	if android.BuildOs == android.Linux {
		commonGlobalCflags = append(commonGlobalCflags, "-fdebug-prefix-map=/proc/self/cwd=")
	}

	pctx.StaticVariable("CommonGlobalCflags", strings.Join(commonGlobalCflags, " "))
	pctx.StaticVariable("CommonGlobalConlyflags", strings.Join(commonGlobalConlyflags, " "))
	pctx.StaticVariable("DeviceGlobalCflags", strings.Join(deviceGlobalCflags, " "))
	pctx.StaticVariable("HostGlobalCflags", strings.Join(hostGlobalCflags, " "))
	pctx.StaticVariable("NoOverrideGlobalCflags", strings.Join(noOverrideGlobalCflags, " "))

	pctx.StaticVariable("CommonGlobalCppflags", strings.Join(commonGlobalCppflags, " "))

	pctx.StaticVariable("CommonClangGlobalCflags",
		strings.Join(append(ClangFilterUnknownCflags(commonGlobalCflags), "${ClangExtraCflags}"), " "))
	pctx.StaticVariable("DeviceClangGlobalCflags",
		strings.Join(append(ClangFilterUnknownCflags(deviceGlobalCflags), "${ClangExtraTargetCflags}"), " "))
	pctx.StaticVariable("HostClangGlobalCflags",
		strings.Join(ClangFilterUnknownCflags(hostGlobalCflags), " "))
	pctx.StaticVariable("NoOverrideClangGlobalCflags",
		strings.Join(append(ClangFilterUnknownCflags(noOverrideGlobalCflags), "${ClangExtraNoOverrideCflags}"), " "))

	pctx.StaticVariable("CommonClangGlobalCppflags",
		strings.Join(append(ClangFilterUnknownCflags(commonGlobalCppflags), "${ClangExtraCppflags}"), " "))

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
		[]string{"libnativehelper/include/nativehelper"})

	pctx.SourcePathVariable("ClangDefaultBase", ClangDefaultBase)
	pctx.VariableFunc("ClangBase", func(config interface{}) (string, error) {
		if override := config.(android.Config).Getenv("LLVM_PREBUILTS_BASE"); override != "" {
			return override, nil
		}
		return "${ClangDefaultBase}", nil
	})
	pctx.VariableFunc("ClangVersion", func(config interface{}) (string, error) {
		if override := config.(android.Config).Getenv("LLVM_PREBUILTS_VERSION"); override != "" {
			return override, nil
		}
		return ClangDefaultVersion, nil
	})
	pctx.StaticVariable("ClangPath", "${ClangBase}/${HostPrebuiltTag}/${ClangVersion}")
	pctx.StaticVariable("ClangBin", "${ClangPath}/bin")

	pctx.VariableFunc("ClangShortVersion", func(config interface{}) (string, error) {
		if override := config.(android.Config).Getenv("LLVM_RELEASE_VERSION"); override != "" {
			return override, nil
		}
		return ClangDefaultShortVersion, nil
	})
	pctx.StaticVariable("ClangAsanLibDir", "${ClangPath}/lib64/clang/${ClangShortVersion}/lib/linux")

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

	pctx.VariableFunc("CcWrapper", func(config interface{}) (string, error) {
		if override := config.(android.Config).Getenv("CC_WRAPPER"); override != "" {
			return override + " ", nil
		}
		return "", nil
	})
}

var HostPrebuiltTag = pctx.VariableConfigMethod("HostPrebuiltTag", android.Config.PrebuiltOS)

func bionicHeaders(bionicArch, kernelArch string) string {
	return strings.Join([]string{
		"-isystem bionic/libc/arch-" + bionicArch + "/include",
		"-isystem bionic/libc/include",
		"-isystem bionic/libc/kernel/uapi",
		"-isystem bionic/libc/kernel/uapi/asm-" + kernelArch,
		"-isystem bionic/libc/kernel/android/scsi",
		"-isystem bionic/libc/kernel/android/uapi",
	}, " ")
}

//This list is captured from VNDK Tag v3.7
func VndkLibraries() []string {
	return []string{
		"android.frameworks.schedulerservice@1.0",
		"android.frameworks.sensorservice@1.0",
		"android.frameworks.vr.composer@1.0",
		"android.frameworks.vr.composer@1.0",
		"android.hardware.audio.common@2.0-util",
		"android.hardware.audio.common@2.0",
		"android.hardware.audio.effect@2.0",
		"android.hardware.audio@2.0",
		"android.hardware.biometrics.fingerprint@2.1",
		"android.hardware.bluetooth@1.0",
		"android.hardware.boot@1.0",
		"android.hardware.broadcastradio@1.0",
		"android.hardware.broadcastradio@1.1",
		"android.hardware.camera.common@1.0",
		"android.hardware.camera.device@1.0",
		"android.hardware.camera.device@3.2",
		"android.hardware.camera.provider@2.4",
		"android.hardware.configstore-utils",
		"android.hardware.configstore@1.0",
		"android.hardware.contexthub@1.0",
		"android.hardware.drm@1.0",
		"android.hardware.dumpstate@1.0",
		"android.hardware.gatekeeper@1.0",
		"android.hardware.gnss@1.0",
		"android.hardware.graphics.allocator@2.0",
		"android.hardware.graphics.bufferqueue@1.0",
		"android.hardware.graphics.common@1.0",
		"android.hardware.graphics.composer@2.1",
		"android.hardware.graphics.mapper@2.0",
		"android.hardware.health@1.0",
		"android.hardware.ir@1.0",
		"android.hardware.keymaster@3.0",
		"android.hardware.light@2.0",
		"android.hardware.media.omx@1.0",
		"android.hardware.media@1.0",
		"android.hardware.memtrack@1.0",
		"android.hardware.nfc@1.0",
		"android.hardware.power@1.0",
		"android.hardware.radio.deprecated@1.0",
		"android.hardware.radio@1.0",
		"android.hardware.renderscript@1.0",
		"android.hardware.sensors@1.0",
		"android.hardware.soundtrigger@2.0",
		"android.hardware.thermal@1.0",
		"android.hardware.tv.cec@1.0",
		"android.hardware.tv.input@1.0",
		"android.hardware.usb@1.0",
		"android.hardware.vibrator@1.0",
		"android.hardware.vr@1.0",
		"android.hardware.wifi.supplicant@1.0",
		"android.hardware.wifi@1.0",
		"android.hidl.allocator@1.0",
		"android.hidl.base@1.0",
		"android.hidl.manager@1.0",
		"android.hidl.memory@1.0",
		"android.hidl.token@1.0",
		"android.system.wifi.keystore@1.0",
		"libaudioroute",
		"libaudioutils",
		"libbacktrace",
		"libbase",
		"libbcinfo",
		"libbinder",
		"libblas",
		"libc++",
		"libcamera_metadata",
		"libcap",
		"libcrypto_utils",
		"libcrypto",
		"libcups",
		"libcurl",
		"libcutils",
		"libdiskconfig",
		"libdumpstateutil",
		"libevent",
		"libexif",
		"libexpat",
		"libfmq",
		"libgatekeeper",
		"libgui",
		"libhardware_legacy",
		"libhardware",
		"libhidlbase",
		"libhidlmemory",
		"libhidltransport",
		"libhwbinder",
		"libicui18n",
		"libicuuc",
		"libion",
		"libjpeg",
		"libkeymaster_messages",
		"libkeymaster1",
		"libldacBT_abr",
		"libldacBT_enc",
		"liblz4",
		"libmdnssd",
		"libmemtrack",
		"libmetricslogger",
		"libminijail",
		"libnbaio",
		"libnetutils",
		"libnl",
		"libopus",
		"libpagemap",
		"libpcap",
		"libpcre2",
		"libpcrecpp",
		"libpdfium",
		"libpiex",
		"libpng",
		"libpower",
		"libprocessgroup",
		"libprocinfo",
		"libprotobuf-cpp-full",
		"libprotobuf-cpp-lite",
		"libradio_metadata",
		"libsoftkeymasterdevice",
		"libsonic",
		"libsonivox",
		"libspeexresampler",
		"libsqlite",
		"libssl",
		"libsuspend",
		"libsysutils",
		"libtinyalsa",
		"libtinyxml2",
		"libui",
		"libusbhost",
		"libvixl-arm",
		"libvixl-arm64",
		"libvorbisidec",
		"libwebrtc_audio_preprocessing",
		"libxml2",
		"libziparchive",
	}
}

// This needs to be kept up to date with the list in system/core/rootdir/etc/ld.config.txt:
// [vendor]
// namespace.default.link.system.shared_libs
func LLndkLibraries() []string {
	return []string{"libc", "libm", "libdl", "liblog", "ld-android"}
}

func replaceFirst(slice []string, from, to string) {
	if slice[0] != from {
		panic(fmt.Errorf("Expected %q, found %q", from, to))
	}
	slice[0] = to
}
