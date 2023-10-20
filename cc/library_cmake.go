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
	"regexp"
	"strings"
	"text/template"
)

const cmakeMain = `cmake_minimum_required(VERSION 3.18)
project({{.M.Name}} CXX)
set(CMAKE_CXX_STANDARD 20)

{{range .M.Properties.Subdirectories -}}
add_subdirectory("${ANDROID_BUILD_TOP}/{{.Path}}" {{.Name}} EXCLUDE_FROM_ALL)
{{end}}
{{setList .M.Name "_CFLAGS" "" .M.Properties.Cflags}}

{{setList .M.Name "_LINKFLAGS" "" .M.Properties.Linkflags}}

{{range $moduleDir, $value := .ModuleDirs -}}
add_subdirectory({{$moduleDir}})
{{end}}
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

const aidlModule = `if (NOT AIDL_BIN)
    find_program(AIDL_BIN aidl REQUIRED)
endif()
{{setList .M.Name "_SRC" "${ANDROID_BUILD_TOP}/" (toStrings .Srcs)}}

set({{.M.Name}}_GENDIR "{{"${PROJECT_BINARY_DIR}"}}/gens")
set({{.M.Name}}_GEN_SRC)
foreach(AIDL_FILE {{print "${" .M.Name "_SRC}"}})
    get_filename_component(ABS_AIDL_FILEPATH {{"${AIDL_FILE}"}} ABSOLUTE)
    get_filename_component(ABS_AIDL_DIR {{"${ABS_AIDL_FILEPATH}"}} DIRECTORY)
    get_filename_component(AIDL_FILENAME {{"${AIDL_FILE}"}} NAME)
    get_filename_component(AIDL_FILENAME_WE {{"${AIDL_FILE}"}} NAME_WE)
    set(AIDL_CPP_FILE "{{print "${" .M.Name "_GENDIR}"}}/{{"${AIDL_FILENAME_WE}"}}.cpp")
    add_custom_command(
        OUTPUT "{{"${AIDL_CPP_FILE}"}}"
        COMMAND "{{"${AIDL_BIN}"}}"
        ARGS
        --lang=cpp
        -Weverything
        -Wno-missing-permission-annotation
        -Wno-mixed-oneway
        --min_sdk_version current
        --structured
        --ninja
        --trace
        -d "{{print "${" .M.Name "_GENDIR}"}}/${AIDL_FILENAME_WE}.cpp.d"
        -h "{{print "${" .M.Name "_GENDIR}"}}/include"
        -o "{{print "${" .M.Name "_GENDIR}"}}"
        -I "{{"${ABS_AIDL_DIR}"}}"
        "{{"${ABS_AIDL_FILEPATH}"}}"
    )
list(APPEND {{.M.Name}}_GEN_SRC "{{"${AIDL_CPP_FILE}"}}")
endforeach()

add_library({{.M.Name}} {{print "${" .M.Name "_GEN_SRC}"}})
target_compile_options({{.M.Name}}
    PRIVATE
    {{print "${" .MT.Name "_CFLAGS}"}}
)
target_include_directories({{.M.Name}}
    PUBLIC
    "{{print "${" .M.Name "_GENDIR}"}}/include"
    "{{"${ANDROID_BUILD_TOP}"}}/system/libbase/include"
    "{{"${ANDROID_BUILD_TOP}"}}/external/fmtlib/include"
    "{{"${ANDROID_BUILD_TOP}"}}/frameworks/native/libs/binder/include"
    "{{"${ANDROID_BUILD_TOP}"}}/frameworks/native/libs/binder/ndk/include_cpp"
    "{{"${ANDROID_BUILD_TOP}"}}/system/core/libcutils/include"
    "{{"${ANDROID_BUILD_TOP}"}}/system/core/libutils/include"
    "{{"${ANDROID_BUILD_TOP}"}}/system/logging/liblog/include"
    "{{"${ANDROID_BUILD_TOP}"}}/system/core/libprocessgroup/include"
    "{{"${ANDROID_BUILD_TOP}"}}/system/core/libsystem/include"
    "{{"${ANDROID_BUILD_TOP}"}}/system/core/libutils/binder/include"
)
target_link_options({{.M.Name}} PRIVATE {{print "${" .MT.Name "_LINKFLAGS}"}})
target_link_libraries({{.M.Name}}
    libbinder
    libutils
    libcutils
)
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

	zipPath    android.WritablePath
	Properties LibraryCmakeSnapshotProperties
}

func (m *LibraryCmakeSnapshot) OutputFiles(tag string) (android.Paths, error) {
	if tag == "" {
		return android.Paths{m.zipPath}, nil
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
	zipList := android.Paths{}
	var templateBuffer bytes.Buffer

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

	// Generating CMakeLists.txt for all modules in dependency tree

	var android_lib_to_cmake = make(map[string][]string)
	for _, a2c := range m.Properties.Android_libname_to_cmake {
		android_lib_to_cmake[a2c.Android_lib] = a2c.Cmake_libs
	}

	cmakeModuleCompiled := template.Must(template.New("").Funcs(funcMap).Parse(cmakeModule))
	aidlModuleCompiled := template.Must(template.New("").Funcs(funcMap).Parse(aidlModule))

	modules := make(map[string]bool)

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

	moduleDirs := make(map[string][]string)

	ctx.WalkDeps(func(dep android.Module, parent android.Module) bool {
		if mdep, ok := dep.(*Module); ok {
			if _, ok := modules[mdep.Name()]; ok {
				return false
			}
			modules[mdep.Name()] = true

			// poor man's AIDL module detection
			isAidl := strings.HasSuffix(mdep.Name(), "-cpp")

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

			templateToUse := cmakeModuleCompiled
			if isAidl {
				templateToUse = aidlModuleCompiled
				input.Srcs = GuessAidlSourceList(ctx, input.Srcs)
			}
			if err := templateToUse.Execute(&templateBuffer, input); err != nil {
				panic(err)
			}
			moduleFragment := templateBuffer.String()
			templateBuffer.Reset()

			moduleDir := ctx.OtherModuleDir(dep)
			if _, ok := moduleDirs[moduleDir]; !ok { // TODO: can we just skip creating 0-len slice?
				moduleDirs[moduleDir] = make([]string, 0)
			}
			moduleDirs[moduleDir] = append(moduleDirs[moduleDir], moduleFragment)

			return true
		}
		return false
	})

	for moduleDir, fragments := range moduleDirs {
		moduleCmakePath := android.PathForModuleOut(ctx, moduleDir+"/CMakeLists.txt") // TODO: path append?
		zipList = append(zipList, moduleCmakePath)
		android.WriteFileRule(ctx, moduleCmakePath, strings.Join(fragments, "\n"))
	}

	// Generating main CMakeLists.txt

	type MainCmakeTemplateInput struct {
		M          *LibraryCmakeSnapshot
		ModuleDirs map[string][]string
	}
	input := MainCmakeTemplateInput{
		m,
		moduleDirs,
	}

	mainCmakePath := android.PathForModuleOut(ctx, "CMakeLists.txt")
	zipList = append(zipList, mainCmakePath)
	cmakeMainCompiled := template.Must(template.New("").Funcs(funcMap).Parse(cmakeMain))
	if err := cmakeMainCompiled.Execute(&templateBuffer, input); err != nil {
		panic(err)
	}
	android.WriteFileRule(ctx, mainCmakePath, templateBuffer.String())
	templateBuffer.Reset()

	// Packaging all CMakeLists.txt into a single zip file

	m.zipPath = android.PathForModuleOut(ctx, m.Name()+".zip")
	zipRule := android.NewRuleBuilder(pctx, ctx)
	rspFile := android.PathForModuleOut(ctx, m.Name()+"_list.rsp")
	zipRule.Command().
		BuiltTool("soong_zip").
		FlagWithOutput("-o ", m.zipPath).
		FlagWithArg("-C ", android.PathForModuleOut(ctx).OutputPath.String()).
		FlagWithRspFileInputList("-r ", rspFile, zipList)
	zipRule.Build(m.zipPath.String(), "archiving "+m.Name())
}

func (m *LibraryCmakeSnapshot) AndroidMkEntries() []android.AndroidMkEntries {
	return []android.AndroidMkEntries{{
		Class:      "ETC",
		OutputFile: android.OptionalPathForPath(m.zipPath),
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

// poor man's fetching AIDL source list from AIDL generated cpp source list
func GuessAidlSourceList(ctx android.ModuleContext, genlist []android.Path) []android.Path {
	m := regexp.MustCompile("^out/soong/.intermediates/(.*)/[^/]+-cpp-source/gen/([^/]+).cpp$")
	replaceto := "${1}/${2}.aidl"
	for i, _ := range genlist {
		genlist[i] = android.PathForSource(ctx, m.ReplaceAllString(genlist[i].String(), replaceto))
	}
	return genlist
}

func LibraryCmakeSnapshotFactory() android.Module {
	module := &LibraryCmakeSnapshot{}
	module.AddProperties(&module.Properties)
	android.InitAndroidArchModule(module, android.HostSupported, android.MultilibFirst)
	return module
}
