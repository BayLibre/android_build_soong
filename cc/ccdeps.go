// Copyright 2019 Google Inc. All rights reserved.
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
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	"android/soong/android"
)

// This singleton collects cc modules' source and flags into to a json file.
// It does so for generating CMakeLists.txt project files needed data when
// either make, mm, mma, mmm or mmma is called.
// The info file is generated in $OUT/module_bp_cc_depend.json.

func init() {
	android.RegisterSingletonType("ccdeps_generator", ccDepsGeneratorSingleton)
}

func ccDepsGeneratorSingleton() android.Singleton {
	return &ccdepsGeneratorSingleton{}
}

type ccdepsGeneratorSingleton struct {
}

const (
	// Environment variables used to control the behavior of this singleton.
	envVariableCollectCCDeps = "SOONG_COLLECT_CC_DEPS"
	ccdepsJsonFileName       = "module_bp_cc_deps.json"
	cClang                   = "clang"
	cppClang                 = "clang++"
)

type ccIdeInfo struct {
	Path                 []string                   `json:"path,omitempty"`
	Srcs                 []string                   `json:"srcs,omitempty"`
	Global_Common_Flags  map[string]ccMapParameters `json:"global_common_flags,omitempty"`
	Local_Common_Flags   map[string]ccMapParameters `json:"local_common_flags,omitempty"`
	Global_C_flags       map[string]ccMapParameters `json:"global_c_flags,omitempty"`
	Local_C_flags        map[string]ccMapParameters `json:"local_c_flags,omitempty"`
	Global_C_only_flags  map[string]ccMapParameters `json:"global_c_only_flags,omitempty"`
	Local_C_only_flags   map[string]ccMapParameters `json:"local_c_only_flags,omitempty"`
	Global_Cpp_flags     map[string]ccMapParameters `json:"global_cpp_flags,omitempty"`
	Local_Cpp_flags      map[string]ccMapParameters `json:"local_cpp_flags,omitempty"`
	System_include_flags map[string]ccMapParameters `json:"system_include_flags,omitempty"`
	Module_name          string                     `json:"module_name,omitempty"`
}

type ccParameters struct {
	HeaderSearchPath       []string          `json:"header_search_path,omitempty"`
	SystemHeaderSearchPath []string          `json:"system_search_path,omitempty"`
	FlagParameters         []string          `json:"flag,omitempty"`
	SysRoot                string            `json:"system_root,omitempty"`
	RelativeFilePathFlags  map[string]string `json:"relative_file_path,omitempty"`
}

type ccMapParameters map[string]ccParameters

type ccMapIdeInfos map[string]ccIdeInfo

type ccDeps struct {
	C_clang   string        `json:"clang,omitempty"`
	Cpp_clang string        `json:"clang++,omitempty"`
	Modules   ccMapIdeInfos `json:"modules,omitempty"`
}

func (c *ccdepsGeneratorSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	if !ctx.Config().IsEnvTrue(envVariableCollectCCDeps) {
		return
	}

	moduleDeps := ccDeps{}
	moduleInfos := map[string]ccIdeInfo{}

	// Track which projects have already had CMakeLists.txt generated to keep the first
	// variant for each project.
	seenProjects := map[string]bool{}

	pathToCC, _ := evalVariable(ctx, "${config.ClangBin}/")
	moduleDeps.C_clang = fmt.Sprintf("%s%s", buildCMakePath(pathToCC), cClang)
	moduleDeps.Cpp_clang = fmt.Sprintf("%s%s", buildCMakePath(pathToCC), cppClang)

	ctx.VisitAllModules(func(module android.Module) {
		if ccModule, ok := module.(*Module); ok {
			if compiledModule, ok := ccModule.compiler.(CompiledInterface); ok {
				generateCLionProjectData(ctx, compiledModule, ccModule, seenProjects, moduleInfos)
			}
		}
	})

	moduleDeps.Modules = moduleInfos

	ccfpath := android.PathForOutput(ctx, ccdepsJsonFileName).String()
	err := createJsonFile(moduleDeps, ccfpath)
	if err != nil {
		ctx.Errorf(err.Error())
	}
}

func convertCompilerParametersToccParameters(params compilerParameters) ccParameters {
	ccParams := ccParameters{}
	ccParams.HeaderSearchPath = append(ccParams.HeaderSearchPath, params.headerSearchPath...)
	ccParams.SystemHeaderSearchPath = append(ccParams.SystemHeaderSearchPath, params.systemHeaderSearchPath...)
	ccParams.FlagParameters = append(ccParams.FlagParameters, params.flags...)
	ccParams.SysRoot = params.sysroot
	ccParams.RelativeFilePathFlags = map[string]string{}
	for _, param := range params.relativeFilePathFlags {
		ccParams.RelativeFilePathFlags[param.flag] = param.relativeFilePath
	}
	return ccParams
}

// Collect all variable flags' config info here, e.g.: ${config.CommonGlobalIncludes}.
// The config info is easierly evaluated here than in Python.
func parseCompilerVariables(ctx android.SingletonContext, params []string) map[string]ccMapParameters {
	ccVariables := map[string]ccMapParameters{}
	params = normalizeParameters(params)
	for i, param := range params {
		nextParam := ""
		if i != len(params)-1 {
			nextParam = params[i+1]
		}
		ccParamMaps := ccMapParameters{}
		if strings.HasPrefix(param, "$") {
			skip, cmd, varCompilerParameters := parseVariableCompilerParameter(ctx, param, nextParam)
			if skip {
				i = i + 1
			}
			ccParamMaps[cmd] = convertCompilerParametersToccParameters(varCompilerParameters)
		}
		ccVariables[param] = ccParamMaps
	}
	return ccVariables
}

func parseVariableCompilerParameter(ctx android.SingletonContext, param, nextParam string) (bool, string, compilerParameters) {
	var compilerParams = makeCompilerParameters()
	cmd := ""
	skip := false

	switch categorizeParameter(param) {
	case headerSearchPath:
		compilerParams.headerSearchPath =
			append(compilerParams.headerSearchPath, strings.TrimPrefix(param, "-I"))
	case variable:
		if evaluated, error := evalVariable(ctx, param); error == nil {
			cmd = evaluated
			params := strings.Split(evaluated, " ")
			for i := 0; i < len(params); i++ {
				param, nextParam := params[i], ""
				if i != len(params)-1 {
					param, nextParam = params[i], params[i+1]
				}
				skip, _, paramsFromVar := parseVariableCompilerParameter(ctx, param, nextParam)
				if skip {
					i = i + 1
				}
				concatenateParams(&compilerParams, paramsFromVar)
			}
		}
	case systemHeaderSearchPath:
		if nextParam != "" {
			compilerParams.systemHeaderSearchPath = append(compilerParams.systemHeaderSearchPath, nextParam)
		}
		skip = true
	case flag:
		c := cleanupParameter(param)
		compilerParams.flags = append(compilerParams.flags, fmt.Sprintf("# FLAG '%s' became %s\n", param, c))
	case systemRoot:
		if nextParam != "" {
			compilerParams.sysroot = nextParam
		}
		skip = true
	case relativeFilePathFlag:
		flagComponents := strings.Split(param, "=")
		if len(flagComponents) == 2 {
			flagStruct := relativeFilePathFlagType{flag: flagComponents[0], relativeFilePath: flagComponents[1]}
			compilerParams.relativeFilePathFlags = append(compilerParams.relativeFilePathFlags, flagStruct)
		}
	}
	return skip, cmd, compilerParams
}

func generateCLionProjectData(ctx android.SingletonContext, compiledModule CompiledInterface,
	ccModule *Module, seenProjects map[string]bool, moduleInfos map[string]ccIdeInfo) {
	srcs := compiledModule.Srcs()
	if len(srcs) == 0 {
		return
	}

	// Only keep the DeviceArch variant module.
	if ctx.DeviceConfig().DeviceArch() != ccModule.ModuleBase.Arch().ArchType.Name {
		return
	}

	clionProjectLocation := getCMakeListsForModule(ccModule, ctx)
	if seenProjects[clionProjectLocation] {
		return
	}

	seenProjects[clionProjectLocation] = true

	name := ccModule.ModuleBase.Name()
	dpInfo := moduleInfos[name]

	dpInfo.Path = append(dpInfo.Path, path.Dir(ctx.BlueprintFile(ccModule)))
	dpInfo.Srcs = append(dpInfo.Srcs, srcs.Strings()...)
	dpInfo.Path = android.FirstUniqueStrings(dpInfo.Path)
	dpInfo.Srcs = android.FirstUniqueStrings(dpInfo.Srcs)

	global_common_flags := []string{}
	global_common_flags = append(global_common_flags, ccModule.flags.Global.CommonFlags...)
	dpInfo.Global_Common_Flags = parseCompilerVariables(ctx, android.FirstUniqueStrings(global_common_flags))
	local_common_flags := []string{}
	local_common_flags = append(local_common_flags, ccModule.flags.Local.CommonFlags...)
	dpInfo.Local_Common_Flags = parseCompilerVariables(ctx, android.FirstUniqueStrings(local_common_flags))
	global_c_flags := []string{}
	global_c_flags = append(global_c_flags, ccModule.flags.Global.CFlags...)
	dpInfo.Global_C_flags = parseCompilerVariables(ctx, android.FirstUniqueStrings(global_c_flags))
	local_c_flags := []string{}
	local_c_flags = append(local_c_flags, ccModule.flags.Local.CFlags...)
	dpInfo.Local_C_flags = parseCompilerVariables(ctx, android.FirstUniqueStrings(local_c_flags))
	global_c_only_flags := []string{}
	global_c_only_flags = append(global_c_only_flags, ccModule.flags.Global.ConlyFlags...)
	dpInfo.Global_C_only_flags = parseCompilerVariables(ctx, android.FirstUniqueStrings(global_c_only_flags))
	local_c_only_flags := []string{}
	local_c_only_flags = append(local_c_only_flags, ccModule.flags.Local.ConlyFlags...)
	dpInfo.Local_C_only_flags = parseCompilerVariables(ctx, android.FirstUniqueStrings(local_c_only_flags))
	global_cpp_flags := []string{}
	global_cpp_flags = append(global_cpp_flags, ccModule.flags.Global.CppFlags...)
	dpInfo.Global_Cpp_flags = parseCompilerVariables(ctx, android.FirstUniqueStrings(global_cpp_flags))
	local_cpp_flags := []string{}
	local_cpp_flags = append(local_cpp_flags, ccModule.flags.Local.CppFlags...)
	dpInfo.Local_Cpp_flags = parseCompilerVariables(ctx, android.FirstUniqueStrings(local_cpp_flags))
	system_include_flags := []string{}
	system_include_flags = append(system_include_flags, ccModule.flags.SystemIncludeFlags...)
	dpInfo.System_include_flags = parseCompilerVariables(ctx, android.FirstUniqueStrings(system_include_flags))

	dpInfo.Module_name = name

	moduleInfos[name] = dpInfo
}

type Deal struct {
	Name    string
	ideInfo ccIdeInfo
}

type Deals []Deal

// Ensure it satisfies sort.Interface
func (d Deals) Len() int           { return len(d) }
func (d Deals) Less(i, j int) bool { return d[i].Name < d[j].Name }
func (d Deals) Swap(i, j int)      { d[i], d[j] = d[j], d[i] }

func sortMap(moduleInfos map[string]ccIdeInfo) map[string]ccIdeInfo {
	var deals Deals
	for k, v := range moduleInfos {
		deals = append(deals, Deal{k, v})
	}

	sort.Sort(deals)

	m := map[string]ccIdeInfo{}
	for _, d := range deals {
		m[d.Name] = d.ideInfo
	}
	return m
}

func createJsonFile(moduleDeps ccDeps, ccfpath string) error {
	file, err := os.Create(ccfpath)
	if err != nil {
		return fmt.Errorf("Failed to create file: %s, relative: %v", ccdepsJsonFileName, err)
	}
	defer file.Close()
	moduleDeps.Modules = sortMap(moduleDeps.Modules)
	buf, err := json.MarshalIndent(moduleDeps, "", "\t")
	if err != nil {
		return fmt.Errorf("Write file failed: %s, relative: %v", ccdepsJsonFileName, err)
	}
	fmt.Fprintf(file, string(buf))
	return nil
}
