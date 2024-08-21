package android

import (
	"strings"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

func init() {
	RegisterBaseConfigModuleTypes(InitRegistrationContext)
}

func RegisterBaseConfigModuleTypes(ctx RegistrationContext) {
	ctx.RegisterModuleType("base_configuration", baseConfigurationModuleFactory)
}

func RegisterBaseConfigMutator(ctx RegisterMutatorsContext) {
	ctx.BottomUp("set_base_configuration_providers", func(ctx BottomUpMutatorContext) {
		if m, ok := ctx.Module().(*baseConfigurationModule); ok {
			m.setProvider(ctx)
		}
	}).Parallel()
	ctx.Transition("base_config", &baseConfigMutator{})
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
		// as soong doesn't have support for unpacking maps.
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
		if ty == "string" {
			// The rest of soong assumes strings by default
			ty = ""
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

type MutableConfigurationModuleBase struct {
	properties struct {
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
// under. The base configuration is a concept that supercedes the blueprint Config object and
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
	return []string{"", "my_base_config"}
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
		// This module is specified to always build under a certain configuration, so always use
		// the empty variant.
		return ""
	}

	return incomingVariation
}

var globalBaseConfigOnceKey = NewOnceKey("global_base_config_once_key")

func (b *baseConfigMutator) Mutate(ctx BottomUpMutatorContext, variation string) {
	configModuleName := variation
	if m, ok := ctx.Module().(MutableConfigurationModule); ok {
		if x := m.getBaseConfigModule(); x != "" {
			configModuleName = x
		}
	}

	if configModuleName != "" && ctx.ModuleName() != configModuleName {
		type configModuleDepTag struct {
			blueprint.BaseDependencyTag
		}

		configModule := ctx.AddDependency(ctx.Module(), configModuleDepTag{}, configModuleName)[0]
		cfg, ok := OtherModuleProvider(ctx, configModule, baseConfigProviderKeyEarly)
		if !ok {
			ctx.ModuleErrorf("Expected a base_configuration module to be used as configuration")
			return
		}
		SetProvider(ctx, BaseConfigProviderKey, cfg)
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
