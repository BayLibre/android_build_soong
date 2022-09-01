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
	"strings"

	"android/soong/android"
)

var (
	linuxBionicCflags = []string{
		"-Wa,--noexecstack",

		"-fPIC",

		"-U_FORTIFY_SOURCE",
		"-D_FORTIFY_SOURCE=2",
		"-fstack-protector-strong",

		// From x86_64_device
		"-ffunction-sections",
		"-fno-short-enums",
		"-funwind-tables",

		// Tell clang where the gcc toolchain is
		"--gcc-toolchain=${LinuxBionicGccRoot}",

		// This is normally in ClangExtraTargetCflags, but this is considered host
		"-nostdlibinc",
	}

	linuxBionicLdflags = []string{
		"-Wl,-z,noexecstack",
		"-Wl,-z,relro",
		"-Wl,-z,now",
		"-Wl,--build-id=md5",
		"-Wl,--fatal-warnings",
		"-Wl,--hash-style=gnu",
		"-Wl,--no-undefined-version",

		// Use the device gcc toolchain
		"--gcc-toolchain=${LinuxBionicGccRoot}",
	}

	// Embed the linker into host bionic binaries. This is needed to support host bionic,
	// as the linux kernel requires that the ELF interpreter referenced by PT_INTERP be
	// either an absolute path, or relative from CWD. To work around this, we extract
	// the load sections from the runtime linker ELF binary and embed them into each host
	// bionic binary, omitting the PT_INTERP declaration. The kernel will treat it as a static
	// binary, and then we use a special entry point to fix up the arguments passed by
	// the kernel before jumping to the embedded linker.
	linuxBionicCrtBeginSharedBinary = append(android.CopyOf(bionicCrtBeginSharedBinary),
		"host_bionic_linker_script")
)

func init() {
	pctx.StaticVariable("LinuxBionicCflags", strings.Join(linuxBionicCflags, " "))
	pctx.StaticVariable("LinuxBionicLdflags", strings.Join(linuxBionicLdflags, " "))
	pctx.StaticVariable("LinuxBionicLldflags", strings.Join(linuxBionicLdflags, " "))

	// Use the device gcc toolchain for now
	pctx.StaticVariable("LinuxBionicGccRoot", "${X86_64GccRoot}")
}

type toolchainLinuxBionic struct {
	toolchainBionic
}

func (t *toolchainLinuxBionic) GccRoot() string {
	return "${config.LinuxBionicGccRoot}"
}

func (t *toolchainLinuxBionic) GccVersion() string {
	return "4.9"
}

func (t *toolchainLinuxBionic) IncludeFlags() string {
	return ""
}

func (t *toolchainLinuxBionic) Cppflags() string {
	return ""
}

func (t *toolchainLinuxBionic) AvailableLibraries() []string {
	return nil
}

func (toolchainLinuxBionic) CrtBeginSharedBinary() []string {
	return linuxBionicCrtBeginSharedBinary
}

func (t *toolchainLinuxBionic) Cflags() string {
	return "${config.LinuxBionicCflags}"
}

func (t *toolchainLinuxBionic) Ldflags() string {
	return "${config.LinuxBionicLdflags}"
}

func (t *toolchainLinuxBionic) Lldflags() string {
	return "${config.LinuxBionicLldflags}"
}

type toolchainLinuxBionicX8664 struct {
	toolchain64Bit
	toolchainLinuxBionic
}

func (t *toolchainLinuxBionicX8664) Name() string {
	return "x86_64"
}

func (t *toolchainLinuxBionicX8664) GccTriple() string {
	return "x86_64-linux-android"
}

func (t *toolchainLinuxBionicX8664) ClangTriple() string {
	// TODO: we don't have a triple yet b/31393676
	return "x86_64-linux-android"
}

func (t *toolchainLinuxBionicX8664) ToolchainCflags() string {
	return "-m64 -march=x86-64" +
		// TODO: We're not really android, but we don't have a triple yet b/31393676
		" -U__ANDROID__"
}

func (t *toolchainLinuxBionicX8664) ToolchainLdflags() string {
	return "-m64"
}

func (toolchainLinuxBionicX8664) LibclangRuntimeLibraryArch() string {
	return "x86_64"
}

type toolchainLinuxBionicX86 struct {
	toolchain32Bit
	toolchainLinuxBionic
}

func (t *toolchainLinuxBionicX86) Name() string {
	return "x86"
}

func (t *toolchainLinuxBionicX86) GccTriple() string {
	return "i686-linux-android"
}

func (t *toolchainLinuxBionicX86) ClangTriple() string {
	// TODO: we don't have a triple yet b/31393676
	return "i686-linux-android"
}

func (t *toolchainLinuxBionicX86) ToolchainCflags() string {
	return "-m32 -march=i686" +
		// TODO: We're not really android, but we don't have a triple yet b/31393676
		" -U__ANDROID__"
}

func (t *toolchainLinuxBionicX86) ToolchainLdflags() string {
	return "-m32"
}

func (toolchainLinuxBionicX86) LibclangRuntimeLibraryArch() string {
	return "i686"
}

var toolchainLinuxBionicX86Singleton Toolchain = &toolchainLinuxBionicX86{}
var toolchainLinuxBionicX8664Singleton Toolchain = &toolchainLinuxBionicX8664{}

func linuxBionicX86ToolchainFactory(arch android.Arch) Toolchain {
	return toolchainLinuxBionicX86Singleton
}

func linuxBionicX8664ToolchainFactory(arch android.Arch) Toolchain {
	return toolchainLinuxBionicX8664Singleton
}

func init() {
	registerToolchainFactory(android.LinuxBionic, android.X86, linuxBionicX86ToolchainFactory)
	registerToolchainFactory(android.LinuxBionic, android.X86_64, linuxBionicX8664ToolchainFactory)
}
