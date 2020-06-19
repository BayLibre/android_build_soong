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
	"android/soong/android"
)

func init() {
	registerToolchainFactory(android.Linux, android.Arm64, LinuxArm64ToolchainFactory)
	// Linux_cross-arm64 uses the same rust toolchain as the Android-arm64
	registerToolchainFactory(android.LinuxBionic, android.Arm64, Arm64ToolchainFactory)
}

type toolchainLinuxArm64 struct {
	toolchainArm64
	toolchainLinux
}

func (toolchainLinuxArm64) Supported() bool {
	return true
}

func (toolchainLinuxArm64) Bionic() bool {
	return false
}

func (t *toolchainLinuxArm64) Name() string {
	return "aarch64"
}

func (t *toolchainLinuxArm64) RustTriple() string {
	return "aarch64-unknown-linux-gnu"
}

func LinuxArm64ToolchainFactory(arch android.Arch) Toolchain {
	return &toolchainLinuxArm64{}
}
