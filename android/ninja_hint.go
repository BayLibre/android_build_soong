// Copyright 2023 Google Inc. All rights reserved.
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
	"fmt"
	"strings"

	"android/soong/android/allowlists"
)

// This singleton collects huges modules' outputs into to a ninja weight file.
// The info file is generated in $OUT/soong/ninja_weight_list.

func init() {
	RegisterSingletonType("ninja_hint_generator", ninjaHintGeneratorSingletonFactory)
}

func ninjaHintGeneratorSingletonFactory() Singleton {
	return &ninjaHintGeneratorSingleton{}
}

type ninjaHintGeneratorSingleton struct {
}

const (
	ninjaWeightListFileName = ".ninja_weight_list"
)

func (c *ninjaHintGeneratorSingleton) GenerateBuildActions(ctx SingletonContext) {
	pathsHint := make(map[string]int)
	ctx.VisitAllModules(func(module Module) {
		if val, ok := allowlists.HugeModulesMap[module.Name()]; ok {
			for _, output := range module.base().outputs {
				pathsHint[output] = val
			}
		}
	})

	weightListFile := PathForOutput(ctx, ninjaWeightListFileName)
	err := createNinjaWeightFile(pathsHint, weightListFile)
	if err != nil {
		ctx.Errorf(err.Error())
	}

	// This is necessary to satisfy the dangling rules check as this file is written by Soong rather than a rule.
	ctx.Build(pctx, BuildParams{
		Rule:   Touch,
		Output: weightListFile,
	})
}

func createNinjaWeightFile(hint map[string]int, out WritablePath) error {
	var outputBuilder strings.Builder
	for k, v := range hint {
		outputBuilder.WriteString(fmt.Sprintf("%s,%d\n", k, v))
	}
	err := WriteFileToOutputDir(out, []byte(outputBuilder.String()), 0644)
	if err != nil {
		return fmt.Errorf("Could not write ninja weight list file %s", err)
	}
	return nil
}
