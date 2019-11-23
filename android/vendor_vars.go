// Copyright 2019 Google Inc. All rights reserved.
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

package android

// This file provides module types that implement wrapper module types that add conditionals on
// vendor variables.

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"text/scanner"

	"github.com/google/blueprint"
	"github.com/google/blueprint/parser"
	"github.com/google/blueprint/proptools"
)

func init() {
	RegisterModuleType("soong_vendor_vars_import", vendorVarsImportFactory)
}

var vendorVariableProperty = proptools.FieldNameForProperty("vendor_variables")

type vendorVarsImport struct {
	ModuleBase
	properties vendorVarsImportProperties
}

type vendorVarsImportProperties struct {
	From         string
	Module_types []string
}

// soong_vendor_vars_import reads a VendorVars.bp file and imports module types with conditionals
// on vendor variables.  For example, an Android.bp file could have:
//
// 	   soong_vendor_vars_import {
//         from: "device/acme/VendorVars.bp",
//         module_types: ["acme_cc_defaults"],
//     }
//
//     acme_cc_defaults {
//         name: "acme_defaults",
//         cflags: ["-DGENERIC"],
//         vendor_variables: {
//             board: {
//                 soc_a: {
//                     cflags: ["-DSOC_A"],
//                 },
//                 soc_b: {
//                     cflags: ["-DSOC_B"],
//                 },
//             },
//             feature: {
//                 cflags: ["-DFEATURE"],
//             },
//         },
//     }
//
//     cc_library {
//         name: "libacme_foo",
//         defaults: ["acme_defaults"],
//         srcs: ["*.cpp"],
//     }
//
// And device/acme/Vendors.bp could have:
//
//     soong_vendor_vars_module_type {
//         name: "acme_cc_defaults",
//         module_type: "cc_defaults",
//         config_namespace: "acme",
//         variables: ["board", "feature"],
//         properties: ["cflags", "srcs"],
//     }
//
//     soong_vendor_vars_string_variable {
//         name: "board",
//         values: ["soc_a", "soc_b"],
//     }
//
//     soong_vendor_vars_bool_variable {
//         name: "feature",
//     }
//
// If an acme BoardConfig.mk file contained:
//
//     SOONG_CONFIG_NAMESPACES += acme
//     SOONG_CONFIG_acme += \
//         board \
//         feature \
//
//     SOONG_CONFIG_acme_board := soc_a
//     SOONG_CONFIG_acme_feature := true
//
// Then libacme_foo would build with cflags "-DGENERIC -DSOC_A -DFEATURE".
func vendorVarsImportFactory() Module {
	module := &vendorVarsImport{}

	module.AddProperties(&module.properties)
	AddLoadHook(module, func(ctx LoadHookContext) {
		importModuleTypes(ctx, module.properties.From, module.properties.Module_types)
	})

	InitAndroidModule(module)
	return module
}

func (v *vendorVarsImport) Name() string {
	return "soong_vendor_vars_import_" + canonicalizeToProperty(v.properties.From)
}

func (*vendorVarsImport) GenerateAndroidBuildActions(ModuleContext) {}

func importModuleTypes(ctx LoadHookContext, from string, moduleTypes []string) {
	if filepath.Base(from) != "VendorVars.bp" {
		ctx.PropertyErrorf("from",
			"%q must be the full path from the top of the source tree to a file named VendorVars.bp",
			from)
		return
	}

	vendorVarsDefinition := loadVendorVarsDefinition(ctx, from)
	if vendorVarsDefinition == nil {
		ctx.PropertyErrorf("from",
			"failed to parse %q", from)
		return
	}
	for _, moduleType := range moduleTypes {
		if factory, ok := vendorVarsDefinition.factories[moduleType]; ok {
			ctx.registerScopedModuleType(moduleType, factory)
		} else {
			ctx.PropertyErrorf("module_types", "module type %q not defined in %q",
				moduleType, from)
		}
	}
}

func reportErrors(ctx LoadHookContext, filename string, errs ...error) {
	for _, err := range errs {
		if parseErr, ok := err.(*parser.ParseError); ok {
			ctx.Errorf(parseErr.Pos, "%s", parseErr.Err)
		} else {
			ctx.Errorf(scanner.Position{Filename: filename}, "%s", err)
		}
	}
}

// loadVendorVarsDefinition loads the contents of a VendorVars.bp file.  It caches the result
// so each file is only parsed once.
func loadVendorVarsDefinition(ctx LoadHookContext, from string) *vendorVarsDefinition {
	type onceKeyType string
	key := NewCustomOnceKey(onceKeyType(filepath.Clean(from)))

	return ctx.Config().Once(key, func() interface{} {
		r, err := ctx.Config().fs.Open(from)
		if err != nil {
			ctx.PropertyErrorf("from", "failed to open %q: %s", from, err)
			return (*vendorVarsDefinition)(nil)
		}
		scope := parser.NewScope(nil)
		file, errs := parser.ParseAndEval(from, r, scope)

		if len(errs) > 0 {
			reportErrors(ctx, from, errs...)
			return (*vendorVarsDefinition)(nil)
		}

		vendorVars := &vendorVarsDefinition{
			moduleTypes: make(map[string]*vendorModuleType),
			variables:   make(map[string]vendorVariable),
			factories:   make(map[string]blueprint.ModuleFactory),
		}

		for _, def := range file.Defs {
			switch def := def.(type) {
			case *parser.Module:
				newErrs := processImportModuleDef(vendorVars, def)

				if len(newErrs) > 0 {
					errs = append(errs, newErrs...)
				}

			case *parser.Assignment:
				// Already handled via Scope object
			default:
				panic("unknown definition type")
			}
		}

		if len(errs) > 0 {
			reportErrors(ctx, from, errs...)
			return (*vendorVarsDefinition)(nil)
		}

		globalModuleTypes := ctx.moduleFactories()

		for name, moduleType := range vendorVars.moduleTypes {
			for _, varName := range moduleType.variableNames {
				if v, ok := vendorVars.variables[varName]; ok {
					moduleType.variables = append(moduleType.variables, v)
				} else {
					reportErrors(ctx, from,
						fmt.Errorf("unknown variable %q in module type %q", varName, name))
				}
			}

			factory := globalModuleTypes[moduleType.baseModuleType]
			if factory != nil {
				vendorVars.factories[name] = vendorModuleFactory(factory, moduleType)
			} else {
				reportErrors(ctx, from,
					fmt.Errorf("missing global module type factory for %q", moduleType.baseModuleType))
			}
		}

		if ctx.Failed() {
			return (*vendorVarsDefinition)(nil)
		}

		return vendorVars
	}).(*vendorVarsDefinition)
}

func processImportModuleDef(v *vendorVarsDefinition, def *parser.Module) (errs []error) {
	switch def.Type {
	case "soong_vendor_vars_module_type":
		return processModuleTypeDef(v, def)
	case "soong_vendor_vars_string_variable":
		return processStringVariableDef(v, def)
	case "soong_vendor_vars_bool_variable":
		return processBoolVariableDef(v, def)
	default:
		return []error{fmt.Errorf("unknown type %q", def.Type)}
	}
}

func processModuleTypeDef(v *vendorVarsDefinition, def *parser.Module) (errs []error) {
	type importProperties struct {
		Name             string
		Module_type      string
		Config_namespace string
		Variables        []string
		Properties       []string
	}

	props := &importProperties{}

	_, errs = proptools.UnpackProperties(def.Properties, props)
	if len(errs) > 0 {
		return errs
	}

	if props.Name == "" {
		errs = append(errs, fmt.Errorf("name property must be set"))
	}

	if props.Config_namespace == "" {
		errs = append(errs, fmt.Errorf("config_namespace property must be set"))
	}

	if props.Module_type == "" {
		errs = append(errs, fmt.Errorf("module_type property must be set"))
	}

	if len(errs) > 0 {
		return errs
	}

	mt := &vendorModuleType{
		affectableProperties: props.Properties,
		configNamespace:      props.Config_namespace,
		baseModuleType:       props.Module_type,
		variableNames:        props.Variables,
	}
	v.moduleTypes[props.Name] = mt

	return nil
}

func processStringVariableDef(v *vendorVarsDefinition, def *parser.Module) (errs []error) {
	type stringVariableProperties struct {
		Values []string
	}

	stringProps := &stringVariableProperties{}

	base, errs := processVariableDef(def, stringProps)
	if len(errs) > 0 {
		return errs
	}

	if len(stringProps.Values) == 0 {
		return []error{fmt.Errorf("values property must be set")}
	}

	v.variables[base.vendorVar] = &stringVariable{
		baseVariable: base,
		values:       canonicalizeToProperties(stringProps.Values),
	}

	return nil
}

func processBoolVariableDef(v *vendorVarsDefinition, def *parser.Module) (errs []error) {
	base, errs := processVariableDef(def)
	if len(errs) > 0 {
		return errs
	}

	v.variables[base.vendorVar] = &boolVariable{
		baseVariable: base,
	}

	return nil
}

func processVariableDef(def *parser.Module,
	extraProps ...interface{}) (cond baseVariable, errs []error) {

	type variableProperties struct {
		Name string
	}

	props := &variableProperties{}

	allProps := append([]interface{}{props}, extraProps...)

	_, errs = proptools.UnpackProperties(def.Properties, allProps...)
	if len(errs) > 0 {
		return baseVariable{}, errs
	}

	if props.Name == "" {
		return baseVariable{}, []error{fmt.Errorf("name property must be set")}
	}

	return baseVariable{
		vendorVar: props.Name,
	}, nil
}

type vendorVarsDefinition struct {
	moduleTypes map[string]*vendorModuleType
	variables   map[string]vendorVariable

	factories map[string]blueprint.ModuleFactory
}

// vendorModuleFactory takes an existing ModuleFactory and a vendorModuleType and returns a new
// ModuleFactory that wraps the existing ModuleFactory and adds vendor conditionals.
func vendorModuleFactory(factory blueprint.ModuleFactory,
	moduleType *vendorModuleType) blueprint.ModuleFactory {

	conditionalFactoryProps := createProperties(factory, moduleType)
	if conditionalFactoryProps.IsValid() {
		return func() (blueprint.Module, []interface{}) {
			module, props := factory()

			conditionalProps := proptools.CloneEmptyProperties(conditionalFactoryProps.Elem())
			props = append(props, conditionalProps.Interface())

			AddLoadHook(module, func(ctx LoadHookContext) {
				applyVendorConditionals(ctx, moduleType, conditionalProps)
			})

			return module, props
		}
	} else {
		return factory
	}
}

// applyVendorConditionals appends the applicable properties from a vendorModuleType based on
// Config.VendorConfig values.
func applyVendorConditionals(ctx LoadHookContext, moduleType *vendorModuleType, props reflect.Value) {
	props = props.Elem().FieldByName(vendorVariableProperty)
	vendorConfig := ctx.Config().VendorConfig(moduleType.configNamespace)
	for i, c := range moduleType.variables {
		if ps := c.propertiesToApply(vendorConfig, props.Field(i)); ps != nil {
			ctx.AppendProperties(ps)
		}
	}
}

// createProperties returns a reflect.Value of a newly constructed type that contains the desired property layout
// for the vendor variables, with each possible value an interface{} that contains a nil pointer to another newly
// constructed type that contains the affectable properties.  The reflect.Value will be cloned for each call to the
// vendor variable module type's factory method.
//
// For example, the acme_cc_defaults example above would
// produce a reflect.Value whose type is:
// *struct {
//     Vendor_variables struct {
//         Board struct {
//             Soc_a interface{}
//             Soc_b interface{}
//         }
//     }
// }
// And whose value is:
// &{
//	   Vendor_variables: {
//		   Board: {
//             Soc_a: (*struct{ Cflags []string })(nil),
//	           Soc_b: (*struct{ Cflags []string })(nil),
//         },
//	   },
// }
func createProperties(factory blueprint.ModuleFactory, moduleType *vendorModuleType) reflect.Value {
	var fields []reflect.StructField

	_, factoryProps := factory()
	affectablePropertiesType := createAffectablePropertiesType(moduleType.affectableProperties, factoryProps)
	if affectablePropertiesType == nil {
		return reflect.Value{}
	}

	for _, c := range moduleType.variables {
		fields = append(fields, reflect.StructField{
			Name: proptools.FieldNameForProperty(c.variableProperty()),
			Type: c.variableValuesType(),
		})
	}

	typ := reflect.StructOf([]reflect.StructField{{
		Name: vendorVariableProperty,
		Type: reflect.StructOf(fields),
	}})

	props := reflect.New(typ)
	structConditions := props.Elem().FieldByName(vendorVariableProperty)

	for i, c := range moduleType.variables {
		c.initializeProperties(structConditions.Field(i), affectablePropertiesType)
	}

	return props
}

// createAffectablePropertiesType creates a reflect.Type of a struct that has a field for each affectable property
// that exists in factoryProps.
func createAffectablePropertiesType(affectableProperties []string, factoryProps []interface{}) reflect.Type {
	affectableProperties = append([]string(nil), affectableProperties...)
	sort.Strings(affectableProperties)

	var recurse func(prefix string, aps []string) ([]string, reflect.Type)
	recurse = func(prefix string, aps []string) ([]string, reflect.Type) {
		var fields []reflect.StructField

		for len(affectableProperties) > 0 {
			p := affectableProperties[0]
			if !strings.HasPrefix(affectableProperties[0], prefix) {
				break
			}
			affectableProperties = affectableProperties[1:]

			nestedProperty := strings.TrimPrefix(p, prefix)
			if i := strings.IndexRune(nestedProperty, '.'); i >= 0 {
				var nestedType reflect.Type
				nestedPrefix := nestedProperty[:i+1]

				affectableProperties, nestedType = recurse(prefix+nestedPrefix, affectableProperties)

				if nestedType != nil {
					nestedFieldName := proptools.FieldNameForProperty(strings.TrimSuffix(nestedPrefix, "."))

					fields = append(fields, reflect.StructField{
						Name: nestedFieldName,
						Type: nestedType,
					})
				}
			} else {
				typ := typeForPropertyFromPropertyStructs(factoryProps, p)
				if typ != nil {
					fields = append(fields, reflect.StructField{
						Name: proptools.FieldNameForProperty(nestedProperty),
						Type: typ,
					})
				}
			}
		}

		var typ reflect.Type
		if len(fields) > 0 {
			typ = reflect.StructOf(fields)
		}
		return affectableProperties, typ
	}

	affectableProperties, typ := recurse("", affectableProperties)
	if len(affectableProperties) > 0 {
		panic(fmt.Errorf("didn't handle all affectable properties"))
	}

	if typ != nil {
		return reflect.PtrTo(typ)
	}

	return nil
}

func typeForPropertyFromPropertyStructs(psList []interface{}, property string) reflect.Type {
	for _, ps := range psList {
		if typ := typeForPropertyFromPropertyStruct(ps, property); typ != nil {
			return typ
		}
	}

	return nil
}

func typeForPropertyFromPropertyStruct(ps interface{}, property string) reflect.Type {
	v := reflect.ValueOf(ps)
	for len(property) > 0 {
		if !v.IsValid() {
			return nil
		}

		if v.Kind() == reflect.Interface {
			if v.IsNil() {
				return nil
			} else {
				v = v.Elem()
			}
		}

		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				v = reflect.Zero(v.Type().Elem())
			} else {
				v = v.Elem()
			}
		}

		if v.Kind() != reflect.Struct {
			return nil
		}

		if index := strings.IndexRune(property, '.'); index >= 0 {
			prefix := property[:index]
			property = property[index+1:]

			v = v.FieldByName(proptools.FieldNameForProperty(prefix))
		} else {
			f := v.FieldByName(proptools.FieldNameForProperty(property))
			if !f.IsValid() {
				return nil
			}
			return f.Type()
		}
	}
	return nil
}

type vendorModuleType struct {
	affectableProperties []string
	configNamespace      string
	baseModuleType       string
	variableNames        []string

	variables []vendorVariable
}

type vendorVariable interface {
	// variableProperty returns the name of the variable.
	variableProperty() string

	// conditionalValuesType returns a reflect.Type that contains an interface{} for each possible value.
	variableValuesType() reflect.Type

	// initializeProperties is passed a reflect.Value of the reflect.Type returned by conditionalValuesType and a
	// reflect.Type of the affectable properties, and should initialize each interface{} in the reflect.Value with
	// the zero value of the affectable properties type.
	initializeProperties(v reflect.Value, typ reflect.Type)

	// propertiesToApply should return one of the interface{} values set by initializeProperties to be applied
	// to the module.
	propertiesToApply(vendorConfig VendorConfig, values reflect.Value) interface{}
}

type baseVariable struct {
	vendorVar string
}

func (c *baseVariable) variableProperty() string {
	return canonicalizeToProperty(c.vendorVar)
}

type stringVariable struct {
	baseVariable
	values []string
}

func (s *stringVariable) variableValuesType() reflect.Type {
	var fields []reflect.StructField

	for _, v := range s.values {
		fields = append(fields, reflect.StructField{
			Name: proptools.FieldNameForProperty(v),
			Type: emptyInterfaceType,
		})
	}

	return reflect.StructOf(fields)
}

func (s *stringVariable) initializeProperties(v reflect.Value, typ reflect.Type) {
	for i := range s.values {
		v.Field(i).Set(reflect.Zero(typ))
	}
}

func (s *stringVariable) propertiesToApply(vendorConfig VendorConfig, values reflect.Value) interface{} {
	for j, v := range s.values {
		if vendorConfig.String(s.vendorVar) == v {
			return values.Field(j).Interface()
		}
	}

	return nil
}

type boolVariable struct {
	baseVariable
}

func (b boolVariable) variableValuesType() reflect.Type {
	return emptyInterfaceType
}

func (b boolVariable) initializeProperties(v reflect.Value, typ reflect.Type) {
	v.Set(reflect.Zero(typ))
}

func (b boolVariable) propertiesToApply(vendorConfig VendorConfig, values reflect.Value) interface{} {
	if vendorConfig.Bool(b.vendorVar) {
		return values.Interface()
	}

	return nil
}

func canonicalizeToProperty(v string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'A' && r <= 'Z',
			r >= 'a' && r <= 'z',
			r >= '0' && r <= '9',
			r == '_':
			return r
		default:
			return '_'
		}
	}, v)
}

func canonicalizeToProperties(values []string) []string {
	ret := make([]string, len(values))
	for i, v := range values {
		ret[i] = canonicalizeToProperty(v)
	}
	return ret
}

type emptyInterfaceStruct struct {
	i interface{}
}

var emptyInterfaceType = reflect.TypeOf(emptyInterfaceStruct{}).Field(0).Type
