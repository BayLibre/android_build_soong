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
    deps = %s,
%s)

`

	// The soong_module rule implementation in a .bzl file
	soongModuleBzl = `SoongModuleInfo = provider(
    fields = {
        "name": "Name of module",
        "type": "Type of module",
        "variant": "Variant of module",
    },
)

def _add_dicts(*dicts):
    """Adds a list of dictionaries into a single dictionary."""

    # If keys are repeated in multiple dictionaries, the latter one "wins".
    result = {}
    for d in dicts:
        result.update(d)

    return result

def _generic_soong_module_impl(ctx):
    return [
        SoongModuleInfo(
            name = ctx.attr.module_name,
            type = ctx.attr.module_type,
            variant = ctx.attr.module_variant,
        ),
    ]

_COMMON_ATTRS = {
    "module_name": attr.string(mandatory = True),
    "module_type": attr.string(mandatory = True),
    "module_variant": attr.string(),
    "deps": attr.label_list(providers = [SoongModuleInfo]),
}


generic_soong_module = rule(
    implementation = _generic_soong_module_impl,
    attrs = _COMMON_ATTRS,
)

# TODO(jingwen): auto generate Soong module shims
def _soong_filegroup_impl(ctx):
    return [SoongModuleInfo(),]

soong_filegroup = rule(
    implementation = _soong_filegroup_impl,
    # Matches https://cs.android.com/android/platform/superproject/+/master:build/soong/android/filegroup.go;l=25-40;drc=6a6478d49e78703ba22a432c41d819c8df79ef6c
    attrs = _add_dicts(_COMMON_ATTRS, {
        "srcs": attr.string_list(doc = "srcs lists files that will be included in this filegroup"),
        "exclude_srcs": attr.string_list(),
        "path": attr.string(doc = "The base path to the files. May be used by other modules to determine which portion of the path to use. For example, when a filegroup is used as data in a cc_test rule, the base path is stripped off the path and the remaining path is used as the installation directory."),
        "export_to_make_var": attr.string(doc = "Create a make variable with the specified name that contains the list of files in the filegroup, relative to the root of the source tree."),
    })
)

soong_module_rule_map = {
    "filegroup": soong_filegroup,
}

# soong_module is a macro that supports arbitrary kwargs, and uses module_type to
# expand to the right underlying shim.
def soong_module(name, module_type, **kwargs):
    soong_module_rule = soong_module_rule_map.get(module_type)

    if soong_module_rule == None:
        generic_soong_module(
            name = name,
            module_type = module_type,
            module_name = kwargs.pop("module_name", ""),
            module_variant = kwargs.pop("module_variant", ""),
            deps = kwargs.pop("deps", []),
        )
    else:
        soong_module_rule(
            name = name,
            module_type = module_type,
            **kwargs,
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
	s = strings.Replace(s, "\\", "\\\\", -1)
	return strings.Replace(s, "\"", "\\\"", -1)
}

// prettyPrint a property value into the equivalent Starlark representation
// recursively.
func prettyPrint(propertyValue reflect.Value) (string, error) {
	if isZero(propertyValue) {
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
		return prettyPrint(propertyValue.Elem())
	case reflect.Slice:
		ret = "[\n"
		for i := 0; i < propertyValue.Len(); i++ {
			indexedValue, err := prettyPrint(propertyValue.Index(i))
			if err != nil {
				return "", err
			}

			if indexedValue != "" {
				ret += "        "
				ret += indexedValue
				ret += ",\n"
			}
		}
		ret += "    ]"
	case reflect.Struct:
		ret = "{\n"
		for k, v := range extractStructProperties(propertyValue) {
			ret += "        "
			ret += "\"" + k + "\": "
			ret += v
			ret += ",\n"
		}
		ret += "    }"
	case reflect.Interface:
		// TODO(jin): implement pretty print for interfaces.
		// Interfaces are used for for arch, multilib and target properties.
		return "", nil
	default:
		return "", fmt.Errorf(
			"unexpected kind for property struct field: %s", propertyValue.Kind())
	}
	return ret, nil
}

func extractStructProperties(structValue reflect.Value) map[string]string {
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
		prettyPrintedValue, err := prettyPrint(fieldValue)
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
			for k, v := range extractStructProperties(structValue) {
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

func createBazelOverlay(ctx *android.Context, bazelOverlayDir string) error {
	blueprintCtx := ctx.Context
	blueprintCtx.VisitAllModules(func(module blueprint.Module) {
		buildFile, err := buildFileForModule(blueprintCtx, module)
		if err != nil {
			panic(err)
		}

		// TODO(b/163018919): DirectDeps can have duplicate (module, variant)
		// items, if the modules are added using different DependencyTag. Figure
		// out the implications of that.
		depLabels := map[string]bool{}
		blueprintCtx.VisitDirectDeps(module, func(depModule blueprint.Module) {
			depLabels[qualifiedTargetLabel(blueprintCtx, depModule)] = true
		})

		var props map[string]string
		if aModule, ok := module.(android.Module); ok {
			props = extractModuleProperties(aModule)
		}
		buildFile.Write([]byte(
			generateSoongModuleTarget(blueprintCtx, module, depLabels, props)))
		buildFile.Close()
	})

	if err := writeReadOnlyFile(bazelOverlayDir, "WORKSPACE", ""); err != nil {
		return err
	}

	if err := writeReadOnlyFile(bazelOverlayDir, "BUILD", ""); err != nil {
		return err
	}

	return writeReadOnlyFile(bazelOverlayDir, "soong_module.bzl", soongModuleBzl)
}

var ignoredProps map[string]bool = map[string]bool{
	"name":       true, // redundant, since this is explicitly generated for every target
	"from":       true, // reserved keyword
	"in":         true, // reserved keyword
	"arch":       true, // interface prop type is not supported yet.
	"multilib":   true, // interface prop type is not supported yet.
	"target":     true, // interface prop type is not supported yet.
	"dist":       true, // interface prop type is not supported yet.
	"dists":      true, // interface prop type is not supported yet.
	"visibility": true, // Bazel has native visibility semantics. Handle later.
}

func shouldGenerateAttribute(prop string) bool {
	return !ignoredProps[prop]
}

// Convert a module and its deps and props into a Bazel macro/rule
// representation in the BUILD file.
func generateSoongModuleTarget(
	blueprintCtx *blueprint.Context,
	module blueprint.Module,
	depLabels map[string]bool,
	props map[string]string) string {

	var attributes string
	for k, v := range props {
		if shouldGenerateAttribute(k) {
			attributes += fmt.Sprintf("    %s = %s,\n", k, v)
		}
	}

	depLabelList := "[\n"
	for depLabel, _ := range depLabels {
		depLabelList += "        \""
		depLabelList += depLabel
		depLabelList += "\",\n"
	}
	depLabelList += "    ]"

	return fmt.Sprintf(
		soongModuleTarget,
		targetNameWithVariant(blueprintCtx, module),
		blueprintCtx.ModuleName(module),
		blueprintCtx.ModuleType(module),
		// misleading name, this actually returns the variant.
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
