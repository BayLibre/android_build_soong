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
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"

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
	cLang                    = "clang"
	cppLang                  = "clang++"
)

type ccIdeInfo struct {
	Path                 []string `json:"path,omitempty"`
	Srcs                 []string `json:"srcs,omitempty"`
	Global_flags         []string `json:"global_flags,omitempty"`
	C_flags              []string `json:"c_flags,omitempty"`
	C_only_flags         []string `json:"c_only_flags,omitempty"`
	Cpp_flags            []string `json:"cpp_flags,omitempty"`
	System_include_flags []string `json:"system_include_flags,omitempty"`
	Module_name          string   `json:"module_name,omitempty"`
}

func (c *ccdepsGeneratorSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	if !ctx.Config().IsEnvTrue(envVariableCollectCCDeps) {
		return
	}

	moduleInfos := map[string]ccIdeInfo{}

	// Track which projects have already had CMakeLists.txt generated to keep the first
	// variant for each project.
	seenProjects := map[string]bool{}

	writeCLangPathsToConfig(ctx)

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

func writeCLangPathsToConfig(ctx android.SingletonContext) error {
	home := ctx.Config().Getenv("HOME")
	if home == "" {
		return nil
	}

	config_path := filepath.Join(home, ".config", "asuite", "aidegen", "aidegen.config")
	aidegenConfigVars := map[string]string{}
	byt, err := ioutil.ReadFile(config_path)
	if err != nil {
		return fmt.Errorf("Failed to read file: %s, relative: %v", config_path, err)
	}

	err = json.Unmarshal(byt, &aidegenConfigVars)
	if err != nil {
		return fmt.Errorf("Failed to unmarshal file contents to json: %s, relative: %v", config_path, err)
	}
	pathToCC, _ := evalVariable(ctx, "${config.ClangBin}/")
	aidegenConfigVars[cLang] = fmt.Sprintf("%s%s", buildCMakePath(pathToCC), cLang)
	aidegenConfigVars[cppLang] = fmt.Sprintf("%s%s", buildCMakePath(pathToCC), cppLang)

	f, err := os.Create(config_path)
	if err != nil {
		return fmt.Errorf("Failed to open file: %s, relative: %v", config_path, err)
	}
	defer f.Close()

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
	_, dpInfo.Global_flags = android.RemoveFromList("", dpInfo.Global_flags)
	dpInfo.C_flags = android.FirstUniqueStrings(dpInfo.C_flags)
	_, dpInfo.C_flags = android.RemoveFromList("", dpInfo.C_flags)
	dpInfo.C_only_flags = android.FirstUniqueStrings(dpInfo.C_only_flags)
	_, dpInfo.C_only_flags = android.RemoveFromList("", dpInfo.C_only_flags)
	dpInfo.Cpp_flags = android.FirstUniqueStrings(dpInfo.Cpp_flags)
	_, dpInfo.Cpp_flags = android.RemoveFromList("", dpInfo.Cpp_flags)
	dpInfo.System_include_flags = android.FirstUniqueStrings(dpInfo.System_include_flags)
	_, dpInfo.System_include_flags = android.RemoveFromList("", dpInfo.System_include_flags)
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
