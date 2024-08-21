// Copyright 2024 Google Inc. All rights reserved.
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

// This file defines the "base configuration". The base configuration is meant to be the new
// home for many of the variables in ctx.Config() that originate from product config. The advantage
// is that certain modules can override the base configuration and build in a constant
// configuration, no matter what lunch product you're using. They do that by setting their
// `configuration` property to point towards a `base_configuration` module, and that module's
// properties define the base configuration. The configuration will be used for the module that
// requested it, and all of its transitive dependencies.

package android

import (
	"sort"
	"strings"
	"sync"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

func init() {
	RegisterBaseConfigModuleTypes(InitRegistrationContext)
}

func RegisterBaseConfigModuleTypes(ctx RegistrationContext) {
	ctx.RegisterModuleType("base_configuration", baseConfigurationModuleFactory)
}

var allBaseConfigModuleNamesLock sync.Mutex
var allBaseConfigModuleNamesOnceKey = NewOnceKey("all_base_config_module_names")

func RegisterBaseConfigMutator(ctx RegisterMutatorsContext) {
	ctx.BottomUp("set_base_configuration_providers", func(ctx BottomUpMutatorContext) {
		if m, ok := ctx.Module().(*baseConfigurationModule); ok {
			m.setProvider(ctx)

			// We need to collect a list of all base configuration modules to create
			// variants based off of until the variants on demand project is finished.
			allBaseConfigModuleNamesLock.Lock()
			defer allBaseConfigModuleNamesLock.Unlock()
			allBaseConfigModuleNames := ctx.Config().Once(allBaseConfigModuleNamesOnceKey, func() interface{} {
				return &[]string{""}
			}).(*[]string)
			*allBaseConfigModuleNames = append(*allBaseConfigModuleNames, ctx.ModuleName())
			firstUniqueInPlace(*allBaseConfigModuleNames)
			sort.Strings(*allBaseConfigModuleNames)
		}
	}).Parallel()
	ctx.Transition("base_config", &baseConfigMutator{}).NeverFar()
}

type BaseConfigInfo struct {
	VendorVars     map[string]map[string]string
	VendorVarTypes map[string]map[string]string
}

// Returns the value and type of a soong config variable, and whether it was set or not
func (c *BaseConfigInfo) SoongConfigVariable(namespace string, variable string) (string, string, bool) {
	ty := ""
	if vars, ok := c.VendorVarTypes[namespace]; ok {
		ty = vars[variable]
	}
	if vars, ok := c.VendorVars[namespace]; ok {
		if v, ok := vars[variable]; ok {
			return v, ty, true
		}
	}
	return "", "", false
}

var baseConfigProviderKeyEarly = blueprint.NewMutatorProvider[BaseConfigInfo]("set_base_configuration_providers")
var BaseConfigProviderKey = blueprint.NewMutatorProvider[BaseConfigInfo]("base_config_mutate")

type baseConfigurationModule struct {
	ModuleBase
	properties struct {
		// The soong config variables that will be set in the configuration.
		// Each string in the list must be of the format `namespace:varname:type:value`,
		// as soong doesn't have support for unpacking maps. The type may be either
		// "string" or "bool".
		Soong_config_variables []string
	}
}

func baseConfigurationModuleFactory() Module {
	m := &baseConfigurationModule{}
	InitAndroidModule(m)
	m.AddProperties(&m.properties)
	return m
}

func (m *baseConfigurationModule) setProvider(ctx BaseModuleContext) {
	result := BaseConfigInfo{}
	for _, v := range m.properties.Soong_config_variables {
		parts := strings.SplitN(v, ":", 4)
		if len(parts) < 4 {
			ctx.PropertyErrorf("soong_config_variables", "Expected a value in the format namespace:varname:type:value, got: %q", v)
			continue
		}
		namespace := parts[0]
		variable := parts[1]
		ty := parts[2]
		value := parts[3]

		if !isGoIdentifier(namespace) {
			ctx.PropertyErrorf("soong_config_variables", "soong config namespaces must be valid identifiers: %q", namespace)
			continue
		}
		if !isGoIdentifier(variable) {
			ctx.PropertyErrorf("soong_config_variables", "soong config variables must be valid identifiers: %q", variable)
			continue
		}
		if ty != "string" && ty != "bool" {
			ctx.PropertyErrorf("soong_config_variables", "soong config variables types must be either string or bool, found: %q", ty)
			continue
		}

		if result.VendorVars == nil {
			result.VendorVars = make(map[string]map[string]string)
			result.VendorVarTypes = make(map[string]map[string]string)
		}
		if result.VendorVars[namespace] == nil {
			result.VendorVars[namespace] = make(map[string]string)
			result.VendorVarTypes[namespace] = make(map[string]string)
		}

		result.VendorVars[namespace][variable] = value
		result.VendorVarTypes[namespace][variable] = ty
	}

	SetProvider(ctx, baseConfigProviderKeyEarly, result)
}

func (m *baseConfigurationModule) GenerateAndroidBuildActions(ctx ModuleContext) {
	// Nothing to do, the base configuration provider was set in setProvider().
	m.HideFromMake()
}

// Modules implementing this interface can have their base configuration changed
// in blueprint files. Even if a module doesn't implement this interface, its base
// configuration may be changed by a reverse dependency that does. Modules that
// want to allow changing their base configurations (currently, only the android
// system image is planned) should embed MutableConfigurationModuleBase and call
// InitMutableConfigurationModule() in order to implement this interface.
type MutableConfigurationModule interface {
	Module
	getMutableConfigurationModuleBase() *MutableConfigurationModuleBase

	// Returns the name of the module of type base_configuration that the
	// current module should be built with. A variant for all of the current
	// module's deps will be created using the base_configuration module's name,
	// and that new variant will be built using its configuration. The current
	// module will only have one variation, also built using the configuration
	// from the base_configuration module. An empty string can be returned to
	// use the global base configuration, which is also what modules that don't
	// implement this interface will use.
	getBaseConfigModule() string
}

// Embedding this struct will allow a module to implement MutableConfigurationModule.
// If embeddeding this struct, you must also call InitMutableConfigurationModule().
type MutableConfigurationModuleBase struct {
	properties struct {
		// The name of a `base_configuration` module that defines the configuration
		// to use for this module and all of its dependencies. It is intentionally
		// non-configurable because it is part of deciding the configuration.
		Configuration *string
	}
}

func (m *MutableConfigurationModuleBase) getMutableConfigurationModuleBase() *MutableConfigurationModuleBase {
	return m
}

func (m *MutableConfigurationModuleBase) getBaseConfigModule() string {
	return proptools.String(m.properties.Configuration)
}

func InitMutableConfigurationModule(m MutableConfigurationModule) {
	base := m.getMutableConfigurationModuleBase()
	m.AddProperties(&base.properties)
}

// baseConfigMutator allows changing the global "base configuration" that modules are built
// under. The base configuration is a concept that supersedes the blueprint Config object and
// allows changing it for certain modules in the build.
//
// If a module explicitly requests a certain configuration (by returning a configuration module
// name from GetBaseConfigModule()), it will only have the empty string variant. Otherwise,
// modules will have the empty string variant for the global configuration, and a variant
// for each configuration module that its transitive reverse dependencies use.
type baseConfigMutator struct {
	global_config             BaseConfigInfo
	global_config_initialized bool
}

var _ TransitionMutator = (*baseConfigMutator)(nil)

func (*baseConfigMutator) Split(ctx BaseModuleContext) []string {
	if _, ok := ctx.Module().(*baseConfigurationModule); ok {
		// Modules that actually define the configuration are never built under
		// alternate configurations.
		return []string{""}
	}
	if m, ok := ctx.Module().(MutableConfigurationModule); ok {
		// Modules with a constant configuration set in the bp file will always use that
		// configuration, so we can just create the 1 variant.
		if baseConfigModule := m.getBaseConfigModule(); baseConfigModule != "" {
			return []string{baseConfigModule}
		}
	}
	// Until soong has fully realized the "variants on demand" project, we need to
	// know the full list of possible variants and create them during this mutator,
	// in case later mutators add dependencies on the new variants. Ideally we would
	// just return []string{""} here and have soong/blueprint figure out the rest,
	// but that's not available yet.
	allBaseConfigModuleNames := ctx.Config().Once(allBaseConfigModuleNamesOnceKey, func() interface{} {
		return &[]string{""}
	}).(*[]string)
	return *allBaseConfigModuleNames
}

func (*baseConfigMutator) OutgoingTransition(ctx OutgoingTransitionContext, sourceVariation string) string {
	if m, ok := ctx.Module().(MutableConfigurationModule); ok {
		if baseConfigModule := m.getBaseConfigModule(); baseConfigModule != "" {
			return baseConfigModule
		}
	}
	return sourceVariation
}

func (*baseConfigMutator) IncomingTransition(ctx IncomingTransitionContext, incomingVariation string) string {
	if m, ok := ctx.Module().(MutableConfigurationModule); ok && m.getBaseConfigModule() != "" {
		if baseConfigModule := m.getBaseConfigModule(); baseConfigModule != "" {
			// This module is specified to always build under a certain configuration, so always use
			// the that variant.
			return baseConfigModule
		}
	}
	if _, ok := ctx.Module().(*baseConfigurationModule); ok {
		// Modules that actually define the configuration are never built under
		// alternate configurations.
		return ""
	}

	return incomingVariation
}

var globalBaseConfigOnceKey = NewOnceKey("global_base_config_once_key")

type BaseConfigModuleDepTag struct {
	blueprint.BaseDependencyTag
}

func (b *baseConfigMutator) Mutate(ctx BottomUpMutatorContext, variation string) {
	isModuleChangingConfig := false
	if m, ok := ctx.Module().(MutableConfigurationModule); ok {
		if x := m.getBaseConfigModule(); x != "" {
			isModuleChangingConfig = true
		}
	}

	if variation != "" {
		configModule := ctx.AddDependency(ctx.Module(), BaseConfigModuleDepTag{}, variation)[0]
		cfg, ok := OtherModuleProvider(ctx, configModule, baseConfigProviderKeyEarly)
		if !ok {
			ctx.ModuleErrorf("Expected a base_configuration module to be used as configuration")
			return
		}
		SetProvider(ctx, BaseConfigProviderKey, cfg)

		if !isModuleChangingConfig {
			// Because we only intend on using this feature for the soong-built system image, the
			// dependencies of the system image that are built under a different base config don't
			// need to be exported to make. However, their global base configuration variants
			// and the system image itself should still be exported to make, to maintain the same
			// behavior as soong before base configurations were added.
			//
			// We know we're building an extra configuration for the system image if the variation
			// is not "", and that variation is not the only variation.
			// (isModuleChangingConfig=true)
			ctx.Module().HideFromMake()
		}
	} else {
		globalBaseConfig := ctx.Config().Once(globalBaseConfigOnceKey, func() interface{} {
			return createGlobalBaseConfig(ctx.Config())
		}).(BaseConfigInfo)

		SetProvider(ctx, BaseConfigProviderKey, globalBaseConfig)
	}
}

func createGlobalBaseConfig(c Config) BaseConfigInfo {
	var result BaseConfigInfo
	result.VendorVars = c.productVariables.VendorVars
	result.VendorVarTypes = c.productVariables.VendorVarTypes
	return result
}
