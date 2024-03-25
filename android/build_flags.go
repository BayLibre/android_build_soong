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

import (
	"encoding/json"
)

func init() {
	registerReleaseFlagsModuleType(InitRegistrationContext)
}

func registerReleaseFlagsModuleType(ctx RegistrationContext) {
	ctx.RegisterModuleType("release_flags_json", releaseFlagsFactory)
}

var PrepareForTestWithReleaseFlags = FixtureRegisterWithContext(registerReleaseFlagsModuleType)

type releaseFlags struct {
	ModuleBase

	jsonBytes  []byte
	outputPath OutputPath
}

func releaseFlagsFactory() Module {
	module := &releaseFlags{}
	InitAndroidArchModule(module, DeviceSupported, MultilibCommon)
	return module
}

type BuildFlagData struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Set      string `json:"set"`
	Default  string `json:"default"`
	Declared string `json:"declared"`
}

type releaseFlagsJson struct {
	Flags []BuildFlagData `json:"flags,omitempty"`
}

func GenerateBuildFlagsMapForTest(m map[string]string) map[string]BuildFlagData {
	buildFlag := make(map[string]BuildFlagData)
	for k, v := range m {
		buildFlag[k] = BuildFlagData{k, v, "", "", ""}
	}
	return buildFlag
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

func (m *releaseFlags) generateFlagsJson(ctx ModuleContext) []byte {
	partition := m.getPartitionString()
	buildFlags, ok := ctx.Config().GetBuildFlagsInPartition(partition)
	if !ok {
		ctx.ModuleErrorf("Partition %q does not have release flags", partition)
	}

	var flags []BuildFlagData
	for _, buildFlag := range buildFlags {
		if flag, ok := ctx.Config().GetBuildFlag(buildFlag); ok {
			flags = append(flags, flag)
		}
	}

	jsonData := releaseFlagsJson{flags}
	result, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		ctx.ModuleErrorf("json marshal failed: %#v", err)
	}
	return result
}

func (m *releaseFlags) GenerateAndroidBuildActions(ctx ModuleContext) {
	filename := "build_flags.json"
	m.jsonBytes = m.generateFlagsJson(ctx)
	m.outputPath = PathForModuleOut(ctx, filename).OutputPath
	WriteFileRule(ctx, m.outputPath, string(m.jsonBytes))
	installPath := PathForModuleInstall(ctx, "etc")
	ctx.InstallFile(installPath, filename, m.outputPath)
}

func (m *releaseFlags) AndroidMkEntries() []AndroidMkEntries {
	return []AndroidMkEntries{AndroidMkEntries{
		Class:      "ETC",
		OutputFile: OptionalPathForPath(m.outputPath),
	}}
}
