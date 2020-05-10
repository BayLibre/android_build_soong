// Copyright 2020 Google Inc. All rights reserved.
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

package android

import (
	"encoding/json"
	"fmt"
)

// This singleton generates android non java dependency into to a json file. It does so for each
// blueprint Android.bp resulting in a non java.Module when either make, mm, mma, mmm or mmma is
// called. Dependency info file is generated in $OUT/module_bp_non_java_depend.json.

func init() {
	RegisterSingletonType("nonjdeps_generator", nonjDepsGeneratorSingleton)
}

func nonjDepsGeneratorSingleton() Singleton {
	return &nonjdepsGeneratorSingleton{}
}

type nonjdepsGeneratorSingleton struct {
}

const (
	// Environment variables used to modify behavior of this singleton.
	envVariableCollectNonJavaDeps = "SOONG_COLLECT_NON_JAVA_DEPS"
	jdepsJsonFileName             = "module_bp_non_java_deps.json"
)

func (j *nonjdepsGeneratorSingleton) GenerateBuildActions(ctx SingletonContext) {
	if !ctx.Config().IsEnvTrue(envVariableCollectNonJavaDeps) {
		return
	}

	moduleInfos := make(map[string]IdeInfoNonJava)

	ctx.VisitAllModules(func(module Module) {
		if !module.Enabled() {
			return
		}

		ideInfoProvider, ok := module.(IDEInfoNonJava)
		if !ok {
			return
		}
		name := ideInfoProvider.BaseModuleName()
		dpInfo := moduleInfos[name]
		ideInfoProvider.IDEInfoNonJava(&dpInfo)
		dpInfo.Deps = FirstUniqueStrings(dpInfo.Deps)
		dpInfo.Srcs = FirstUniqueStrings(dpInfo.Srcs)
		dpInfo.Paths = FirstUniqueStrings(dpInfo.Paths)
		moduleInfos[name] = dpInfo
	})

	jfpath := PathForOutput(ctx, jdepsJsonFileName)
	err := createJsonFile(moduleInfos, jfpath)
	if err != nil {
		ctx.Errorf(err.Error())
	}
}

func createJsonFile(moduleInfos map[string]IdeInfoNonJava, jfpath WritablePath) error {
	buf, err := json.MarshalIndent(moduleInfos, "", "\t")
	if err != nil {
		return fmt.Errorf("JSON marshal of java deps failed: %s", err)
	}
	err = WriteFileToOutputDir(jfpath, buf, 0666)
	if err != nil {
		return fmt.Errorf("Writing java deps to %s failed: %s", jfpath.String(), err)
	}
	return nil
}
