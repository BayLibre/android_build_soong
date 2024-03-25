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

const (
	outJsonFileName = "build_flags.json"
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

	jsonString string
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

func (m *releaseFlags) generateFlagsJson(ctx ModuleContext) string {
	partition := m.PartitionTag(ctx.DeviceConfig())
	buildFlags, _ := ctx.Config().GetBuildFlagsInPartition(partition)

	jsonData := releaseFlagsJson{buildFlags}
	result, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		ctx.ModuleErrorf("json marshal failed: %#v", err)
	}
	return string(result)
}

func (m *releaseFlags) GenerateAndroidBuildActions(ctx ModuleContext) {
	m.jsonString = m.generateFlagsJson(ctx)
	m.outputPath = PathForModuleOut(ctx, outJsonFileName).OutputPath
	WriteFileRule(ctx, m.outputPath, m.jsonString)
	installPath := PathForModuleInstall(ctx, "etc")
	ctx.InstallFile(installPath, outJsonFileName, m.outputPath)
}

func (m *releaseFlags) AndroidMkEntries() []AndroidMkEntries {
	return []AndroidMkEntries{AndroidMkEntries{
		Class:      "ETC",
		OutputFile: OptionalPathForPath(m.outputPath),
	}}
}
