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
	"fmt"
	"github.com/google/blueprint"
	"strings"
)

type DeviceConfigModule struct {
	android.ModuleBase
	android.DefaultableModuleBase

	// Properties for "device_config"
	properties struct {
		// aconfig files, relative to this Android.bp file
		Srcs []string `android:"path"`

		// Release config flag namespace
		Namespace string

		// Overrides from TARGET_RELEASE / RELEASE_DEVICE_CONFIG_OVERRIDES
		Overrides []string `blueprint:"mutated"`
	}

	intermediatePath android.WritablePath
	srcJarPath       android.WritablePath
}

func DeviceConfigFactory() android.Module {
	module := &DeviceConfigModule{}

	android.InitAndroidModule(module)
	android.InitDefaultableModule(module)
	module.AddProperties(&module.properties)
	// TODO: bp2build
	//android.InitBazelModule(module)

	return module
}

type implicitOverridesTagType struct {
	blueprint.BaseDependencyTag
}

var implicitOverridesTag = implicitOverridesTagType{}

func (module *DeviceConfigModule) DepsMutator(ctx android.BottomUpMutatorContext) {
	// Validate Properties
	if len(module.properties.Srcs) == 0 {
		ctx.PropertyErrorf("srcs", "missing source files")
		return
	}
	if len(module.properties.Namespace) == 0 {
		ctx.PropertyErrorf("namespace", "missing namespace property")
	}

	// Add a dependency on the device_config_override_sets defined in
	// RELEASE_DEVICE_CONFIG_OVERRIDES, and add any device_config_overrides that
	// match our namespace.
	overridesFromConfig := ctx.Config().ReleaseDeviceConfigOverrides()
	ctx.AddDependency(ctx.Module(), implicitOverridesTag, overridesFromConfig...)
}

func (module *DeviceConfigModule) OutputFiles(tag string) (android.Paths, error) {
	switch tag {
	case ".srcjar":
		return []android.Path{module.srcJarPath}, nil
	case "":
		// The default output of this module is the intermediates format, which is
		// not installable and in a private format that no other rules can handle
		// correctly.
		return []android.Path{module.intermediatePath}, nil
	default:
		return nil, fmt.Errorf("unsupported device_config module reference tag %q", tag)
	}
}

func (module *DeviceConfigModule) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	// Get the overrides that came from the global RELEASE_DEVICE_CONFIG_OVERRIDES flag
	ctx.VisitDirectDeps(func(dep android.Module) {
		if !ctx.OtherModuleHasProvider(dep, overrideSetProviderKey) {
			// Other modules get injected as dependencies too, for example the license modules
			return
		}
		depData := ctx.OtherModuleProvider(dep, overrideSetProviderKey).(overrideSetProviderData)
		overrideFiles, ok := depData.AvailableNamespaces[module.properties.Namespace]
		if ok {
			for _, path := range overrideFiles {
				module.properties.Overrides = append(module.properties.Overrides, path.String())
			}
		}
	})

	// Intermediate format
	inputFiles := android.PathsForModuleSrc(ctx, module.properties.Srcs)
	module.intermediatePath = android.PathForModuleOut(ctx, "intermediate.json")
	ctx.Build(pctx, android.BuildParams{
		Rule:        aconfigRule,
		Inputs:      inputFiles,
		Output:      module.intermediatePath,
		Description: "device_config",
		Args: map[string]string{
			"overrides": strings.Join(module.properties.Overrides, " "),
		},
	})

	// Generated java inside a srcjar
	module.srcJarPath = android.PathForModuleGen(ctx, ctx.ModuleName()+".srcjar")
	ctx.Build(pctx, android.BuildParams{
		Rule:        srcJarRule,
		Input:       module.intermediatePath,
		Output:      module.srcJarPath,
		Description: "device_config.srcjar",
	})

	// TODO: C++

	// Phony target for debugging convenience
	ctx.Build(pctx, android.BuildParams{
		Rule:   blueprint.Phony,
		Output: android.PathForPhony(ctx, ctx.ModuleName()),
		Inputs: []android.Path{module.srcJarPath}, // TODO: C++
	})
}
