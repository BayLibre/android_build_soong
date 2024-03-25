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

package packaging

import (
	"android/soong/android"
	"encoding/json"
)

func init() {
	registerReleaseFlagsModuleType(android.InitRegistrationContext)
}

func registerReleaseFlagsModuleType(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("release_flags_json", releaseFlagsFactory)
}

type releaseFlags struct {
	android.ModuleBase

	jsonBytes  []byte
	outputPath android.OutputPath
}

func releaseFlagsFactory() android.Module {
	module := &releaseFlags{}
	android.InitAndroidArchModule(module, android.DeviceSupported, android.MultilibCommon)
	return module
}

type releaseFlagData struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Set      string `json:"set"`
	Default  string `json:"default"`
	Declared string `json:"declared"`
}

type releaseFlagsJson struct {
	Flags []releaseFlagData `json:"flags,omitempty"`
}

func (m *releaseFlags) getPartitionString() string {
	partition := "system"
	if m.SocSpecific() {
		partition = "vendor"
	} else if m.ProductSpecific() {
		partition = "product"
	} else if m.SystemExtSpecific() {
		partition = "system_ext"
	} else if m.DeviceSpecific() {
		partition = "odm"
	}
	return partition
}

func (m *releaseFlags) generateFlagsJson(ctx android.ModuleContext) []byte {
	partition := m.getPartitionString()
	modules, ok := ctx.Config().GetBuildFlagsInPartition(partition)
	if !ok {
		ctx.ModuleErrorf("Partition %q does not have release flags", partition)
	}

	var flags []releaseFlagData
	for _, module := range modules {
		val, _ := ctx.Config().GetBuildFlag(module)
		set, _ := ctx.Config().GetBuildFlagExtra(module, "set")
		def, _ := ctx.Config().GetBuildFlagExtra(module, "default")
		dec, _ := ctx.Config().GetBuildFlagExtra(module, "declared")
		flags = append(flags, releaseFlagData{module, val, set, def, dec})
	}

	jsonData := releaseFlagsJson{flags}
	result, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		ctx.ModuleErrorf("json marshal failed: %#v", err)
	}
	return result
}

func (m *releaseFlags) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	filename := "build_flags.json"
	m.jsonBytes = m.generateFlagsJson(ctx)
	m.outputPath = android.PathForModuleOut(ctx, filename).OutputPath
	android.WriteFileRule(ctx, m.outputPath, string(m.jsonBytes))
	installPath := android.PathForModuleInstall(ctx, "etc")
	ctx.InstallFile(installPath, filename, m.outputPath)
}

func (m *releaseFlags) AndroidMkEntries() []android.AndroidMkEntries {
	return []android.AndroidMkEntries{android.AndroidMkEntries{
		Class:      "ETC",
		OutputFile: android.OptionalPathForPath(m.outputPath),
	}}
}
