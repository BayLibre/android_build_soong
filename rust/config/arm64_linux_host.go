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
	registerToolchainFactory(android.Linux, android.Arm64, linuxX8664ToolchainFactory)
}

type toolchainLinuxArm64 struct {
	toolchain64Bit
	toolchainLinux
}

func (toolchainLinuxArm64) Supported() bool {
	return true
}

func (toolchainLinuxArm64) Bionic() bool {
	return false
}

func linuxArm64ToolchainFactory(arch android.Arch) Toolchain {
	return toolchainLinuxArm64Singleton
}

var toolchainLinuxArm64Singleton Toolchain = &toolchainLinuxArm64{}
