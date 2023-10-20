// Copyright 2023 Google Inc. All rights reserved.
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
	"fmt"
	"github.com/google/blueprint"
	"strings"
)

func RegisterLibraryCmakeComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("cc_library_cmake_snapshot", LibraryCmakeSnapshotFactory)
}

func RegisterPreArchMutators(ctx android.RegisterMutatorsContext) {
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

func init() {
	RegisterLibraryCmakeComponents(android.InitRegistrationContext)
	android.PreDepsMutators(RegisterPreArchMutators)
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

	rule := android.NewRuleBuilder(pctx, ctx)
	rule.Command().Text("rm").Flag("-f").Output(m.cmakeFilePath)
	rule.Command().Text("echo 'cmake_minimum_required(VERSION 3.18)' >> ").Output(m.cmakeFilePath)
	rule.Command().Text(fmt.Sprintf("echo 'project(%s CXX)' >> ", m.Name())).Output(m.cmakeFilePath)
	rule.Command().Text("echo 'set(CMAKE_CXX_STANDARD 20)' >>").Output(m.cmakeFilePath)
	rule.Command().Text("echo '' >> ").Output(m.cmakeFilePath)

	for _, subdir := range m.properties.Subdirectories {
		rule.Command().Text(fmt.Sprintf("echo 'add_subdirectory(\"%s\" %s EXCLUDE_FROM_ALL)' >> ", subdir.Path, subdir.Name)).Output(m.cmakeFilePath)
		rule.Command().Text("echo '' >> ").Output(m.cmakeFilePath)
	}

	CmakeCreateList(rule, m.cmakeFilePath, fmt.Sprintf("%s_CFLAGS", m.Name()), m.properties.Cflags)
	CmakeCreateList(rule, m.cmakeFilePath, fmt.Sprintf("%s_LINKFLAGS", m.Name()), m.properties.Linkflags)

	var android_lib_to_cmake = make(map[string][]string)
	for _, a2c := range m.properties.Android_libname_to_cmake {
		android_lib_to_cmake[a2c.Android_lib] = a2c.Cmake_libs
	}

	ctx.VisitDirectDeps(func(dep android.Module) {
		if mdep, ok := dep.(*Module); ok {
			is_interface := len(mdep.compiler.(CompiledInterface).Srcs()) == 0
			if is_interface {
				rule.Command().Text(fmt.Sprintf("echo 'add_library(%s INTERFACE)' >> ", mdep.Name())).Output(m.cmakeFilePath)
			} else {
				var srcs_str []string
				for _, src := range mdep.compiler.(CompiledInterface).Srcs() {
					srcs_str = append(srcs_str, src.String())
				}
				CmakeCreateList(rule, m.cmakeFilePath, fmt.Sprintf("%s_SRC", mdep.Name()), srcs_str)
				rule.Command().Text(fmt.Sprintf("echo 'add_library(%s ${%s_SRC})' >> ", mdep.Name(), mdep.Name())).Output(m.cmakeFilePath)
			}
			rule.Command().Text("echo '' >> ").Output(m.cmakeFilePath)

			depExporterInfo := ctx.OtherModuleProvider(dep, FlagExporterInfoProvider).(FlagExporterInfo)
			var exp_inc_str []string
			for _, exp_inc := range depExporterInfo.IncludeDirs {
				exp_inc_str = append(exp_inc_str, exp_inc.String())
			}

			CmakeCreateList(rule, m.cmakeFilePath, fmt.Sprintf("%s_EXPINC", mdep.Name()), exp_inc_str)
			CmakeCreateList(rule, m.cmakeFilePath, fmt.Sprintf("%s_INC", mdep.Name()),
				append(mdep.compiler.(*libraryDecorator).baseCompiler.Properties.Include_dirs, mdep.compiler.(*libraryDecorator).baseCompiler.Properties.Local_include_dirs...))
			if is_interface {
				rule.Command().Text(fmt.Sprintf("echo 'target_include_directories(%s INTERFACE ${%s_EXPINC} ${%s_INC})' >> ", mdep.Name(), mdep.Name(), mdep.Name())).Output(m.cmakeFilePath)
			} else {
				rule.Command().Text(fmt.Sprintf("echo 'target_include_directories(%s PUBLIC ${%s_EXPINC} PRIVATE ${%s_INC})' >> ", mdep.Name(), mdep.Name(), mdep.Name())).Output(m.cmakeFilePath)
				rule.Command().Text(fmt.Sprintf("echo 'target_compile_options(%s PRIVATE ${%s_CFLAGS})' >> ", mdep.Name(), m.Name())).Output(m.cmakeFilePath)
				rule.Command().Text(fmt.Sprintf("echo 'target_link_options(%s PRIVATE ${%s_LINKFLAGS})' >> ", mdep.Name(), m.Name())).Output(m.cmakeFilePath)
			}
			rule.Command().Text("echo '' >> ").Output(m.cmakeFilePath)

			CmakeCreateList(rule, m.cmakeFilePath, fmt.Sprintf("%s_WSDEP", mdep.Name()),
				AndroidLibListToCmake(mdep.linker.(*libraryDecorator).baseLinker.Properties.Whole_static_libs, android_lib_to_cmake))
			CmakeCreateList(rule, m.cmakeFilePath, fmt.Sprintf("%s_SDEP", mdep.Name()),
				AndroidLibListToCmake(mdep.linker.(*libraryDecorator).baseLinker.Properties.Static_libs, android_lib_to_cmake))
			CmakeCreateList(rule, m.cmakeFilePath, fmt.Sprintf("%s_DDEP", mdep.Name()),
				AndroidLibListToCmake(mdep.linker.(*libraryDecorator).baseLinker.Properties.Shared_libs, android_lib_to_cmake))
			CmakeCreateList(rule, m.cmakeFilePath, fmt.Sprintf("%s_HDEP", mdep.Name()),
				AndroidLibListToCmake(mdep.linker.(*libraryDecorator).baseLinker.Properties.Header_libs, android_lib_to_cmake))

			if is_interface {
				rule.Command().Text(fmt.Sprintf("echo 'target_link_libraries(%s INTERFACE ${%s_WSDEP} ${%s_SDEP} ${%s_DDEP} ${%s_HDEP})' >> ",
					mdep.Name(), mdep.Name(), mdep.Name(), mdep.Name(), mdep.Name())).Output(m.cmakeFilePath)
			} else {
				rule.Command().Text(fmt.Sprintf("echo 'target_link_libraries(%s ${%s_WSDEP} ${%s_SDEP} ${%s_DDEP} ${%s_HDEP})' >> ",
					mdep.Name(), mdep.Name(), mdep.Name(), mdep.Name(), mdep.Name())).Output(m.cmakeFilePath)
			}
			rule.Command().Text("echo '' >> ").Output(m.cmakeFilePath)
		}
	})

	rule.Build(m.Name(), "CMake snapshot "+m.Name())
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

func CmakeCreateList(rule *android.RuleBuilder, out android.WritablePath, name string, itemList []string) {
	rule.Command().Text(fmt.Sprintf("echo 'set(%s' >> ", name)).Output(out)
	for _, items := range itemList {
		for _, item := range strings.Split(items, " ") {
			if item == "" {
				continue
			}
			rule.Command().Text(fmt.Sprintf("echo '    %s' >> ", item)).Output(out)
		}
	}
	rule.Command().Text("echo ')' >> ").Output(out)
	rule.Command().Text("echo '' >> ").Output(out)
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
	android.InitAndroidModule(module)
	return module
}
