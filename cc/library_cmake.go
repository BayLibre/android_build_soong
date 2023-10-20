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

package cc

import (
	"android/soong/android"
	"bytes"
	"fmt"
	"github.com/google/blueprint"
	"strings"
	"text/template"
)

const cmakeContents = `cmake_minimum_required(VERSION 3.18)
project({{.Name}} CXX)
set(CMAKE_CXX_STANDARD 20)

{{range .Properties.Subdirectories -}}
add_subdirectory("${ANDROID_BUILD_TOP}/{{.Path}}" {{.Name}} EXCLUDE_FROM_ALL)
{{end}}
{{setList .Name "_CFLAGS" "" .Properties.Cflags}}

{{setList .Name "_LINKFLAGS" "" .Properties.Linkflags}}
`

const cmakeModule = `
{{- if .Srcs -}}
{{setList .M.Name "_SRC" "${ANDROID_BUILD_TOP}/" (toStrings .Srcs)}}

add_library({{.M.Name}} {{"${"}}{{.M.Name}}{{"_SRC}"}})
{{else -}}
add_library({{.M.Name}} INTERFACE)
{{end}}
{{setList .M.Name "_EXPINC" "${ANDROID_BUILD_TOP}/" (toStrings .ExpInc)}}

{{setList .M.Name "_INC" "${ANDROID_BUILD_TOP}/" .Inc}}

{{if .Srcs -}}
target_include_directories({{.M.Name}} PUBLIC {{"${"}}{{.M.Name}}{{"_EXPINC}"}} PRIVATE {{"${"}}{{.M.Name}}{{"_INC}"}})
target_compile_options({{.M.Name}} PRIVATE {{"${"}}{{.MT.Name}}{{"_CFLAGS}"}})
target_link_options({{.M.Name}} PRIVATE {{"${"}}{{.MT.Name}}{{"_LINKFLAGS}"}})
{{else -}}
target_include_directories({{.M.Name}} INTERFACE {{"${"}}{{.M.Name}}{{"_EXPINC}"}} {{"${"}}{{.M.Name}}{{"_INC}"}})
{{end}}
{{setList .M.Name "_WSDEP" "" .Wsdep}}

{{setList .M.Name "_SDEP" "" .Sdep}}

{{setList .M.Name "_DDEP" "" .Ddep}}

{{setList .M.Name "_HDEP" "" .Hdep}}

{{ if .Srcs -}}
target_link_libraries({{.M.Name}} {{"${"}}{{.M.Name}}{{"_WSDEP}"}} {{"${"}}{{.M.Name}}{{"_SDEP}"}} {{"${"}}{{.M.Name}}{{"_DDEP}"}} {{"${"}}{{.M.Name}}{{"_HDEP}"}})
{{else -}}
target_link_libraries({{.M.Name}} INTERFACE {{"${"}}{{.M.Name}}{{"_WSDEP}"}} {{"${"}}{{.M.Name}}{{"_SDEP}"}} {{"${"}}{{.M.Name}}{{"_DDEP}"}} {{"${"}}{{.M.Name}}{{"_HDEP}"}})
{{end -}}
`

func RegisterLibraryCmakeComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("cc_library_cmake_snapshot", LibraryCmakeSnapshotFactory)
}

var _ android.OutputFileProducer = (*LibraryCmakeSnapshot)(nil)

func RegisterPreDepsMutators(ctx android.RegisterMutatorsContext) {
	ctx.BottomUp("addCMakeLibs", addCMakeLibs).Parallel()
}

type LibraryCmakeSubdir struct {
	Name string
	Path string
}

type CmakeLibraryAlternative struct {
	Android_lib string
	Cmake_libs  []string
}

type LibraryCmakeSnapshotProperties struct {
	Cc_libs                  []string
	Cflags                   []string
	Linkflags                []string
	Subdirectories           []LibraryCmakeSubdir
	Android_libname_to_cmake []CmakeLibraryAlternative
}

type LibraryCmakeSnapshot struct {
	android.ModuleBase

	cmakeFilePath android.WritablePath
	Properties    LibraryCmakeSnapshotProperties
}

func (m *LibraryCmakeSnapshot) OutputFiles(tag string) (android.Paths, error) {
	if tag == "" {
		return android.Paths{m.cmakeFilePath}, nil
	}
	return nil, fmt.Errorf("unrecognized tag %q", tag)
}

func init() {
	RegisterLibraryCmakeComponents(android.InitRegistrationContext)
	android.PreDepsMutators(RegisterPreDepsMutators)
}

type LibraryCmakeSnapshotDepTag struct {
	blueprint.BaseDependencyTag
}

func addCMakeLibs(mctx android.BottomUpMutatorContext) {
	if m, ok := mctx.Module().(*LibraryCmakeSnapshot); ok {
		mctx.AddDependency(mctx.Module(), LibraryCmakeSnapshotDepTag{}, m.Properties.Cc_libs...)
	}
}

func (m *LibraryCmakeSnapshot) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	m.cmakeFilePath = android.PathForModuleOut(ctx, "CMakeLists.txt")

	funcMap := template.FuncMap{
		"setList": func(name string, nameSuffix string, itemPrefix string, items []string) (string, error) {
			var list strings.Builder
			list.WriteString("set(" + name + nameSuffix + "\n")
			for _, item := range items {
				list.WriteString("    " + itemPrefix + item + "\n")
			}
			list.WriteString(")")
			return list.String(), nil
		},
		"toStrings": func(files []android.Path) []string {
			strings := make([]string, len(files))
			for idx, file := range files {
				strings[idx] = file.String()
			}
			return strings
		},
	}

	var fragments []string
	modules := make(map[string]bool)

	var templateBuffer bytes.Buffer
	cmakeContentsCompiled := template.Must(template.New("").Funcs(funcMap).Parse(cmakeContents))
	if err := cmakeContentsCompiled.Execute(&templateBuffer, m); err != nil {
		panic(err)
	}
	fragments = append(fragments, templateBuffer.String())
	templateBuffer.Reset()

	var android_lib_to_cmake = make(map[string][]string)
	for _, a2c := range m.Properties.Android_libname_to_cmake {
		android_lib_to_cmake[a2c.Android_lib] = a2c.Cmake_libs
	}

	cmakeModuleCompiled := template.Must(template.New("").Funcs(funcMap).Parse(cmakeModule))

	// TODO: ignored system libs
	modules["libc++"] = true
	modules["libc++_static"] = true
	modules["prebuilt_libclang_rt.builtins"] = true
	modules["prebuilt_libclang_rt.ubsan_minimal"] = true

	// TODO: integrate with android_libname_to_cmake
	modules["libbinder"] = true // TODO: binderRpcTestIface-cpp -> libbinder
	modules["libcutils"] = true // TODO: binderRpcTestIface-cpp -> libcutils
	modules["libutils"] = true  // TODO: binderRpcTestIface-cpp -> libutils
	modules["fmtlib"] = true
	modules["libcrypto"] = true
	modules["libssl"] = true
	modules["liblog"] = true // TODO: binderRpcTestIface-cpp -> libbinder -> ... -> liblog

	// TODO: not sure what
	modules["jni_headers"] = true

	ctx.WalkDeps(func(dep android.Module, parent android.Module) bool {
		if mdep, ok := dep.(*Module); ok {
			if _, ok := modules[mdep.Name()]; ok {
				return false
			}
			modules[mdep.Name()] = true
			fmt.Println("WalkDeps: " + dep.Name() + " (" + parent.Name() + ")")

			type ModuleTemplateInput struct {
				M      *Module
				MT     *LibraryCmakeSnapshot
				Srcs   []android.Path
				ExpInc []android.Path
				Inc    []string // why isn't it []android.Path?
				Wsdep  []string
				Sdep   []string
				Ddep   []string
				Hdep   []string
			}
			depExporterInfo, _ := android.OtherModuleProvider(ctx, dep, FlagExporterInfoProvider)
			baseCompilerProperties := mdep.compiler.(*libraryDecorator).baseCompiler.Properties
			baseLinkerProperties := mdep.linker.(*libraryDecorator).baseLinker.Properties
			var input = ModuleTemplateInput{
				mdep,
				m,
				mdep.compiler.(CompiledInterface).Srcs(),
				depExporterInfo.IncludeDirs,
				append(
					baseCompilerProperties.Include_dirs,
					baseCompilerProperties.Local_include_dirs...),
				AndroidLibListToCmake(baseLinkerProperties.Whole_static_libs, android_lib_to_cmake),
				AndroidLibListToCmake(baseLinkerProperties.Static_libs, android_lib_to_cmake),
				AndroidLibListToCmake(baseLinkerProperties.Shared_libs, android_lib_to_cmake),
				AndroidLibListToCmake(baseLinkerProperties.Header_libs, android_lib_to_cmake),
			}

			if err := cmakeModuleCompiled.Execute(&templateBuffer, input); err != nil {
				panic(err)
			}
			fragments = append(fragments, templateBuffer.String())
			templateBuffer.Reset()

			return true
		}
		return false
	})

	android.WriteFileRule(ctx, m.cmakeFilePath, strings.Join(fragments, "\n"))
}

func (m *LibraryCmakeSnapshot) AndroidMkEntries() []android.AndroidMkEntries {
	return []android.AndroidMkEntries{{
		Class:      "ETC",
		OutputFile: android.OptionalPathForPath(m.cmakeFilePath),
		ExtraEntries: []android.AndroidMkExtraEntriesFunc{
			func(ctx android.AndroidMkExtraEntriesContext, entries *android.AndroidMkEntries) {
				entries.SetBool("LOCAL_UNINSTALLABLE_MODULE", true)
			},
		},
	}}
}

func AndroidLibListToCmake(alibs []string, a2c_map map[string][]string) []string {
	var cmake_lib_list []string
	for _, alib := range alibs {
		clibs, exists := a2c_map[alib]
		if exists {
			cmake_lib_list = append(cmake_lib_list, clibs...)
		} else {
			cmake_lib_list = append(cmake_lib_list, alib)
		}
	}
	return cmake_lib_list
}

func LibraryCmakeSnapshotFactory() android.Module {
	module := &LibraryCmakeSnapshot{}
	module.AddProperties(&module.Properties)
	android.InitAndroidArchModule(module, android.HostSupported, android.MultilibFirst)
	return module
}
