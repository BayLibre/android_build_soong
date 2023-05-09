// Copyright 2023 Google Inc. All rights reserved.
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

package device_config

import (
	"android/soong/android"
	"github.com/google/blueprint"
)

// Properties for "device_config_override"
type DeviceConfigOverrideSetModule struct {
	android.ModuleBase
	android.DefaultableModuleBase

	properties struct {
		// device_config_override modules
		Overrides []string
	}
}

func DeviceConfigOverrideSetFactory() android.Module {
	module := &DeviceConfigOverrideSetModule{}

	android.InitAndroidModule(module)
	android.InitDefaultableModule(module)
	module.AddProperties(&module.properties)
	// TODO: bp2build
	//android.InitBazelModule(module)

	return module
}

// Dependency tag for overrides property
type overrideSetType struct {
	blueprint.BaseDependencyTag
}

var overrideSetTag = overrideSetType{}

// Provider published by device_config_override_set
type overrideSetProviderData struct {
	// The namespace of each of the
	// (map of namespace --> device_config_module)
	AvailableNamespaces map[string]android.Paths
}

var overrideSetProviderKey = blueprint.NewProvider(overrideSetProviderData{})

func (module *DeviceConfigOverrideSetModule) DepsMutator(ctx android.BottomUpMutatorContext) {
	deps := ctx.AddDependency(ctx.Module(), overrideSetTag, module.properties.Overrides...)
	for _, dep := range deps {
		_, ok := dep.(*DeviceConfigOverrideModule)
		if !ok {
			ctx.PropertyErrorf("overrides", "override must be a device_config_override module")
			return
		}
	}
}

func (module *DeviceConfigOverrideSetModule) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	// Accumulate the namespaces of the override modules listed, and set that as an
	// overrideSetProviderKey provider that device_config modules can read and use
	// to append overrides to their aconfig actions.
	namespaces := make(map[string]android.Paths)
	ctx.VisitDirectDeps(func(dep android.Module) {
		if !ctx.OtherModuleHasProvider(dep, overrideProviderKey) {
			// Other modules get injected as dependencies too, for example the license modules
			return
		}
		depData := ctx.OtherModuleProvider(dep, overrideProviderKey).(overrideProviderData)

		srcs := make([]android.Path, len(depData.Overrides))
		copy(srcs, depData.Overrides)
		namespaces[depData.Namespace] = srcs

	})
	ctx.SetProvider(overrideSetProviderKey, overrideSetProviderData{
		AvailableNamespaces: namespaces,
	})
}
