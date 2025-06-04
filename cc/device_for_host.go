// Copyright (C) 2025 The Android Open Source Project
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

package cc

import (
	"android/soong/android"
)

//
// cc_device_for_host serves as an intermediate module, making the output of a device module
// available to host modules. This can only be used for the ART simulator.
//

type deviceForHost struct {
	Module

	properties deviceForHostProperties
}

type deviceForHostProperties struct {
	// List of modules whose contents will be visible to modules that depend on this module.
	Srcs []string
}

func init() {
	android.RegisterModuleType("cc_device_for_host", deviceForHostFactory)
}

func deviceForHostFactory() android.Module {
	module := &deviceForHost{}
	module.converter = module
	module.multilib = android.MultilibBoth
	module.AddProperties(&module.properties)

	// This module only supports host builds so it is visible to other host modules.
	module.hod = android.HostSupported
	return module.Init()
}

var deviceForHostDepTag = dependencyTag{name: "device_for_host"}

func (module *deviceForHost) getSrcs() []string {
	return module.properties.Srcs
}
