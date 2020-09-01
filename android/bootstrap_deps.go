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

	"github.com/google/blueprint"
	"github.com/google/blueprint/bootstrap"
)

// This singleton generates bootstrap_go_package modules' dependency into to a json file. It does
// so for each blueprint Android.bp resulting in a bootstrap_go_package type module when either
// make, mm, mma, mmm or mmma is called. Dependency info file is generated in
// $OUT/module_bp_bootstrap_deps.json.

func init() {
	RegisterSingletonType("bootstrap_deps_generator", bootstrapDepsGeneratorSingleton)
}

func bootstrapDepsGeneratorSingleton() Singleton {
	return &bootstrapdepsGeneratorSingleton{}
}

type bootstrapdepsGeneratorSingleton struct {
}

const (
	// Environment variables used to modify behavior of this singleton.
	envVariableCollectBootstrapDeps = "SOONG_COLLECT_BOOTSTRAP_DEPS"
	bpdepsJsonFileName              = "module_bp_bootstrap_deps.json"
)

func (j *bootstrapdepsGeneratorSingleton) GenerateBuildActions(ctx SingletonContext) {
	if !ctx.Config().IsEnvTrue(envVariableCollectBootstrapDeps) {
		return
	}

	moduleInfos := make(map[string]bootstrap.IdeInfoBootstrap)
	ctx.VisitAllModulesBlueprint(func(module blueprint.Module) {
		ideInfoProvider, ok := module.(bootstrap.IDEInfoBootstrap)
		if !ok {
			return
		}
		name := ctx.ModuleName(module)
		dpInfo := moduleInfos[name]
		ideInfoProvider.CollectIdeInfoBootstrap(&dpInfo)
		dpInfo.Paths = append(dpInfo.Paths, ctx.ModuleDir(module))
		dpInfo.Deps = FirstUniqueStrings(dpInfo.Deps)
		dpInfo.Paths = FirstUniqueStrings(dpInfo.Paths)
		moduleInfos[name] = dpInfo
	})
	jfpath := PathForOutput(ctx, bpdepsJsonFileName)
	createJsonFile(moduleInfos, jfpath)
}

func createJsonFile(moduleInfos map[string]bootstrap.IdeInfoBootstrap, jfpath WritablePath) error {
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
