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
	"android/soong/android"
)

const (
	linuxArm64GccVersion   = "4.8.3"
	linuxArm64GlibcVersion = "2.17"
)

func init() {
	pctx.StaticVariable("LinuxArm64GccVersion", linuxArm64GccVersion)
	pctx.SourcePathVariable("LinuxArm64GccRoot",
		"prebuilts/gcc/${HostPrebuiltTag}/host/aarch64-linux-glibc${LinuxGlibcVersion}-${ShortLinuxGccVersion}")
	pctx.StaticVariable("LinuxArm64GccTriple", "aarch64-linux")
}

type toolchainLinuxArm64 struct {
	toolchainArm64
	toolchainLinux
}

func (t *toolchainLinuxArm64) Bionic() bool {
	return false
}

func (t *toolchainLinuxArm64) GccRoot() string {
	return "${config.LinuxArm64GccRoot}"
}

func (t *toolchainLinuxArm64) GccTriple() string {
	return "${config.LinuxArm64GccTriple}"
}

func (t *toolchainLinuxArm64) GccVersion() string {
	return linuxArm64GccVersion
}

func (t *toolchainLinuxArm64) IncludeFlags() string {
	return ""
}

var toolchainLinuxArm64Singleton Toolchain = &toolchainLinuxArm64{}

func linuxArm64ToolchainFactory(arch android.Arch) Toolchain {
	return toolchainLinuxArm64Singleton
}

func init() {
	registerToolchainFactory(android.Linux, android.Arm64, linuxArm64ToolchainFactory)
}
