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
	// Environment variables used to modify behavior of this singleton.
	envVariableCollectCCDeps = android.EnvVariableCollectJavaDeps
	ccdepsJsonFileName       = "module_bp_cc_deps.json"
)

type ccIdeInfo struct {
	SrcPath                []string `json:"path,omitempty"`
	CCSrcs                 []string `json:"cc_srcs,omitempty"`
	CC_global_flags        []string `json:"cc_global_flags,omitempty"`
	CC_cflags              []string `json:"cc_cflags,omitempty"`
	CC_conlyflags          []string `json:"cc_conlyflags,omitempty"`
	CC_cppflags            []string `json:"cc_cppflags,omitempty"`
	CC_system_includeflags []string `json:"cc_system_includeflags,omitempty"`
}

func (c *ccdepsGeneratorSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	if !ctx.Config().IsEnvTrue(envVariableCollectCCDeps) {
		return
	}

	moduleInfos := make(map[string]ccIdeInfo)

	seenProjects := map[string]bool{}

	pathToCC, _ := evalVariable(ctx, "${config.ClangBin}/")
	dpInfo := moduleInfos["clang"]
	dpInfo.SrcPath = append(dpInfo.SrcPath, fmt.Sprintf("\"%s%s\"", buildCMakePath(pathToCC), "clang"))
	moduleInfos["clang"] = dpInfo
	dpInfocpp := moduleInfos["clang++"]
	dpInfocpp.SrcPath = append(dpInfocpp.SrcPath, fmt.Sprintf("\"%s%s\"", buildCMakePath(pathToCC), "clang++"))
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
	dpInfo.SrcPath = append(dpInfo.SrcPath, path.Dir(ctx.BlueprintFile(ccModule)))
	for _, src := range srcs {
		dpInfo.CCSrcs = append(dpInfo.CCSrcs, fmt.Sprintf("${ANDROID_ROOT}/%s", src.String()))
	}
	dpInfo.CC_global_flags = append(dpInfo.CC_global_flags, ccModule.flags.GlobalFlags...)
	dpInfo.CC_cflags = append(dpInfo.CC_cflags, ccModule.flags.CFlags...)
	dpInfo.CC_conlyflags = append(dpInfo.CC_conlyflags, ccModule.flags.ConlyFlags...)
	dpInfo.CC_cppflags = append(dpInfo.CC_cppflags, ccModule.flags.CppFlags...)
	dpInfo.CC_system_includeflags = append(dpInfo.CC_system_includeflags, ccModule.flags.SystemIncludeFlags...)
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
