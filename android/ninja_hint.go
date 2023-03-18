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
	"encoding/json"
	"fmt"

	"android/soong/android/allowlists"
)

// This singleton collects huges modules' outputs and flags into to a json file.
// The info file is generated in $OUT/ninja_hint.json.

func init() {
	RegisterSingletonType("ninja_hint_generator", ninjaHintGeneratorSingletonFactory)
}

func ninjaHintGeneratorSingletonFactory() Singleton {
	return &ninjaHintGeneratorSingleton{}
}

type ninjaHintGeneratorSingleton struct {
}

const (
	ninjaHintJsonFile = "ninja_hint.json"
)

func (c *ninjaHintGeneratorSingleton) GenerateBuildActions(ctx SingletonContext) {
	pathsHint := []string{}
	ctx.VisitAllModules(func(module Module) {
		if InList(module.Name(), allowlists.HugeModulesList) {
			pathsHint = append(pathsHint, module.base().outputs...)
		}
	})

	ninjaHint := PathForOutput(ctx, ninjaHintJsonFile)
	err := createJsonFile(pathsHint, ninjaHint)
	if err != nil {
		ctx.Errorf(err.Error())
	}

	// This is necessary to satisfy the dangling rules check as this file is written by Soong rather than a rule.
	ctx.Build(pctx, BuildParams{
		Rule:   Touch,
		Output: ninjaHint,
	})
}

func createJsonFile(hint []string, out WritablePath) error {
	buf, err := json.MarshalIndent(hint, "", "\t")
	if err != nil {
		return fmt.Errorf("JSON marshal of ninja hint failed: %s", err)
	}
	err = WriteFileToOutputDir(out, buf, 0666)
	if err != nil {
		return fmt.Errorf("Writing ninja hint to %s failed: %s", out.String(), err)
	}
	return nil
}
