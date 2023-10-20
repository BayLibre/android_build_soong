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
	_ "embed"
	"fmt"
	"github.com/google/blueprint"
	"slices"
	"strings"
	"text/template"
)

//go:embed library_cmake_main.txt
var templateCmakeMainRaw string
var templateCmakeMain *template.Template = parseTemplate(templateCmakeMainRaw)

//go:embed library_cmake_module_cc.txt
var templateCmakeModuleCcRaw string
var templateCmakeModuleCc *template.Template = parseTemplate(templateCmakeModuleCcRaw)

//go:embed library_cmake_module_aidl.txt
var templateCmakeModuleAidlRaw string
var templateCmakeModuleAidl *template.Template = parseTemplate(templateCmakeModuleAidlRaw)

//go:embed library_cmake_macro_append_flags.txt
var cmakeMacroAppendFlags string

var defaultUnportableFlags []string = []string{
	"-Wno-class-memaccess",
	"-Wno-exit-time-destructors",
	"-Wno-inconsistent-missing-override",
	"-Wreorder-init-list",
	"-Wno-reorder-init-list",
	"-Wno-restrict",
	"-Wno-stringop-overread",
	"-Wno-subobject-linkage",
}

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
	UnportableFlags          []string
}

type LibraryCmakeSnapshot struct {
	android.ModuleBase

	zipPath    android.WritablePath
	Properties LibraryCmakeSnapshotProperties
}

func parseTemplate(templateContents string) *template.Template {
	funcMap := template.FuncMap{
		"setList": func(name string, nameSuffix string, itemPrefix string, items []string) string {
			var list strings.Builder
			list.WriteString("set(" + name + nameSuffix)
			templateListBuilder(&list, itemPrefix, items)
			return list.String()
		},
		"toStrings": func(files []android.Path) []string {
			strings := make([]string, len(files))
			for idx, file := range files {
				strings[idx] = file.String()
			}
			return strings
		},
		"cflagsList": func(name string, nameSuffix string, flags []string,
			unportableFlags []string) string {
			if len(unportableFlags) == 0 {
				unportableFlags = defaultUnportableFlags
			}

			filteredPortable := []string{}
			filteredUnportable := []string{}
			for _, flag := range flags {
				if slices.Contains(unportableFlags, flag) {
					filteredUnportable = append(filteredUnportable, flag)
				} else {
					filteredPortable = append(filteredPortable, flag)
				}
			}

			var list strings.Builder

			list.WriteString("set(" + name + nameSuffix)
			templateListBuilder(&list, "", filteredPortable)

			list.WriteString("\nappend_cxx_flags_if_supported(" + name + nameSuffix)
			templateListBuilder(&list, "", filteredUnportable)

			return list.String()
		},
		"getSources": func(m *Module) []android.Path {
			return m.compiler.(CompiledInterface).Srcs()
		},
		"getCompilerProperties": getCompilerProperties,
		"getModuleType":         getModuleType,
		"getExporterInfo": func(ctx android.ModuleContext, m *Module) FlagExporterInfo {
			info, _ := android.OtherModuleProvider(ctx, m, FlagExporterInfoProvider)
			return info
		},
	}

	return template.Must(template.New("").Delims("<<", ">>").Funcs(funcMap).Parse(templateContents))
}

func templateListBuilder(builder *strings.Builder, itemPrefix string, items []string) {
	if len(items) > 0 {
		builder.WriteString("\n")
		for _, item := range items {
			builder.WriteString("    " + itemPrefix + item + "\n")
		}
	}
	builder.WriteString(")")
}

func executeTemplate(templ *template.Template, buffer *bytes.Buffer, data any) string {
	buffer.Reset()
	if err := templ.Execute(buffer, data); err != nil {
		panic(err)
	}
	output := strings.TrimSpace(buffer.String())
	buffer.Reset()
	return output
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

	// Generating CMakeLists.txt for all modules in dependency tree

	var android_lib_to_cmake = make(map[string][]string)
	for _, a2c := range m.Properties.Android_libname_to_cmake {
		android_lib_to_cmake[a2c.Android_lib] = a2c.Cmake_libs
	}

	modules := make(map[string]bool)

	// TODO(now): ignored system libs
	modules["libc++"] = true
	modules["libc++_static"] = true
	modules["prebuilt_libclang_rt.ubsan_minimal"] = true

	// TODO(now): integrate with android_libname_to_cmake
	modules["libcrypto"] = true
	modules["libgtest"] = true
	modules["libgtest_main"] = true
	modules["libssl"] = true

	moduleDirs := make(map[string][]string)

	ctx.WalkDeps(func(dep android.Module, parent android.Module) bool {
		if mdep, ok := dep.(*Module); ok {
			if _, ok := modules[mdep.Name()]; ok {
				return false
			}
			modules[mdep.Name()] = true

			// TODO(now): remove before merging
			fmt.Println("WalkDeps " + parent.Name() + " -> " + dep.Name())

			if getModuleType(mdep) == "" {
				return false
			}

			aidlLang := getCompilerProperties(mdep).AidlInterface.Lang

			baseLinkerProperties := getLinkerProperties(mdep)

			templateToUse := templateCmakeModuleCc
			if aidlLang != "" {
				templateToUse = templateCmakeModuleAidl
			}
			moduleFragment := executeTemplate(templateToUse, &templateBuffer, struct {
				Ctx   *android.ModuleContext
				M     *Module
				MT    *LibraryCmakeSnapshot
				Wsdep []string
				Sdep  []string
				Ddep  []string
				Hdep  []string
			}{
				&ctx,
				mdep,
				m,
				AndroidLibListToCmake(baseLinkerProperties.Whole_static_libs, android_lib_to_cmake),
				AndroidLibListToCmake(baseLinkerProperties.Static_libs, android_lib_to_cmake),
				AndroidLibListToCmake(baseLinkerProperties.Shared_libs, android_lib_to_cmake),
				AndroidLibListToCmake(baseLinkerProperties.Header_libs, android_lib_to_cmake),
			})

			moduleDir := ctx.OtherModuleDir(dep)
			moduleDirs[moduleDir] = append(moduleDirs[moduleDir], moduleFragment)

			return aidlLang == "" // if it's AIDL, don't dive into dependencies
		}
		return false
	})

	for moduleDir, fragments := range moduleDirs {
		moduleCmakePath := android.PathForModuleOut(ctx, moduleDir, "CMakeLists.txt")
		zipList = append(zipList, moduleCmakePath)
		android.WriteFileRule(ctx, moduleCmakePath, strings.Join(fragments, "\n"))
	}

	// Generating main CMakeLists.txt

	mainCmakePath := android.PathForModuleOut(ctx, "CMakeLists.txt")
	zipList = append(zipList, mainCmakePath)
	mainContents := executeTemplate(templateCmakeMain, &templateBuffer, struct {
		M          *LibraryCmakeSnapshot
		ModuleDirs map[string][]string
	}{
		m,
		moduleDirs,
	})
	android.WriteFileRule(ctx, mainCmakePath, mainContents)

	// Generating CMake macros

	macroAppendFlagsPath := android.PathForModuleOut(ctx, "cmake", "AppendCxxFlagsIfSupported.cmake")
	zipList = append(zipList, macroAppendFlagsPath)
	android.WriteFileRule(ctx, macroAppendFlagsPath, cmakeMacroAppendFlags)

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

func getModuleType(m *Module) string {
	switch m.compiler.(type) {
	case *libraryDecorator:
		return "library"
	case *testBinary:
		return "executable"
	case nil: // e.g. builtins
		return ""
	}
	panic(fmt.Sprintf("Unexpected module type: %T", m.compiler))
}

func getCompilerProperties(m *Module) BaseCompilerProperties {
	switch decorator := m.compiler.(type) {
	case *libraryDecorator:
		return decorator.baseCompiler.Properties
	case *testBinary:
		return decorator.baseCompiler.Properties
	}
	panic(fmt.Sprintf("Unexpected module type: %T", m.compiler))
}

func getLinkerProperties(m *Module) BaseLinkerProperties {
	switch decorator := m.linker.(type) {
	case *libraryDecorator:
		return decorator.baseLinker.Properties
	case *testBinary:
		return decorator.baseLinker.Properties
	}
	panic(fmt.Sprintf("Unexpected module type: %T", m.linker))
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
