// Copyright 2024 Google Inc. All rights reserved.
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

import "encoding/json"

func init() {
	ctx := InitRegistrationContext
	ctx.RegisterModuleType("product_config", productConfigFactory)
}

type productConfigModule struct {
	ModuleBase

	outputFilePath OutputPath
}

func (p *productConfigModule) GenerateAndroidBuildActions(ctx ModuleContext) {
	if ctx.ModuleName() != "product_config" || ctx.ModuleDir() != "build/soong" {
		ctx.ModuleErrorf("There can only be one product_config module in build/soong")
		return
	}
	p.outputFilePath = PathForModuleOut(ctx, p.Name()+".json").OutputPath
	productVariablesJson, err := json.Marshal(ctx.Config().config.productVariables)
	if err != nil {
		ctx.ModuleErrorf("Cannot marshal product variables: %w", err)
		return
	}
	mergedData := make(map[string]interface{})
	if err = json.Unmarshal(productVariablesJson, &mergedData); err != nil {
		ctx.ModuleErrorf("Cannot unmarshal product variables: %w", err)
		return
	}
	for k, v := range ctx.Config().config.extraVariables {
		mergedData[k] = v
	}

	mergedJson, err := json.Marshal(mergedData)
	if err != nil {
		ctx.ModuleErrorf("Cannot marshal merged variables: %w", err)
		return
	}

	WriteFileRule(ctx, p.outputFilePath, string(mergedJson))

	ctx.SetOutputFiles(Paths{p.outputFilePath}, "")
}

// product_config module exports product variables and extra variables as a JSON file.
func productConfigFactory() Module {
	module := &productConfigModule{}
	InitAndroidModule(module)
	return module
}
