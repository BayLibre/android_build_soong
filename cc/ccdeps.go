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
	nativeConfigFileName     = "native_config.json"
	cLang                    = "clang"
	cppLang                  = "clang++"
)

type ccIdeInfo struct {
	Path                 []string   `json:"path,omitempty"`
	Srcs                 []string   `json:"srcs,omitempty"`
	Global_flags         []string   `json:"global_flags,omitempty"`
	C_flags              []string   `json:"c_flags,omitempty"`
	C_only_flags         []string   `json:"c_only_flags,omitempty"`
	Cpp_flags            []string   `json:"cpp_flags,omitempty"`
	System_include_flags []string   `json:"system_include_flags,omitempty"`
	Variables            ccVariable `json:"variables,omitempty"`
	Module_name          string     `json:"module_name,omitempty"`
}

type ccVariable struct {
	Global_flags         map[string]map[string]ccParameters `json:"global_flags,omitempty"`
	C_flags              map[string]map[string]ccParameters `json:"c_flags,omitempty"`
	C_only_flags         map[string]map[string]ccParameters `json:"c_only_flags,omitempty"`
	Cpp_flags            map[string]map[string]ccParameters `json:"cpp_flags,omitempty"`
	System_include_flags map[string]map[string]ccParameters `json:"system_include_flags,omitempty"`
}

type ccParameters struct {
	HeaderSearchPath       []string          `json:"header_search_path,omitempty"`
	SystemHeaderSearchPath []string          `json:"system_search_path,omitempty"`
	FlagParameters         []string          `json:"flag,omitempty"`
	SysRoot                string            `json:"system_root,omitempty"`
	RelativeFilePathFlags  map[string]string `json:"relative_file_path,omitempty"`
}

func (c *ccdepsGeneratorSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	if !ctx.Config().IsEnvTrue(envVariableCollectCCDeps) {
		return
	}

	moduleInfos := map[string]ccIdeInfo{}

	// Track which projects have already had CMakeLists.txt generated to keep the first
	// variant for each project.
	seenProjects := map[string]bool{}

	writeClangPathsToConfig(ctx)

	ctx.VisitAllModules(func(module android.Module) {
		if ccModule, ok := module.(*Module); ok {
			if compiledModule, ok := ccModule.compiler.(CompiledInterface); ok {
				generateCLionProjectData(compiledModule, ctx, ccModule, seenProjects, moduleInfos)
			}
		}
	})

	ccfpath := android.PathForOutput(ctx, ccdepsJsonFileName).String()
	err := createJsonFile(moduleInfos, ccfpath)
	if err != nil {
		ctx.Errorf(err.Error())
	}
}

func writeClangPathsToConfig(ctx android.SingletonContext) error {
	config_path := android.PathForOutput(ctx, nativeConfigFileName).String()
	f, err := os.Create(config_path)
	if err != nil {
		return fmt.Errorf("Failed to open file: %s, relative: %v", config_path, err)
	}
	defer f.Close()

	aidegenConfigVars := map[string]string{}
	pathToCC, _ := evalVariable(ctx, "${config.ClangBin}/")
	aidegenConfigVars[cLang] = fmt.Sprintf("%s%s", buildCMakePath(pathToCC), cLang)
	aidegenConfigVars[cppLang] = fmt.Sprintf("%s%s", buildCMakePath(pathToCC), cppLang)

	buf, err := json.MarshalIndent(aidegenConfigVars, "", "    ")
	if err != nil {
		return fmt.Errorf("Struct to byte buffer failed, relative: %v", err)
	}

	_, err = fmt.Fprintf(f, string(buf))
	if err != nil {
		return fmt.Errorf("Write file failed: %s, relative: %v", config_path, err)
	}
	return nil
}

func filterCompilerVariables(params []string, ctx android.SingletonContext) []string {
	variableParameters := []string{}
	for i := 0; i < len(params); i++ {
		param := params[i]
		if strings.HasPrefix(param, "$") {
			variableParameters = append(variableParameters, param)
		}
	}
	return variableParameters
}

func convertCompilerParametersToccParameters(cmd string, params compilerParameters) map[string]ccParameters {
	ccParams := ccParameters{}
	ccParams.HeaderSearchPath = append(ccParams.HeaderSearchPath, params.headerSearchPath...)
	ccParams.SystemHeaderSearchPath = append(ccParams.SystemHeaderSearchPath, params.systemHeaderSearchPath...)
	ccParams.FlagParameters = append(ccParams.FlagParameters, params.flags...)
	ccParams.SysRoot = params.sysroot
	ccParams.RelativeFilePathFlags = map[string]string{}
	for i := 0; i < len(params.relativeFilePathFlags); i++ {
		param := params.relativeFilePathFlags[i]
		ccParams.RelativeFilePathFlags[param.flag] = param.relativeFilePath
	}
	ccParamMaps := map[string]ccParameters{}
	ccParamMaps[cmd] = ccParams
	return ccParamMaps
}

func parseCompilerVariables(params []string, ctx android.SingletonContext) map[string]map[string]ccParameters {
	ccVariables := map[string]map[string]ccParameters{}
	variableParameters := filterCompilerVariables(params, ctx)
	for i := 0; i < len(variableParameters); i++ {
		param, nextParam := "", ""
		if i != len(variableParameters)-1 {
			param, nextParam = variableParameters[i], variableParameters[i+1]
		} else {
			param, nextParam = variableParameters[i], ""
		}
		skip, cmd, varComilerParameters := parseVariableCompilerParameter(param, nextParam, ctx)
		if skip {
			i = i + 1
		}
		ccVariables[param] = convertCompilerParametersToccParameters(cmd, varComilerParameters)
	}
	return ccVariables
}

func parseVariableCompilerParameter(param, nextParam string, ctx android.SingletonContext) (bool, string, compilerParameters) {
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
				param, nextParam := "", ""
				if i != len(params)-1 {
					param, nextParam = params[i], params[i+1]
				} else {
					param, nextParam = params[i], ""
				}
				skip, _, paramsFromVar := parseVariableCompilerParameter(param, nextParam, ctx)
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
		compilerParams.flags = append(compilerParams.flags, param)
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

func generateCLionProjectData(compiledModule CompiledInterface, ctx android.SingletonContext,
	ccModule *Module, seenProjects map[string]bool, moduleInfos map[string]ccIdeInfo) {
	srcs := compiledModule.Srcs()
	if len(srcs) == 0 {
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
	dpInfo.Global_flags = append(dpInfo.Global_flags, ccModule.flags.GlobalFlags...)
	dpInfo.C_flags = append(dpInfo.C_flags, ccModule.flags.CFlags...)
	dpInfo.C_only_flags = append(dpInfo.C_only_flags, ccModule.flags.ConlyFlags...)
	dpInfo.Cpp_flags = append(dpInfo.Cpp_flags, ccModule.flags.CppFlags...)
	dpInfo.System_include_flags = append(dpInfo.System_include_flags, ccModule.flags.SystemIncludeFlags...)

	dpInfo.Path = android.FirstUniqueStrings(dpInfo.Path)
	dpInfo.Srcs = android.FirstUniqueStrings(dpInfo.Srcs)
	dpInfo.Global_flags = android.FirstUniqueStrings(dpInfo.Global_flags)
	dpInfo.C_flags = android.FirstUniqueStrings(dpInfo.C_flags)
	dpInfo.C_only_flags = android.FirstUniqueStrings(dpInfo.C_only_flags)
	dpInfo.Cpp_flags = android.FirstUniqueStrings(dpInfo.Cpp_flags)
	dpInfo.System_include_flags = android.FirstUniqueStrings(dpInfo.System_include_flags)

	// It's better to compile variables here than in Python.
	dpInfo.Variables.Global_flags = parseCompilerVariables(dpInfo.Global_flags, ctx)
	dpInfo.Variables.C_flags = parseCompilerVariables(dpInfo.C_flags, ctx)
	dpInfo.Variables.C_only_flags = parseCompilerVariables(dpInfo.C_only_flags, ctx)
	dpInfo.Variables.Cpp_flags = parseCompilerVariables(dpInfo.Cpp_flags, ctx)
	dpInfo.Variables.System_include_flags = parseCompilerVariables(dpInfo.System_include_flags, ctx)

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

func createJsonFile(moduleInfos map[string]ccIdeInfo, ccfpath string) error {
	file, err := os.Create(ccfpath)
	if err != nil {
		return fmt.Errorf("Failed to create file: %s, relative: %v", ccdepsJsonFileName, err)
	}
	defer file.Close()
	moduleInfos = sortMap(moduleInfos)
	buf, err := json.MarshalIndent(moduleInfos, "", "\t")
	if err != nil {
		return fmt.Errorf("Write file failed: %s, relative: %v", ccdepsJsonFileName, err)
	}
	fmt.Fprintf(file, string(buf))
	return nil
}
