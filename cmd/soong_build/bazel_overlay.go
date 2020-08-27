// Copyright 2020 Google Inc. All rights reserved.
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

package main

import (
	"android/soong/android"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/google/blueprint"
	"github.com/google/blueprint/bootstrap"
	"github.com/google/blueprint/bootstrap/bpdoc"
	"github.com/google/blueprint/proptools"
)

const (
	soongModuleLoad = `package(default_visibility = ["//visibility:public"])
load("//:soong_module.bzl", "soong_module")

`

	// A BUILD file target snippet representing a Soong module
	soongModuleTarget = `soong_module(
    name = "%s",
    module_name = "%s",
    module_type = "%s",
    module_variant = "%s",
    module_deps = %s,
%s)`

	providersBzl = `SoongModuleInfo = provider(
    fields = {
        "name": "Name of module",
        "type": "Type of module",
        "variant": "Variant of module",
    },
)
`

	// The soong_module rule implementation in a .bzl file
	soongModuleBzl = `
%s

load(":providers.bzl", "SoongModuleInfo")

def _generic_soong_module_impl(ctx):
    return [
        SoongModuleInfo(
            name = ctx.attr.module_name,
            type = ctx.attr.module_type,
            variant = ctx.attr.module_variant,
        ),
    ]

generic_soong_module = rule(
    implementation = _generic_soong_module_impl,
    attrs = {
    "module_name": attr.string(mandatory = True),
    "module_type": attr.string(mandatory = True),
    "module_variant": attr.string(),
    "module_deps": attr.label_list(providers = [SoongModuleInfo]),
    },
)

soong_module_rule_map = {
%s}

_SUPPORTED_TYPES = ["bool", "int", "string"]

# soong_module is a macro that supports arbitrary kwargs, and uses module_type to
# expand to the right underlying shim.
def soong_module(name, module_type, **kwargs):
    soong_module_rule = soong_module_rule_map.get(module_type)

    if soong_module_rule == None:
        # This module type does not have an existing rule to map to, so use the
        # generic_soong_module rule instead.
        generic_soong_module(
            name = name,
            module_type = module_type,
            module_name = kwargs.pop("module_name", ""),
            module_variant = kwargs.pop("module_variant", ""),
            module_deps = kwargs.pop("module_deps", []),
        )
    else:
        supported_kwargs = dict()
        for key, value in kwargs.items():
            if type(value) in _SUPPORTED_TYPES:
                supported_kwargs[key] = value
            elif type(value) == "list":
                supported = True
                for v in value:
                    supported = supported and type(v) in _SUPPORTED_TYPES
                if supported:
                    supported_kwargs[key] = value
            else:
                continue
        soong_module_rule(
            name = name,
            module_type = module_type,
            **supported_kwargs,
        )
`

	moduleRuleShim = `
def _%s_impl(ctx):
    return [SoongModuleInfo()]

%s = rule(
    implementation = _%s_impl,
    attrs = %s
)
`
)

func targetNameWithVariant(c *blueprint.Context, logicModule blueprint.Module) string {
	name := ""
	if c.ModuleSubDir(logicModule) != "" {
		// TODO(b/162720883): Figure out a way to drop the "--" variant suffixes.
		name = c.ModuleName(logicModule) + "--" + c.ModuleSubDir(logicModule)
	} else {
		name = c.ModuleName(logicModule)
	}

	return strings.Replace(name, "//", "", 1)
}

func qualifiedTargetLabel(c *blueprint.Context, logicModule blueprint.Module) string {
	return "//" +
		packagePath(c, logicModule) +
		":" +
		targetNameWithVariant(c, logicModule)
}

func packagePath(c *blueprint.Context, logicModule blueprint.Module) string {
	return filepath.Dir(c.BlueprintFile(logicModule))
}

func escapeString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	return strings.ReplaceAll(s, "\"", "\\\"")
}

func makeIndent(indent int) string {
	if indent < 0 {
		panic(fmt.Errorf("indent column cannot be less than 0, but got %d", indent))
	}
	return strings.Repeat("    ", indent)
}

// prettyPrint a property value into the equivalent Starlark representation
// recursively.
func prettyPrint(propertyValue reflect.Value, indent int) (string, error) {
	if isZero(propertyValue) {
		// A property value being set or unset actually matters -- Soong does set default
		// values for unset properties, like system_shared_libs = ["libc", "libm", "libdl"] at
		// https://cs.android.com/android/platform/superproject/+/master:build/soong/cc/linker.go;l=281-287;drc=f70926eef0b9b57faf04c17a1062ce50d209e480
		//
		// In Bazel-parlance, we would use "attr.<type>(default = <default value>)" to set the default
		// value of unset attributes.
		return "", nil
	}

	var ret string
	switch propertyValue.Kind() {
	case reflect.String:
		ret = fmt.Sprintf("\"%v\"", escapeString(propertyValue.String()))
	case reflect.Bool:
		ret = strings.Title(fmt.Sprintf("%v", propertyValue.Interface()))
	case reflect.Int, reflect.Uint, reflect.Int64:
		ret = fmt.Sprintf("%v", propertyValue.Interface())
	case reflect.Ptr:
		return prettyPrint(propertyValue.Elem(), indent)
	case reflect.Slice:
		ret = "[\n"
		for i := 0; i < propertyValue.Len(); i++ {
			indexedValue, err := prettyPrint(propertyValue.Index(i), indent+1)
			if err != nil {
				return "", err
			}

			if indexedValue != "" {
				ret += makeIndent(indent + 1)
				ret += indexedValue
				ret += ",\n"
			}
		}
		ret += makeIndent(indent)
		ret += "]"
	case reflect.Struct:
		ret = "{\n"
		// Sort and print the struct props by the key.
		structProps := extractStructProperties(propertyValue, indent)
		for _, k := range android.SortedStringKeys(structProps) {
			ret += makeIndent(indent + 1)
			ret += "\"" + k + "\": "
			ret += structProps[k]
			ret += ",\n"
		}
		ret += makeIndent(indent)
		ret += "}"
	case reflect.Interface:
		// TODO(b/164227191): implement pretty print for interfaces.
		// Interfaces are used for for arch, multilib and target properties.
		return "", nil
	default:
		return "", fmt.Errorf(
			"unexpected kind for property struct field: %s", propertyValue.Kind())
	}
	return ret, nil
}

func extractStructProperties(structValue reflect.Value, indent int) map[string]string {
	if structValue.Kind() != reflect.Struct {
		panic(fmt.Errorf("Expected a reflect.Struct type, but got %s", structValue.Kind()))
	}

	ret := map[string]string{}
	structType := structValue.Type()
	for i := 0; i < structValue.NumField(); i++ {
		field := structType.Field(i)
		if field.PkgPath != "" {
			// Skip unexported fields. Some properties are
			// internal to Soong only, and these fields do not have PkgPath.
			continue
		}
		if proptools.HasTag(field, "blueprint", "mutated") {
			continue
		}

		fieldValue := structValue.Field(i)
		if isZero(fieldValue) {
			// Ignore zero-valued fields
			continue
		}

		propertyName := proptools.PropertyNameForField(field.Name)
		prettyPrintedValue, err := prettyPrint(fieldValue, indent+1)
		if err != nil {
			panic(
				fmt.Errorf(
					"Error while parsing property: %q. %s",
					propertyName,
					err))
		}
		if prettyPrintedValue != "" {
			ret[propertyName] = prettyPrintedValue
		}
	}

	return ret
}

func isStructPtr(t reflect.Type) bool {
	return t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct
}

// Generically extract module properties and types into a map, keyed by the module property name.
func extractModuleProperties(aModule android.Module) map[string]string {
	ret := map[string]string{}

	// Iterate over this android.Module's property structs.
	for _, properties := range aModule.GetProperties() {
		propertiesValue := reflect.ValueOf(properties)
		// Check that propertiesValue is a pointer to the Properties struct, like
		// *cc.BaseLinkerProperties or *java.CompilerProperties.
		//
		// propertiesValue can also be type-asserted to the structs to
		// manipulate internal props, if needed.
		if isStructPtr(propertiesValue.Type()) {
			structValue := propertiesValue.Elem()
			for k, v := range extractStructProperties(structValue, 0) {
				ret[k] = v
			}
		} else {
			panic(fmt.Errorf(
				"properties must be a pointer to a struct, got %T",
				propertiesValue.Interface()))
		}

	}

	return ret
}

var allowedPropTypes map[string]bool = map[string]bool{
	"int":         true,
	"bool":        true,
	"string_list": true,
	"string":      true,
}

// TODO(b/166563303)
var additionalPropTypes map[string]map[string]string = map[string]map[string]string{
	"sdk": {
		"native_header_libs":  "string_list",
		"native_shared_libs":  "string_list",
		"native_static_libs":  "string_list",
		"native_objects":      "string_list",
		"native_libs":         "string_list",
		"java_sdk_libs":       "string_list",
		"java_header_libs":    "string_list",
		"java_system_modules": "string_list",
	},
	"module_exports": {
		"native_binaries":    "string_list",
		"java_libs":          "string_list",
		"java_tests":         "string_list",
		"native_shared_libs": "string_list",
	},
	"hidl_package_root": {
		"use_current": "bool",
	},
}

func canonicalizeModuleType(moduleName string) string {
	if strings.HasSuffix(moduleName, "_test") {
		return moduleName + "_"
	}

	return moduleName
}

// Create <module>.bzl containing Bazel rule shims for every module type available in Soong and
// user-specified Go plugins.
//
// This function reuses documentation generation APIs to ensure parity between modules-as-docs
// and modules-as-code, including the names and types of module properties.
func createModuleBzlShims(ctx *android.Context, bazelOverlayDir string) (map[string][]string, error) {
	bzlLoads := map[string][]string{}

	moduleTypeFactories := android.ModuleTypeFactories()
	bpModuleTypeFactories := make(map[string]reflect.Value)
	for moduleType, factory := range moduleTypeFactories {
		bpModuleTypeFactories[moduleType] = reflect.ValueOf(factory)
	}
	packages, err := bootstrap.ModuleTypeDocs(ctx.Context, bpModuleTypeFactories)
	if err != nil {
		return nil, err
	}

	var propToAttr func(prop bpdoc.Property, propName string) string
	propToAttr = func(prop bpdoc.Property, propName string) string {
		propName = strings.ReplaceAll(propName, ".", "__")
		starlarkAttrType := prop.Type

		// Canonicalize and normalize module property types to Bazel attribute types
		if starlarkAttrType == "list of strings" {
			starlarkAttrType = "string_list"
		} else if starlarkAttrType == "int64" {
			starlarkAttrType = "int"
		} else if starlarkAttrType == "" {
			var attr string
			for _, nestedProp := range prop.Properties {
				nestedAttr := propToAttr(nestedProp, propName+"__"+nestedProp.Name)
				if nestedAttr != "" {
					// FIXME(jingwen): Fix nested props resulting in too many attributes.
					// Still generate these for now, but comment them out.
					attr += "# " + nestedAttr
				}
			}
			return attr
		}

		if !allowedPropTypes[starlarkAttrType] || !shouldGenerateAttribute(propName) {
			return ""
		}

		return "        \"" + propName + "\": attr." + starlarkAttrType + "(),\n"
	}

	for _, pkg := range packages {
		packageBzl := `load(":providers.bzl", "SoongModuleInfo")
`
		bzlFileName := strings.ReplaceAll(pkg.Path, "/", "_")
		for _, moduleTypeTemplate := range moduleTypeDocsToTemplates(pkg.ModuleTypes) {
			attrs := `{
        "module_name": attr.string(mandatory = True),
        "module_type": attr.string(mandatory = True),
        "module_variant": attr.string(),
        "module_deps": attr.label_list(providers = [SoongModuleInfo]),
`
			for _, prop := range moduleTypeTemplate.Properties {
				attrs += propToAttr(prop, prop.Name)
			}

			for propName, propType := range additionalPropTypes[moduleTypeTemplate.Name] {
				attrs += "        \"" + propName + "\": attr." + propType + "(),\n"
			}

			attrs += "    },\n"

			moduleType := canonicalizeModuleType(moduleTypeTemplate.Name)
			packageBzl += fmt.Sprintf(moduleRuleShim, moduleType, moduleType, moduleType, attrs)

			bzlLoads[bzlFileName] = append(bzlLoads[bzlFileName], moduleType)
		}

		if err := writeReadOnlyFile(bazelOverlayDir, bzlFileName+".bzl", packageBzl); err != nil {
			return nil, err
		}
	}
	return bzlLoads, nil
}

func createBazelOverlay(ctx *android.Context, bazelOverlayDir string) error {
	blueprintCtx := ctx.Context
	blueprintCtx.VisitAllModules(func(module blueprint.Module) {
		buildFile, err := buildFileForModule(blueprintCtx, module)
		if err != nil {
			panic(err)
		}

		buildFile.Write([]byte(generateSoongModuleTarget(blueprintCtx, module) + "\n\n"))
		buildFile.Close()
	})

	if err := writeReadOnlyFile(bazelOverlayDir, "WORKSPACE", ""); err != nil {
		return err
	}

	if err := writeReadOnlyFile(bazelOverlayDir, "BUILD", ""); err != nil {
		return err
	}

	if err := writeReadOnlyFile(bazelOverlayDir, "providers.bzl", providersBzl); err != nil {
		return err
	}

	bzlLoads, err := createModuleBzlShims(ctx, bazelOverlayDir)
	if err != nil {
		return err
	}

	return writeReadOnlyFile(bazelOverlayDir, "soong_module.bzl", generateSoongModuleBzl(bzlLoads))
}

func generateSoongModuleBzl(bzlLoads map[string][]string) string {
	var loadStmts string
	var moduleRuleMap string
	for bzl, rules := range bzlLoads {
		loadStmt := "load(\"//:"
		loadStmt += bzl
		loadStmt += ".bzl\""
		for _, rule := range rules {
			loadStmt += ", \"" + rule + "\""
			moduleRuleMap += "    \"" + rule + "\": " + rule + ",\n"
		}
		loadStmt += ")\n"
		loadStmts += loadStmt
	}

	return fmt.Sprintf(soongModuleBzl, loadStmts, moduleRuleMap)
}

var ignoredPropNames map[string]bool = map[string]bool{
	"name":        true, // redundant, since this is explicitly generated for every target
	"from":        true, // reserved keyword
	"in":          true, // reserved keyword
	"arch":        true, // interface prop type is not supported yet.
	"multilib":    true, // interface prop type is not supported yet.
	"target":      true, // interface prop type is not supported yet.
	"visibility":  true, // Bazel has native visibility semantics. Handle later.
	"features":    true, // There is already a built-in attribute 'features' which cannot be overridden.
	"module_type": true, // FIXME: used by soong_config_module_type
}

func shouldGenerateAttribute(prop string) bool {
	return !(ignoredPropNames[prop] || strings.Contains(prop, "."))
}

// props is an unsorted map. This function ensures that
// the generated attributes are sorted to ensure determinism.
func propsToAttributes(props map[string]string) string {
	var attributes string
	for _, propName := range android.SortedStringKeys(props) {
		if shouldGenerateAttribute(propName) {
			attributes += fmt.Sprintf("    %s = %s,\n", propName, props[propName])
		}
	}
	return attributes
}

// Convert a module and its deps and props into a Bazel macro/rule
// representation in the BUILD file.
func generateSoongModuleTarget(
	blueprintCtx *blueprint.Context,
	module blueprint.Module) string {

	var props map[string]string
	if aModule, ok := module.(android.Module); ok {
		props = extractModuleProperties(aModule)
	}
	attributes := propsToAttributes(props)

	// TODO(b/163018919): DirectDeps can have duplicate (module, variant)
	// items, if the modules are added using different DependencyTag. Figure
	// out the implications of that.
	depLabels := map[string]bool{}
	blueprintCtx.VisitDirectDeps(module, func(depModule blueprint.Module) {
		depLabels[qualifiedTargetLabel(blueprintCtx, depModule)] = true
	})

	depLabelList := "[\n"
	for depLabel, _ := range depLabels {
		depLabelList += "        \""
		depLabelList += depLabel
		depLabelList += "\",\n"
	}
	depLabelList += "    ]"

	moduleType := canonicalizeModuleType(blueprintCtx.ModuleType(module))
	if strings.HasSuffix(moduleType, "_test") {
		moduleType += "_"
	}

	return fmt.Sprintf(
		soongModuleTarget,
		targetNameWithVariant(blueprintCtx, module),
		blueprintCtx.ModuleName(module),
		moduleType,
		blueprintCtx.ModuleSubDir(module),
		depLabelList,
		attributes)
}

func buildFileForModule(ctx *blueprint.Context, module blueprint.Module) (*os.File, error) {
	// Create nested directories for the BUILD file
	dirPath := filepath.Join(bazelOverlayDir, packagePath(ctx, module))
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		os.MkdirAll(dirPath, os.ModePerm)
	}
	// Open the file for appending, and create it if it doesn't exist
	f, err := os.OpenFile(
		filepath.Join(dirPath, "BUILD.bazel"),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644)
	if err != nil {
		return nil, err
	}

	// If the file is empty, add the load statement for the `soong_module` rule
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if fi.Size() == 0 {
		f.Write([]byte(soongModuleLoad + "\n"))
	}

	return f, nil
}

// The overlay directory should be read-only, sufficient for bazel query.
func writeReadOnlyFile(dir string, baseName string, content string) error {
	workspaceFile := filepath.Join(bazelOverlayDir, baseName)
	// 0444 is read-only
	return ioutil.WriteFile(workspaceFile, []byte(content), 0444)
}

func isZero(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.Func, reflect.Map, reflect.Slice:
		return value.IsNil()
	case reflect.Array:
		valueIsZero := true
		for i := 0; i < value.Len(); i++ {
			valueIsZero = valueIsZero && isZero(value.Index(i))
		}
		return valueIsZero
	case reflect.Struct:
		valueIsZero := true
		for i := 0; i < value.NumField(); i++ {
			if value.Field(i).CanSet() {
				valueIsZero = valueIsZero && isZero(value.Field(i))
			}
		}
		return valueIsZero
	case reflect.Ptr:
		if !value.IsNil() {
			return isZero(reflect.Indirect(value))
		} else {
			return true
		}
	default:
		zeroValue := reflect.Zero(value.Type())
		result := value.Interface() == zeroValue.Interface()
		return result
	}
}
