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
    deps = [
        %s
    ],
)
`

	// The soong_module rule implementation in a .bzl file
	soongModuleBzl = `
SoongModuleInfo = provider(
    fields = {
        "name": "Name of module",
        "type": "Type of module",
        "variant": "Variant of module",
    },
)

def _soong_module_impl(ctx):
    return [
        # SoongModuleInfo(
        #     name = ctx.attr.module_name,
        #     type = ctx.attr.module_type,
        #     variant = ctx.attr.module_variant,
        # ),
    ]

soong_module = rule(
    implementation = _soong_module_impl,
    attrs = {
        "module_name": attr.string(mandatory = True),
        "module_type": attr.string(mandatory = True),
        "module_variant": attr.string(),
        # "deps": attr.label_list(providers = [SoongModuleInfo]),
        "deps": attr.label_list(),
    },
)
`

	// TODO(jingwen): what is the equivalent of the 'path' module property?
	// no 'deps' in soong filegroups
	filegroupTarget = `filegroup(
    name = "%s",
    srcs = [
%s
    ],
)
`
)

func targetNameWithVariant(c *blueprint.Context, logicModule blueprint.Module) string {
	name := ""
	if c.ModuleSubDir(logicModule) != "" {
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

func printGeneralModuleInfo(aModule android.Module) {
	fmt.Println(aModule.Name())
	// fmt.Println(aModule.Enabled())
	// fmt.Println(aModule.Target().Arch.ArchType.String())
}

// A generic way to print module properties
func extractModuleProperties(aModule android.Module) map[string]string {
	ret := map[string]string{}

	for _, properties := range aModule.GetProperties() {
		propertiesValue := reflect.ValueOf(properties)
		if !isStructPtr(propertiesValue.Type()) {
			panic(fmt.Errorf("properties must be a pointer to a struct, got %T",
				propertiesValue.Interface()))
		}

		propertiesValue = propertiesValue.Elem()
		t := propertiesValue.Type()
		fmt.Println(t.Name()) // print the struct type of the properties struct

		for i := 0; i < propertiesValue.NumField(); i++ {
			f := propertiesValue.Field(i)
			propertyName := proptools.PropertyNameForField(
				propertiesValue.Type().Field(i).Name)

			// Ignore zero-valued properties
			if !isZero(f) {
				var propertyValue reflect.Value
				if f.Kind() == reflect.Ptr {
					propertyValue = reflect.Indirect(f)
				} else if f.Kind() == reflect.Interface {
					continue
					// if f.Elem().Kind() == reflect.Ptr {
					// 	propertyValue = reflect.Indirect(f.Elem())
					// 	_f := reflect.Indirect(f.Elem())
					// 	for j := 0; j < _f.NumField(); j++ {
					// 		if !isZero(_f.Field(j)) {
					// 			fmt.Println(_f.Field(j))
					// 		}
					// 	}
					// }
				} else {
					propertyValue = f
				}

				var value string
				if propertyValue.Kind() == reflect.Slice {
					for i := 0; i < propertyValue.Len(); i++ {
						value += "        "
						v := fmt.Sprintf("\"%s\",", propertyValue.Index(i))
						// TODO: skip strings with slashes in them because
						// some src files can cross package boundaries (!!)
						if strings.Contains(v, "/") {
							value += "# "
						}
						value += v
						value += "\n"
					}
				} else {
					value = fmt.Sprintf("%v", propertyValue.Interface())
				}

				ret[propertyName] = value

				fmt.Printf("%d: %s %s = %+v\n", i,
					f.Type(),
					propertyName,
					propertyValue)
			}
		}

		// switch properties.(type) {
		// case *cc.BaseLinkerProperties:
		// 	_ = properties.(*cc.BaseLinkerProperties)
		// }

		// switch properties.(type) {
		// case *java.CompilerProperties:
		// 	compilerProperties := properties.(*java.CompilerProperties)
		// 	fmt.Printf("srcs = %q\n", compilerProperties.Srcs)
		// default:
		// 	// do nothing
		// }
		// fmt.Println(t, propertiesValue)
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

		var depLabelList string
		for depLabel, _ := range depLabels {
			depLabelList += "\"" + depLabel + "\",\n        "
		}

		var target string
		target = genericSoongModule(blueprintCtx, module, depLabelList)

		if aModule, ok := module.(android.Module); ok {
			switch aModule.(type) {
			// case *genrule.Module:
			// 	printGeneralModuleInfo(aModule)
			// 	printModuleProperties(aModule)
			case *android.FileGroup:
				printGeneralModuleInfo(aModule)
				props := extractModuleProperties(aModule)
				target = fmt.Sprintf(
					filegroupTarget,
					aModule.Name(),
					props["srcs"])
				// props["name"] = blueprintCtx.ModuleName(module)
				// var tpl bytes.Buffer
				// err := template.Must(
				// 	template.New("").Parse(
				// 		filegroupTargetTemplate)).Execute(&tpl, props)
				// if err != nil {
				// 	panic(err)
				// }
				// target = tpl.String()
			// case *cc.Module:
			// 	if aModule.Name() == "libm" {
			// 		printModuleProperties(aModule)
			// 	}
			// case *java.Library:
			// 	javaLibraryModule := aModule.(*java.Library)
			// 	fmt.Printf("name = %s\n", javaLibraryModule.Name())
			// 	for _, properties := range javaLibraryModule.GetProperties() {
			// 		switch properties.(type) {
			// 		case *java.CompilerProperties:
			// 			compilerProperties := properties.(*java.CompilerProperties)
			// 			fmt.Printf("srcs = %q\n", compilerProperties.Srcs)
			// 		default:
			// 			// do nothing
			// 		}
			// 	}
			default:
				// generic android module
				// do nothing
			}
		}
		buildFile.Write([]byte(target))
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

func genericSoongModule(blueprintCtx *blueprint.Context, module blueprint.Module, depLabelList string) string {
	return fmt.Sprintf(
		soongModuleTarget,
		targetNameWithVariant(blueprintCtx, module),
		blueprintCtx.ModuleName(module),
		blueprintCtx.ModuleType(module),
		// misleading name, this actually returns the variant.
		blueprintCtx.ModuleSubDir(module),
		depLabelList)
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

func writeDepFile() {

}

func isStructPtr(t reflect.Type) bool {
	return t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct
}

func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Func, reflect.Map, reflect.Slice:
		return v.IsNil()
	case reflect.Array:
		z := true
		for i := 0; i < v.Len(); i++ {
			z = z && isZero(v.Index(i))
		}
		return z
	case reflect.Struct:
		z := true
		for i := 0; i < v.NumField(); i++ {
			if v.Field(i).CanSet() {
				z = z && isZero(v.Field(i))
			}
		}
		return z
	case reflect.Ptr:
		if !v.IsNil() {
			return isZero(reflect.Indirect(v))
		} else {
			return true
		}
	}
	// Compare other types directly:
	z := reflect.Zero(v.Type())
	result := v.Interface() == z.Interface()

	return result
}
