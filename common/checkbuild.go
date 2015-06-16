// Copyright 2015 Google Inc. All rights reserved.
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

package common

import (
	"path/filepath"

	"github.com/google/blueprint"
)

func CheckbuildSingleton() blueprint.Singleton {
	return &checkbuildSingleton{}
}

type checkbuildSingleton struct{}

func (c *checkbuildSingleton) GenerateBuildActions(ctx blueprint.SingletonContext) {
	deps := []string{}

	dirModules := make(map[string][]string)

	ctx.VisitAllModules(func(module blueprint.Module) {
		if a, ok := module.(AndroidModule); ok {
			moduleTarget := a.base().moduleTarget
			if moduleTarget != "" {
				blueprintDir := a.base().blueprintDir
				deps = append(deps, moduleTarget)
				dirModules[blueprintDir] = append(dirModules[blueprintDir], moduleTarget)
			}
		}
	})

	ctx.Build(pctx, blueprint.BuildParams{
		Rule:      blueprint.Phony,
		Outputs:   []string{"checkbuild"},
		Implicits: deps,
		// HACK: checkbuild should be an optional build, but force it enabled for now
		//Optional:  true,
	})

	dirs := sortedKeys(dirModules)
	for _, dir := range dirs {
		ctx.Build(pctx, blueprint.BuildParams{
			Rule:      blueprint.Phony,
			Outputs:   []string{filepath.Join("mm", dir)},
			Implicits: dirModules[dir],
			Optional:  true,
		})
	}
}
