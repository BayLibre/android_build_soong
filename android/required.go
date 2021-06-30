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

package android

// This file supports specifying image-variation-augmented module names in the "required" property.
// Module names specified in the "required" property would be installed whenever the specifying
// module is installed.
//
// If a required module name is in the form of "name{.tag}" then it is image-variation augmented.
// Otherwise the module name is plain Android.mk module name and is passed to Make as-is.
// For an image-variation-augmented required module name, the name would be rewritten to the
// Android.mk name of the module whose BaseModuleName() is "name" and
// ImageVariationForRequiredModule() is "tag".
//
// Naturally, only soong modules can be "required" by the image-variation-augmented form.

import (
	"fmt"
	"strings"
)

// Modules that support specifying image variation tag in "required" need to implement this interface.
type RequiredModuleImageVariarionInterface interface {
	// Modules implementing this interface must be split by the imageMutator.
	ImageInterface

	// Returns true if this module should not be registered to requiredModuleImageVariationSingleton.
	// Only modules registered to the singleton can be specified with a image variation tag.
	SkipRequiredModuleImageVariationSingleton() bool

	// Returns the image variation tag for specifying this module in "required".
	ImageVariationForRequiredModule() string

	// Returns the Android.mk name (LOCAL_MODULE) exported by this module.
	AndroidMkNameForRequiredModule() string
}

const (
	RequiredModuleCoreVariation          string = "platform"
	RequiredModuleProductVariation       string = "product"
	RequiredModuleVendorVariation        string = "vendor"
	RequiredModuleRecoveryVariation      string = "recovery"
	RequiredModuleRamdiskVariation       string = "ramdisk"
	RequiredModuleVendorRamdiskVariation string = "vendor_ramdisk"
)

var requiredModuleImageVariationMapKey = NewOnceKey("required_module_image_variation_map_key")

// BaseModuleName() -> Variation name -> [Android.mk name]
type requiredModuleImageVariationMap map[string]map[string][]string

func getRequiredModuleImageVariationMap(config Config) requiredModuleImageVariationMap {
	return config.Once(requiredModuleImageVariationMapKey, func() interface{} {
		return make(requiredModuleImageVariationMap)
	}).(requiredModuleImageVariationMap)
}

type requiredModuleImageVariationSingleton struct{}

func requiredModuleImageVariationSingletonFactory() Singleton {
	return &requiredModuleImageVariationSingleton{}
}

func (s *requiredModuleImageVariationSingleton) GenerateBuildActions(ctx SingletonContext) {
	moduleVariationMap := getRequiredModuleImageVariationMap(ctx.Config())

	ctx.VisitAllModules(func(m Module) {
		// Skip if module is not for device.
		if m.Target().Os.Class != Device {
			return
		}
		// Skip if module is not exported to Make.
		if m.IsHideFromMake() || !m.Enabled() {
			return
		}

		if r, ok := m.(RequiredModuleImageVariarionInterface); ok {
			if r.SkipRequiredModuleImageVariationSingleton() {
				return
			}

			variation := r.ImageVariationForRequiredModule()
			if variation == "" {
				return
			}

			name := m.base().BaseModuleName()
			androidMkName := r.AndroidMkNameForRequiredModule()

			variationMap := moduleVariationMap[name]
			if variationMap == nil {
				variationMap = make(map[string][]string)
				moduleVariationMap[name] = variationMap
			}
			variationMap[variation] = append(variationMap[variation], androidMkName)
		}
	})

	// Remove any duplicates and sort for stable result.
	for _, variationMap := range moduleVariationMap {
		for k, v := range variationMap {
			variationMap[k] = SortedUniqueStrings(v)
		}
	}
}

// Rewrites any image-variation-augmented module names to their Android.mk form.
func rewriteRequiredModuleWithImageVariation(config Config, modules []string) []string {
	var moduleVariationMap requiredModuleImageVariationMap
	ret := make([]string, 0, len(modules))
	for _, module := range modules {
		ret = append(ret, rewriteOneRequiredModule(config, &moduleVariationMap, module)...)
	}
	return ret
}

func rewriteOneRequiredModule(config Config, moduleVariationMap *requiredModuleImageVariationMap, module string) []string {
	if tagBegin := strings.IndexByte(module, '{'); tagBegin != -1 {
		if tagEnd := len(module) - 1; module[tagEnd] == '}' {
			name := module[:tagBegin]
			imageTag := strings.TrimPrefix(module[tagBegin+1:tagEnd], ".")
			if name == "" || imageTag == "" {
				panic(fmt.Errorf("Invalid required module specifier: %s", module))
			}
			// Access the module variation map on-demand so we don't call config.Once() too often.
			if *moduleVariationMap == nil {
				*moduleVariationMap = getRequiredModuleImageVariationMap(config)
			}
			fmt.Println(name, imageTag, (*moduleVariationMap)[name])
			if variationMap := (*moduleVariationMap)[name]; variationMap != nil {
				if names := variationMap[imageTag]; len(names) > 0 {
					return names
				}
			}
			if !config.AllowMissingDependencies() {
				panic(fmt.Errorf("Cannot find required module with requested name and image variation: %s", module))
			}
		}
	}
	return []string{module}
}
