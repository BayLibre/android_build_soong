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

package etc

import (
	"android/soong/android"

	"github.com/google/blueprint/proptools"
)

func init() {
	android.RegisterModuleType("bootconfig", BootconfigModuleFactory)
}

type bootconfigProperty struct {
	Boot_config      []string
	Boot_config_file *string `android:"path"`
}

type BootconfigModule struct {
	android.ModuleBase

	properties bootconfigProperty

	outputPath android.WritablePath
}

func BootconfigModuleFactory() android.Module {
	module := &BootconfigModule{}
	module.AddProperties(&module.properties)
	android.InitAndroidArchModule(module, android.DeviceSupported, android.MultilibCommon)
	return module
}

func (m *BootconfigModule) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	bootConfig := m.properties.Boot_config
	bootConfigFileStr := proptools.String(m.properties.Boot_config_file)
	if len(bootConfig) == 0 && len(bootConfigFileStr) == 0 {
		return
	}

	var bootConfigFile android.Path
	if len(bootConfigFileStr) > 0 {
		bootConfigFile = android.PathForModuleSrc(ctx, bootConfigFileStr)
	}

	outputPath := android.PathForModuleOut(ctx, ctx.ModuleName(), "vendor-bootconfig.img")

	builder := android.NewRuleBuilder(pctx, ctx)
	builder.Command().Textf("rm -f %s", outputPath.String())
	for _, config := range bootConfig {
		builder.Command().
			Text("echo ").
			Textf("%s\n ", config).
			Text(">> ").
			Text(outputPath.String()).
			ImplicitOutput(outputPath)
	}

	if bootConfigFile != nil {
		builder.Command().Textf("cat ").Input(bootConfigFile).Text(">> ").Output(outputPath)
	}

	builder.Build("building vendor-bootconfig.img", "building vendor-bootconfig.img")
	ctx.SetOutputFiles(android.Paths{outputPath}, "")
}
