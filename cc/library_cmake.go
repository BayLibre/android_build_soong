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
	"text/template"
)

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
	properties    LibraryCmakeSnapshotProperties
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
		mctx.AddDependency(mctx.Module(), LibraryCmakeSnapshotDepTag{}, m.properties.Cc_libs...)
	}
}

func (m *LibraryCmakeSnapshot) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	m.cmakeFilePath = android.PathForModuleOut(ctx, "CMakeLists.txt")

	// TODO: replace cmakeContents generation with templates
	cmakeContents := `cmake_minimum_required(VERSION 3.18)
project(` + m.Name() + ` CXX)
set(CMAKE_CXX_STANDARD 20)

`

	for _, subdir := range m.properties.Subdirectories {
		cmakeContents += fmt.Sprintf("add_subdirectory(\"${ANDROID_BUILD_TOP}/%s\" %s EXCLUDE_FROM_ALL)\n\n", subdir.Path, subdir.Name)
	}

	cmakeContents += CmakeCreateList2(fmt.Sprintf("%s_CFLAGS", m.Name()), m.properties.Cflags, false)
	cmakeContents += CmakeCreateList2(fmt.Sprintf("%s_LINKFLAGS", m.Name()), m.properties.Linkflags, false)

	var android_lib_to_cmake = make(map[string][]string)
	for _, a2c := range m.properties.Android_libname_to_cmake {
		android_lib_to_cmake[a2c.Android_lib] = a2c.Cmake_libs
	}

	// TODO: use WalkDeps instead
	ctx.VisitDirectDeps(func(dep android.Module) {
		if mdep, ok := dep.(*Module); ok {
			is_interface := len(mdep.compiler.(CompiledInterface).Srcs()) == 0
			if is_interface {
				cmakeContents += fmt.Sprintf("add_library(%s INTERFACE)\n", mdep.Name())
			} else {
				var srcs_str []string
				for _, src := range mdep.compiler.(CompiledInterface).Srcs() {
					srcs_str = append(srcs_str, src.String())
				}
				cmakeContents += CmakeCreateList2(fmt.Sprintf("%s_SRC", mdep.Name()), srcs_str, true)
				cmakeContents += fmt.Sprintf("add_library(%s ${%s_SRC})\n", mdep.Name(), mdep.Name())
			}
			cmakeContents += "\n"

			// TODO: fail if not ok?
			depExporterInfo, _ := android.OtherModuleProvider(ctx, dep, FlagExporterInfoProvider)
			var exp_inc_str []string
			for _, exp_inc := range depExporterInfo.IncludeDirs {
				exp_inc_str = append(exp_inc_str, exp_inc.String())
			}

			cmakeContents += CmakeCreateList2(fmt.Sprintf("%s_EXPINC", mdep.Name()), exp_inc_str, true)
			cmakeContents += CmakeCreateList2(fmt.Sprintf("%s_INC", mdep.Name()),
				append(mdep.compiler.(*libraryDecorator).baseCompiler.Properties.Include_dirs, mdep.compiler.(*libraryDecorator).baseCompiler.Properties.Local_include_dirs...), true)
			if is_interface {
				cmakeContents += fmt.Sprintf("target_include_directories(%s INTERFACE ${%s_EXPINC} ${%s_INC})\n", mdep.Name(), mdep.Name(), mdep.Name())
			} else {
				cmakeContents += fmt.Sprintf("target_include_directories(%s PUBLIC ${%s_EXPINC} PRIVATE ${%s_INC})\n", mdep.Name(), mdep.Name(), mdep.Name())
				cmakeContents += fmt.Sprintf("target_compile_options(%s PRIVATE ${%s_CFLAGS})\n", mdep.Name(), m.Name())
				cmakeContents += fmt.Sprintf("target_link_options(%s PRIVATE ${%s_LINKFLAGS})\n", mdep.Name(), m.Name())
			}
			cmakeContents += "\n"

			cmakeContents += CmakeCreateList2(fmt.Sprintf("%s_WSDEP", mdep.Name()),
				AndroidLibListToCmake(mdep.linker.(*libraryDecorator).baseLinker.Properties.Whole_static_libs, android_lib_to_cmake), false)
			cmakeContents += CmakeCreateList2(fmt.Sprintf("%s_SDEP", mdep.Name()),
				AndroidLibListToCmake(mdep.linker.(*libraryDecorator).baseLinker.Properties.Static_libs, android_lib_to_cmake), false)
			cmakeContents += CmakeCreateList2(fmt.Sprintf("%s_DDEP", mdep.Name()),
				AndroidLibListToCmake(mdep.linker.(*libraryDecorator).baseLinker.Properties.Shared_libs, android_lib_to_cmake), false)
			cmakeContents += CmakeCreateList2(fmt.Sprintf("%s_HDEP", mdep.Name()),
				AndroidLibListToCmake(mdep.linker.(*libraryDecorator).baseLinker.Properties.Header_libs, android_lib_to_cmake), false)

			if is_interface {
				cmakeContents += fmt.Sprintf("target_link_libraries(%s INTERFACE ${%s_WSDEP} ${%s_SDEP} ${%s_DDEP} ${%s_HDEP})\n",
					mdep.Name(), mdep.Name(), mdep.Name(), mdep.Name(), mdep.Name())
			} else {
				cmakeContents += fmt.Sprintf("target_link_libraries(%s ${%s_WSDEP} ${%s_SDEP} ${%s_DDEP} ${%s_HDEP})\n",
					mdep.Name(), mdep.Name(), mdep.Name(), mdep.Name(), mdep.Name())
			}
			cmakeContents += "\n"
		}
	})

	android.WriteFileRuleVerbatim(ctx, m.cmakeFilePath, cmakeContents)
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

// TODO: remove
func CmakeCreateList2(name string, itemList []string, areFiles bool) string {
	prefix := ""
	if areFiles {
		prefix = "${ANDROID_BUILD_TOP}/"
	}
	return CmakeCreateList(name, prefix, itemList) + "\n"
}

func CmakeCreateList(name string, prefix string, itemList []string) string {
	var listTemplate = `set({{.Name}}
{{- range .ItemList}}
    {{$.Prefix}}{{.}}
{{- end}}
)
`
	type ListTemplateInput struct {
		Name     string
		Prefix   string
		ItemList []string
	}
	var input = ListTemplateInput{
		name, prefix, itemList,
	}

	var listOut bytes.Buffer
	listTemplateC := template.Must(template.New("listTemplate").Parse(listTemplate))
	if err := listTemplateC.Execute(&listOut, input); err != nil {
		panic(err)
	}
	return listOut.String()
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
	module.AddProperties(&module.properties)
	android.InitAndroidArchModule(module, android.HostSupported, android.MultilibFirst)
	return module
}
