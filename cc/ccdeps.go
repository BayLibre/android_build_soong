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
)

type ccIdeInfo struct {
	Path                 []string `json:"path,omitempty"`
	Srcs                 []string `json:"srcs,omitempty"`
	Global_flags         []string `json:"global_flags,omitempty"`
	C_flags              []string `json:"c_flags,omitempty"`
	C_only_flags         []string `json:"c_only_flags,omitempty"`
	Cpp_flags            []string `json:"cpp_flags,omitempty"`
	System_include_flags []string `json:"system_include_flags,omitempty"`
}

func (c *ccdepsGeneratorSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	if !ctx.Config().IsEnvTrue(envVariableCollectCCDeps) {
		return
	}

	moduleInfos := make(map[string]ccIdeInfo)

	seenProjects := map[string]bool{}

	pathToCC, _ := evalVariable(ctx, "${config.ClangBin}/")
	dpInfo := moduleInfos["clang"]
	dpInfo.Path = append(dpInfo.Path, fmt.Sprintf("\"%s%s\"", buildCMakePath(pathToCC), "clang"))
	moduleInfos["clang"] = dpInfo
	dpInfocpp := moduleInfos["clang++"]
	dpInfocpp.Path = append(dpInfocpp.Path, fmt.Sprintf("\"%s%s\"", buildCMakePath(pathToCC), "clang++"))
	moduleInfos["clang++"] = dpInfocpp

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

func generateCLionProjectData(compiledModule CompiledInterface, ctx android.SingletonContext,
	ccModule *Module, seenProjects map[string]bool, moduleInfos map[string]ccIdeInfo) {
	srcs := compiledModule.Srcs()
	if len(srcs) == 0 {
		return
	}

	clionproject_location := getCMakeListsForModule(ccModule, ctx)
	if seenProjects[clionproject_location] {
		return
	}

	seenProjects[clionproject_location] = true

	name := ccModule.ModuleBase.Name()
	dpInfo := moduleInfos[name]

	dpInfo.Path = append(dpInfo.Path, path.Dir(ctx.BlueprintFile(ccModule)))
	dpInfo.Srcs = append(dpInfo.Srcs, srcs.Strings()...)
	dpInfo.Global_flags = append(dpInfo.Global_flags, ccModule.flags.GlobalFlags...)
	dpInfo.C_flags = append(dpInfo.C_flags, ccModule.flags.CFlags...)
	dpInfo.C_only_flags = append(dpInfo.C_only_flags, ccModule.flags.ConlyFlags...)
	dpInfo.Cpp_flags = append(dpInfo.Cpp_flags, ccModule.flags.CppFlags...)
	dpInfo.System_include_flags = append(dpInfo.System_include_flags, ccModule.flags.SystemIncludeFlags...)

	dpInfo.Srcs = android.FirstUniqueStrings(dpInfo.Srcs)
	dpInfo.Global_flags = android.FirstUniqueStrings(dpInfo.Global_flags)
	_, dpInfo.Global_flags = android.RemoveFromList("", dpInfo.Global_flags)
	dpInfo.C_flags = android.FirstUniqueStrings(dpInfo.C_flags)
	_, dpInfo.C_flags = android.RemoveFromList("", dpInfo.C_flags)
	dpInfo.C_only_flags = android.FirstUniqueStrings(dpInfo.C_only_flags)
	_, dpInfo.C_only_flags = android.RemoveFromList("", dpInfo.C_only_flags)
	dpInfo.Cpp_flags = android.FirstUniqueStrings(dpInfo.Cpp_flags)
	_, dpInfo.Cpp_flags = android.RemoveFromList("", dpInfo.Cpp_flags)
	dpInfo.System_include_flags = android.FirstUniqueStrings(dpInfo.System_include_flags)
	_, dpInfo.System_include_flags = android.RemoveFromList("", dpInfo.System_include_flags)

	moduleInfos[name] = dpInfo
}

func createJsonFile(moduleInfos map[string]ccIdeInfo, ccfpath string) error {
	file, err := os.Create(ccfpath)
	if err != nil {
		return fmt.Errorf("Failed to create file: %s, relative: %v", ccdepsJsonFileName, err)
	}
	defer file.Close()
	buf, err := json.MarshalIndent(moduleInfos, "", "\t")
	if err != nil {
		return fmt.Errorf("Write file failed: %s, relative: %v", ccdepsJsonFileName, err)
	}
	fmt.Fprintf(file, string(buf))
	return nil
}
