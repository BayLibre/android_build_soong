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

	"android/soong/android"
)

// This singleton collects cc source and flags into to a json file. It does so for generating
// CMakeLists.txt project files by these data when either make, mm, mma, mmm or mmma is called.
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

func (c *ccdepsGeneratorSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	if !ctx.Config().IsEnvTrue(envVariableCollectCCDeps) {
		return
	}

	moduleInfos := make(map[string]android.CCIdeInfo)

	ctx.VisitAllModules(func(module android.Module) {
		if !module.Enabled() {
			return
		}

		ideInfoProvider, ok := module.(android.CCIDEInfo)
		if !ok {
			return
		}
		name := ideInfoProvider.BaseModuleName()

		dpInfo := moduleInfos[name]
		ideInfoProvider.CCIDEInfo(&dpInfo)
		dpInfo.CCSrcs = android.FirstUniqueStrings(dpInfo.CCSrcs)
		dpInfo.CC_global_flags = android.FirstUniqueStrings(dpInfo.CC_global_flags)
		_, dpInfo.CC_global_flags = android.RemoveFromList("", dpInfo.CC_global_flags)
		dpInfo.CC_cflags = android.FirstUniqueStrings(dpInfo.CC_cflags)
		_, dpInfo.CC_cflags = android.RemoveFromList("", dpInfo.CC_cflags)
		dpInfo.CC_conlyflags = android.FirstUniqueStrings(dpInfo.CC_conlyflags)
		_, dpInfo.CC_conlyflags = android.RemoveFromList("", dpInfo.CC_conlyflags)
		dpInfo.CC_cppflags = android.FirstUniqueStrings(dpInfo.CC_cppflags)
		_, dpInfo.CC_cppflags = android.RemoveFromList("", dpInfo.CC_cppflags)
		dpInfo.CC_system_includeflags = android.FirstUniqueStrings(dpInfo.CC_system_includeflags)
		_, dpInfo.CC_system_includeflags = android.RemoveFromList("", dpInfo.CC_system_includeflags)
		moduleInfos[name] = dpInfo
	})

	ccfpath := android.PathForOutput(ctx, ccdepsJsonFileName).String()
	err := createJsonFile(moduleInfos, ccfpath)
	if err != nil {
		ctx.Errorf(err.Error())
	}
}

func createJsonFile(moduleInfos map[string]android.CCIdeInfo, ccfpath string) error {
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
