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

import _ "embed"

//go:embed library_cmake_main.txt
var cmakeMain string

//go:embed library_cmake_module_cc.txt
var cmakeModule string

//go:embed library_cmake_module_aidl.txt
var aidlModule string

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

	cmakeModuleCompiled := template.Must(template.New("").Delims("<<", ">>").Funcs(funcMap).Parse(cmakeModule))
	aidlModuleCompiled := template.Must(template.New("").Delims("<<", ">>").Funcs(funcMap).Parse(aidlModule))

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
	cmakeMainCompiled := template.Must(template.New("").Delims("<<", ">>").Funcs(funcMap).Parse(cmakeMain))
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
