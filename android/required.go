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
// For a variation-tag-decorated required module, it would be rewritten to the Android.mk name of
// the soong module whose BaseModuleName() is "name" and ImageVariation() is {"image", tag}.
// Naturally, only soong modules can be "required" with the variation-tag-decorated form.

package android

import (
	"strings"

	"github.com/google/blueprint"
)

// Modules that support being specified with a variation-tag need to implement this interface.
type AndroidMkNamesInterface interface {
	// Returns the Android.mk names (LOCAL_MODULE) exported by this module. This is usually the
	// Module.base().BaseModuleName() + AndroidMkEntries.SubName of the module.
	AndroidMkNames() []string
}

type requiredDependencyTag struct {
	blueprint.BaseDependencyTag
}

var RequiredDepTag requiredDependencyTag

// If module reference is in the format of "name{.tag}" then return the name and tag, otherwise
// return empty strings.
func parseNameWithTag(module string) (name, tag string) {
	if tagBegin := strings.IndexByte(module, '{'); tagBegin > 0 {
		if tagEnd := len(module) - 1; module[tagEnd] == '}' {
			name = module[:tagBegin]
			tag = module[tagBegin+1 : tagEnd]
			return name, tag
		}
	}
	return "", ""
}

func requiredModuleMutator(ctx BottomUpMutatorContext) {
	// We only support device-device required for now.
	if !ctx.Device() {
		return
	}
	for _, requiredModule := range ctx.Module().RequiredModuleNames() {
		if name, tag := parseNameWithTag(requiredModule); name != "" {
			tag = strings.TrimPrefix(tag, ".")
			variations := append(ctx.Target().Variations(), blueprint.Variation{Mutator: "image", Variation: tag})
			dep := ctx.AddFarVariationDependencies(variations, RequiredDepTag, name)[0]
			if dep != nil {
				if _, ok := dep.(AndroidMkNamesInterface); !ok {
					ctx.PropertyErrorf("required", "%s must implement AndroidMkNamesInterface", dep)
				}
			}
		}
	}
}

type requiredModuleSingleton struct{}

func requiredModuleSingletonFactory() Singleton {
	return &requiredModuleSingleton{}
}

func (s *requiredModuleSingleton) GenerateBuildActions(ctx SingletonContext) {
	ctx.VisitAllModules(func(m Module) {
		var ret []string
		requiredModules := m.RequiredModuleNames()
		// We only support device-device required for now.
		if m.base().Device() {
			ret = make([]string, 0, len(requiredModules))
			for _, requiredModule := range requiredModules {
				if name, _ := parseNameWithTag(requiredModule); name == "" {
					ret = append(ret, requiredModule)
				}
			}
			ctx.VisitDirectDeps(m, func(dep Module) {
				if ctx.ModuleDependencyTag(m, dep) == RequiredDepTag {
					if dep.IsHideFromMake() || !m.Enabled() {
						if !ctx.Config().AllowMissingDependencies() {
							ctx.ModuleErrorf(m, "Required module is not exported to Make: %s", dep)
						}
						return
					}
					ret = append(ret, dep.(AndroidMkNamesInterface).AndroidMkNames()...)
				}
			})
			ret = FirstUniqueStrings(ret)
		} else {
			ret = CopyOf(requiredModules)
		}
		m.base().commonProperties.ResolvedRequired = ret
	})
}
