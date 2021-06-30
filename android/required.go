// Copyright 2021 The Android Open Source Project
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This file supports specifying variation-tag-decorated module names in the "required" property.
// Module names specified in the "required" property would be installed whenever the specifying
// module is installed.
//
// If a required module name is in the form of "name{.tag}" then it is decorated with a variation
// tag. Otherwise the module name is plain Android.mk module name and is passed to Make as-is.
// For an variation-tag-decorated required module, the name would be rewritten to the Android.mk
// name of the soong module whose BaseModuleName() is "name" and VariationForRequiredModule() is
// "tag". Naturally, only soong modules can be "required" with the variation-tag-decorated form.

package android

import (
	"strings"
)

// Modules that support specifying variation tag in "required" need to implement this interface.
type RequiredModuleVariarionInterface interface {
	// Returns true if this module should not be registered to requiredModuleVariationSingleton.
	// Only modules registered to the singleton can be required-specified with a variation tag.
	SkipRequiredModuleVariationSingleton() bool

	// Returns the variation tag for specifying this module in "required".
	VariationForRequiredModule() string

	// Returns the Android.mk names (LOCAL_MODULE) exported by this module. This is usually the
	// Module.base().BaseModuleName() + AndroidMkEntries.SubName of the module.
	AndroidMkNamesForRequiredModule() []string
}

const (
	RequiredModuleCoreVariation          string = "platform"
	RequiredModuleRecoveryVariation      string = "recovery"
	RequiredModuleRamdiskVariation       string = "ramdisk"
	RequiredModuleVendorRamdiskVariation string = "vendor_ramdisk"
)

// BaseModuleName() -> Variation name -> list of Android.mk names
type requiredModuleVariationMap map[string]map[string][]string

type requiredModuleVariationSingleton struct{}

func requiredModuleVariationSingletonFactory() Singleton {
	return &requiredModuleVariationSingleton{}
}

func (s *requiredModuleVariationSingleton) GenerateBuildActions(ctx SingletonContext) {
	moduleVariationMap := make(requiredModuleVariationMap)

	ctx.VisitAllModules(func(m Module) {
		// Skip if module is not for device.
		if m.Target().Os.Class != Device {
			return
		}
		// Skip if module is not exported to Make.
		if m.IsHideFromMake() || !m.Enabled() {
			return
		}

		if r, ok := m.(RequiredModuleVariarionInterface); ok && !r.SkipRequiredModuleVariationSingleton() {
			variation := r.VariationForRequiredModule()
			if variation == "" {
				return
			}

			name := m.base().BaseModuleName()
			variationMap := moduleVariationMap[name]
			if variationMap == nil {
				variationMap = make(map[string][]string)
				moduleVariationMap[name] = variationMap
			}

			androidMkNames := r.AndroidMkNamesForRequiredModule()
			variationMap[variation] = append(variationMap[variation], androidMkNames...)
		}
	})

	// Remove any duplicates and sort for stable result.
	for _, variationMap := range moduleVariationMap {
		for k, v := range variationMap {
			variationMap[k] = SortedUniqueStrings(v)
		}
	}

	// Resolve all module names with variation-tag.
	ctx.VisitAllModules(func(m Module) {
		var ret []string
		requiredModules := m.RequiredModuleNames()
		mod := m.base()
		// Skip if module is not for device.
		if mod.Device() {
			ret = make([]string, 0, len(requiredModules))
			for _, requiredModule := range requiredModules {
				ret = append(ret, resolveOneRequiredModule(ctx, m, moduleVariationMap, requiredModule)...)
			}
			ret = FirstUniqueStrings(ret)
		} else {
			ret = CopyOf(requiredModules)
		}
		mod.commonProperties.ResolvedRequired = ret
	})
}

func resolveOneRequiredModule(ctx SingletonContext, m Module, moduleVariationMap requiredModuleVariationMap, module string) []string {
	if tagBegin := strings.IndexByte(module, '{'); tagBegin != -1 {
		if tagEnd := len(module) - 1; module[tagEnd] == '}' {
			name := module[:tagBegin]
			tag := strings.TrimPrefix(module[tagBegin+1:tagEnd], ".")
			if name == "" || tag == "" {
				ctx.ModuleErrorf(m, "Invalid required module: %s", module)
				return nil
			}
			if variationMap := moduleVariationMap[name]; variationMap != nil {
				if names := variationMap[tag]; len(names) > 0 {
					return names
				}
			}
			if !ctx.Config().AllowMissingDependencies() {
				ctx.ModuleErrorf(m, "Cannot find required module with requested name and variation: %s", module)
			}
			return nil
		}
	}
	return []string{module}
}
