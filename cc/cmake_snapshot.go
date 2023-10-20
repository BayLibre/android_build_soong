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
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"text/template"

	"github.com/google/blueprint"
)

const veryVerbose bool = false

//go:embed cmake_main.txt
var templateCmakeMainRaw string
var templateCmakeMain *template.Template = parseTemplate(templateCmakeMainRaw)

//go:embed cmake_module_cc.txt
var templateCmakeModuleCcRaw string
var templateCmakeModuleCc *template.Template = parseTemplate(templateCmakeModuleCcRaw)

//go:embed cmake_module_aidl.txt
var templateCmakeModuleAidlRaw string
var templateCmakeModuleAidl *template.Template = parseTemplate(templateCmakeModuleAidlRaw)

//go:embed cmake_ext_add_aidl_library.txt
var cmakeExtAddAidlLibrary string

//go:embed cmake_ext_append_flags.txt
var cmakeExtAppendFlags string

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

var ignoredSystemLibs []string = []string{
	"libc++",
	"libc++_static",
	"prebuilt_libclang_rt.builtins",
	"prebuilt_libclang_rt.ubsan_minimal",
}

// Mapping entry between Android's library name and the one used when building outside Android tree.
type LibraryMappingProperty struct {
	// Android library name.
	Android_name string

	// Library name used when building outside Android.
	Mapped_name string

	// If the make file is already present in Android source tree, specify its location.
	Package_pregenerated string

	// If the package is expected to be installed on the build host OS, specify its name.
	Package_system string
}

type CmakeSnapshotProperties struct {
	// Modules to add to the snapshot package. Their dependencies are pulled in automatically.
	Modules []string

	// Host prebuilts to bundle with the snapshot. These are tools needed to build outside Android.
	Prebuilts []string

	// Global cflags to add when building outside Android.
	Cflags []string

	// Flags to skip when building outside Android.
	Cflags_ignored []string

	// Mapping between library names used in Android tree and externally.
	Library_mapping []LibraryMappingProperty

	// List of cflags that are not portable between compilers that could potentially be used to
	// build a generated package. If left empty, it's initialized with a default list.
	Unportable_flags []string

	// Whether to include source code as part of the snapshot package.
	Include_sources bool
}

type cmakeSnapshotData struct {
	LibraryMapping       map[string]LibraryMappingProperty
	PregeneratedPackages []string
	SystemPackages       []string

	sourceFiles map[string]android.Path
}

// cmake_snapshot allows defining source packages for release outside of Android build tree.
// As a result of cmake_snapshot module build, a zip file is generated with CMake build definitions
// for selected source modules, their dependencies and optionally also the source code itself.
type CmakeSnapshot struct {
	android.ModuleBase

	Properties CmakeSnapshotProperties

	zipPath android.WritablePath
}

type cmakeSnapshotDependencyTag struct {
	blueprint.BaseDependencyTag
	name string
}

var (
	cmakeSnapshotModuleTag   = cmakeSnapshotDependencyTag{name: "cmake-snapshot-module"}
	cmakeSnapshotPrebuiltTag = cmakeSnapshotDependencyTag{name: "cmake-snapshot-prebuilt"}
)

func parseTemplate(templateContents string) *template.Template {
	funcMap := template.FuncMap{
		"setList": func(name string, nameSuffix string, itemPrefix string, items []string) string {
			var list strings.Builder
			list.WriteString("set(" + name + nameSuffix)
			templateListBuilder(&list, itemPrefix, items)
			return list.String()
		},
		"toStrings": func(files android.Paths) []string {
			strings := make([]string, len(files))
			for idx, file := range files {
				strings[idx] = file.String()
			}
			return strings
		},
		"concat5": func(list1 []string, list2 []string, list3 []string, list4 []string, list5 []string) []string {
			return append(append(append(append(list1, list2...), list3...), list4...), list5...)
		},
		"cflagsList": func(name string, nameSuffix string, flags []string,
			unportableFlags []string, ignoredFlags []string) string {
			if len(unportableFlags) == 0 {
				unportableFlags = defaultUnportableFlags
			}

			var filteredPortable []string
			var filteredUnportable []string
			for _, flag := range flags {
				if slices.Contains(ignoredFlags, flag) {
					continue
				} else if slices.Contains(unportableFlags, flag) {
					filteredUnportable = append(filteredUnportable, flag)
				} else {
					filteredPortable = append(filteredPortable, flag)
				}
			}

			var list strings.Builder

			list.WriteString("set(" + name + nameSuffix)
			templateListBuilder(&list, "", filteredPortable)

			if len(filteredUnportable) > 0 {
				list.WriteString("\nappend_cxx_flags_if_supported(" + name + nameSuffix)
				templateListBuilder(&list, "", filteredUnportable)
			}

			return list.String()
		},
		"getSources": func(m *Module) android.Paths {
			return m.compiler.(CompiledInterface).Srcs()
		},
		"getModuleType":         getModuleType,
		"getCompilerProperties": getCompilerProperties,
		"getLinkerProperties":   getLinkerProperties,
		"getExtraLibs":          getExtraLibs,
		"getIncludeDirs":        getIncludeDirs,
		"mapLibraries": func(libs []string, mapping map[string]LibraryMappingProperty) []string {
			var mappedLibs []string
			for _, lib := range libs {
				mappedLib, exists := mapping[lib]
				if exists {
					lib = mappedLib.Mapped_name
				} else {
					lib = "android::" + lib
				}
				if lib == "" {
					continue
				}
				mappedLibs = append(mappedLibs, lib)
			}
			sort.Strings(mappedLibs)
			mappedLibs = slices.Compact(mappedLibs)
			return mappedLibs
		},
	}

	return template.Must(template.New("").Delims("<<", ">>").Funcs(funcMap).Parse(templateContents))
}

func sliceWithPrefix(prefix string, slice []string) []string {
	output := make([]string, len(slice))
	for i, elem := range slice {
		output[i] = prefix + elem
	}
	return output
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

// CmakeSnapshot is re-initialized between mutators and GenerateAndroidBuildActions so we need to
// hold its data in a cache
var cmakeSnapshotCache map[string]*cmakeSnapshotData = map[string]*cmakeSnapshotData{}

func (m *CmakeSnapshot) Data() *cmakeSnapshotData {
	if data, ok := cmakeSnapshotCache[m.Name()]; ok {
		return data
	} else {
		data := &cmakeSnapshotData{
			sourceFiles: map[string]android.Path{},
		}
		cmakeSnapshotCache[m.Name()] = data
		return data
	}
}

func (m *CmakeSnapshot) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	var templateBuffer bytes.Buffer
	m.zipPath = android.PathForModuleOut(ctx, ctx.ModuleName()+".zip")

	// Generating CMakeLists.txt rules for all modules in dependency tree
	moduleDirs := make(map[string][]string)
	m.walkDepsOnce(ctx, func(dep *Module) {
		templateToUse := templateCmakeModuleCc
		if getCompilerProperties(dep).AidlInterface.Lang != "" {
			templateToUse = templateCmakeModuleAidl
		}
		moduleFragment := executeTemplate(templateToUse, &templateBuffer, struct {
			Ctx      *android.ModuleContext
			M        *Module
			Snapshot *CmakeSnapshot
		}{
			&ctx,
			dep,
			m,
		})

		moduleDir := ctx.OtherModuleDir(dep)
		moduleDirs[moduleDir] = append(moduleDirs[moduleDir], moduleFragment)
	})

	// Merging CMakeLists.txt contents for every module directory
	var makefilesList android.Paths
	for moduleDir, fragments := range moduleDirs {
		moduleCmakePath := android.PathForModuleGen(ctx, moduleDir, "CMakeLists.txt")
		makefilesList = append(makefilesList, moduleCmakePath)
		sort.Strings(fragments)
		android.WriteFileRule(ctx, moduleCmakePath, strings.Join(fragments, "\n\n\n"))
	}

	// Generating top-level CMakeLists.txt
	mainCmakePath := android.PathForModuleGen(ctx, "CMakeLists.txt")
	makefilesList = append(makefilesList, mainCmakePath)
	mainContents := executeTemplate(templateCmakeMain, &templateBuffer, struct {
		M          *CmakeSnapshot
		ModuleDirs map[string][]string
	}{
		m,
		moduleDirs,
	})
	android.WriteFileRule(ctx, mainCmakePath, mainContents)

	// Generating CMake extensions
	extPath := android.PathForModuleGen(ctx, "cmake", "AppendCxxFlagsIfSupported.cmake")
	makefilesList = append(makefilesList, extPath)
	android.WriteFileRuleVerbatim(ctx, extPath, cmakeExtAppendFlags)
	extPath = android.PathForModuleGen(ctx, "cmake", "AddAidlLibrary.cmake")
	makefilesList = append(makefilesList, extPath)
	android.WriteFileRuleVerbatim(ctx, extPath, cmakeExtAddAidlLibrary)

	// Generating the final zip file
	zipRule := android.NewRuleBuilder(pctx, ctx)
	zipCmd := zipRule.Command().
		BuiltTool("soong_zip").
		FlagWithOutput("-o ", m.zipPath)

	// Packaging all sources into the zip file
	if m.Properties.Include_sources {
		var sourcesList android.Paths
		for _, file := range m.Data().sourceFiles {
			sourcesList = append(sourcesList, file)
		}

		sourcesRspFile := android.PathForModuleObj(ctx, ctx.ModuleName()+"_sources.rsp")
		zipCmd.FlagWithRspFileInputList("-r ", sourcesRspFile, sourcesList)
	}

	// Packaging all make files into the zip file
	makefilesRspFile := android.PathForModuleObj(ctx, ctx.ModuleName()+"_makefiles.rsp")
	zipCmd.
		FlagWithArg("-C ", android.PathForModuleGen(ctx).OutputPath.String()).
		FlagWithRspFileInputList("-r ", makefilesRspFile, makefilesList)

	// Packaging all prebuilts into the zip file
	if len(m.Properties.Prebuilts) > 0 {
		var prebuiltsList android.Paths

		ctx.VisitDirectDepsWithTag(cmakeSnapshotPrebuiltTag, func(dep android.Module) {
			for _, file := range dep.FilesToInstall() {
				prebuiltsList = append(prebuiltsList, file)
			}
		})

		prebuiltsRspFile := android.PathForModuleObj(ctx, ctx.ModuleName()+"_prebuilts.rsp")
		zipCmd.
			FlagWithArg("-C ", android.PathForArbitraryOutput(ctx).String()).
			FlagWithArg("-P ", "prebuilts").
			FlagWithRspFileInputList("-r ", prebuiltsRspFile, prebuiltsList)
	}

	// Finish generating the final zip file
	zipRule.Build(m.zipPath.String(), "archiving "+ctx.ModuleName())
}

func (m *CmakeSnapshot) walkDepsOnce(mctx android.BaseModuleContext, visit func(dep *Module)) {
	visitedModules := make(map[string]bool)

	mctx.WalkDeps(func(amod android.Module, parent android.Module) bool {
		if ccmod, ok := amod.(*Module); ok {
			moduleName := mctx.OtherModuleName(amod)
			if _, ok := visitedModules[moduleName]; ok {
				return false // visit only once
			}
			if _, ok := m.Data().LibraryMapping[moduleName]; ok {
				return false // mapped to system or pregenerated, so they are not real dependencies
			}
			if mctx.OtherModuleName(parent) == mctx.ModuleName() {
				if slices.Contains(m.Properties.Prebuilts, moduleName) {
					return false // it's a prebuilt dependency, not a module to add
				}
			}
			if slices.Contains(ignoredSystemLibs, moduleName) {
				return false // system libs build in-tree for Android
			}
			modCompilerProperties := getCompilerProperties(ccmod)
			if modCompilerProperties == nil {
				return false // unsupported module type (e.g. prebuilt)
			}
			visitedModules[moduleName] = true

			if veryVerbose {
				fmt.Println("WalkDeps: " + mctx.OtherModuleName(parent) + " -> " + moduleName)
			}

			visit(ccmod)

			// if it's AIDL module, no need to dive into their dependencies
			return modCompilerProperties.AidlInterface.Lang == ""
		}
		return false
	})
}

func (m *CmakeSnapshot) OutputFiles(tag string) (android.Paths, error) {
	switch tag {
	case "":
		return android.Paths{m.zipPath}, nil
	default:
		return nil, fmt.Errorf("unsupported module reference tag %q", tag)
	}
}

func (m *CmakeSnapshot) AndroidMkEntries() []android.AndroidMkEntries {
	return []android.AndroidMkEntries{{
		Class:      "DATA",
		OutputFile: android.OptionalPathForPath(m.zipPath),
		ExtraEntries: []android.AndroidMkExtraEntriesFunc{
			func(ctx android.AndroidMkExtraEntriesContext, entries *android.AndroidMkEntries) {
				entries.SetBool("LOCAL_UNINSTALLABLE_MODULE", true)
			},
		},
	}}
}

func getModuleType(m *Module) string {
	switch m.linker.(type) {
	case *binaryDecorator:
		return "executable"
	case *libraryDecorator:
		return "library"
	case *testBinary:
		return "executable"
	}
	panic(fmt.Sprintf("Unexpected module type: %T", m.compiler))
}

func getCompilerProperties(m *Module) *BaseCompilerProperties {
	switch decorator := m.compiler.(type) {
	case *baseCompiler:
		return &decorator.Properties
	case *libraryDecorator:
		return &decorator.baseCompiler.Properties
	case *testBinary:
		return &decorator.baseCompiler.Properties
	case nil:
		return nil
	}
	panic(fmt.Sprintf("Unexpected module type: %T", m.compiler))
}

func getLinkerProperties(m *Module) *BaseLinkerProperties {
	switch decorator := m.linker.(type) {
	case *binaryDecorator:
		return &decorator.baseLinker.Properties
	case *libraryDecorator:
		return &decorator.baseLinker.Properties
	case *testBinary:
		return &decorator.baseLinker.Properties
	case nil:
		return nil
	}
	panic(fmt.Sprintf("Unexpected module type: %T", m.linker))
}

func getExtraLibs(m *Module) []string {
	switch decorator := m.linker.(type) {
	case *testBinary:
		if decorator.testDecorator.gtest() {
			return []string{"libgtest"}
		}
	}
	return nil
}

func getIncludeDirs(ctx android.ModuleContext, m *Module) []string {
	moduleDir := ctx.OtherModuleDir(m) + string(filepath.Separator)
	switch decorator := m.compiler.(type) {
	case *libraryDecorator:
		return sliceWithPrefix(moduleDir, decorator.flagExporter.Properties.Export_include_dirs)
	}
	return nil
}

// Factory for CmakeSnapshot (providing cmake_snapshot support for Soong)
func CmakeSnapshotFactory() android.Module {
	module := &CmakeSnapshot{}
	module.AddProperties(&module.Properties)
	android.InitAndroidArchModule(module, android.HostSupported, android.MultilibFirst)
	return module
}

// Which snapshots are using given directories - needed to pass source info from
// cmakeSnapshotPostDepsMutator and Package_pregenerated to cmakeSnapshotFinalDepsMutator.
// key=dir name, value=snapshot names set
var cmakeSnapshotsUsingDirs map[string]map[string]*cmakeSnapshotData = map[string]map[string]*cmakeSnapshotData{}

func addSnapshotUsingDir(moduleDir string, snapshot *CmakeSnapshot) {
	if cmakeSnapshotsUsingDirs[moduleDir] == nil {
		cmakeSnapshotsUsingDirs[moduleDir] = map[string]*cmakeSnapshotData{}
	}
	cmakeSnapshotsUsingDirs[moduleDir][snapshot.Name()] = snapshot.Data()
}

// - adds snapshot dependencies
// - pre-processes Library_mapping property
func cmakeSnapshotPreDepsMutator(mctx android.BottomUpMutatorContext) {
	if m, ok := mctx.Module().(*CmakeSnapshot); ok {
		mctx.AddDependency(mctx.Module(), cmakeSnapshotModuleTag, m.Properties.Modules...)
		mctx.AddDependency(mctx.Module(), cmakeSnapshotPrebuiltTag, m.Properties.Prebuilts...)

		m.Data().LibraryMapping = make(map[string]LibraryMappingProperty)
		for _, elem := range m.Properties.Library_mapping {
			m.Data().LibraryMapping[elem.Android_name] = elem

			if elem.Package_pregenerated != "" && !slices.Contains(m.Data().PregeneratedPackages,
				elem.Package_pregenerated) {
				m.Data().PregeneratedPackages = append(m.Data().PregeneratedPackages, elem.Package_pregenerated)
				addSnapshotUsingDir(elem.Package_pregenerated, m)
			}
			if elem.Package_system != "" && !slices.Contains(m.Data().SystemPackages, elem.Package_system) {
				m.Data().SystemPackages = append(m.Data().SystemPackages, elem.Package_system)
			}
		}
	}
}

func registerCmakeSnapshotPreDepsMutators(ctx android.RegisterMutatorsContext) {
	ctx.BottomUp("cmakeSnapshotPreDepsMutator", cmakeSnapshotPreDepsMutator).Parallel()
}

// generates map saying which snapshots need to pull in which dirs as sources
func cmakeSnapshotPostDepsMutator(mctx android.TopDownMutatorContext) {
	if m, ok := mctx.Module().(*CmakeSnapshot); ok {
		m.walkDepsOnce(mctx, func(dep *Module) {
			addSnapshotUsingDir(mctx.OtherModuleDir(dep), m)
		})
	}
}

func registerCmakeSnapshotPostDepsMutators(ctx android.RegisterMutatorsContext) {
	ctx.TopDown("cmakeSnapshotPostDepsMutator", cmakeSnapshotPostDepsMutator)
}

// adds source file paths to snapshots
func cmakeSnapshotFinalDepsMutator(mctx android.TopDownMutatorContext) {
	moduleDir := mctx.ModuleDir()
	if snapshots, ok := cmakeSnapshotsUsingDirs[moduleDir]; ok {
		var files android.Paths

		for _, path := range android.GlobFiles(mctx, moduleDir+"/**/*", nil) {
			files = append(files, path)
		}

		for _, snapshot := range snapshots {
			for _, file := range files {
				snapshot.sourceFiles[file.String()] = file
			}
		}
		delete(cmakeSnapshotsUsingDirs, moduleDir) // no need to populate this dir again
	}
}

func registerCmakeSnapshotFinalDepsMutators(ctx android.RegisterMutatorsContext) {
	ctx.TopDown("cmakeSnapshotFinalDepsMutator", cmakeSnapshotFinalDepsMutator)
}

func init() {
	android.InitRegistrationContext.RegisterModuleType("cc_cmake_snapshot", CmakeSnapshotFactory)
	android.PreDepsMutators(registerCmakeSnapshotPreDepsMutators)
	android.PostDepsMutators(registerCmakeSnapshotPostDepsMutators)
	android.FinalDepsMutators(registerCmakeSnapshotFinalDepsMutators)
}
