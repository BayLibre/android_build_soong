// Copyright 2021 Google Inc. All rights reserved.
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

package java

import (
	"path/filepath"
	"strings"

	"android/soong/android"
	"android/soong/dexpreopt"

	"github.com/google/blueprint/pathtools"
)

func init() {
	RegisterDexpreoptCheckBuildComponents(android.InitRegistrationContext)
}

func RegisterDexpreoptCheckBuildComponents(ctx android.RegistrationContext) {
	ctx.RegisterSingletonModuleType("dexpreopt_systemserver_check", dexpreoptSystemserverCheckFactory)
}

type dexpreoptSystemserverCheck struct {
	android.SingletonModuleBase

	outputFileOnHost    android.OutputPath
	installFileOnDevice android.InstallPath
}

func dexpreoptSystemserverCheckFactory() android.SingletonModule {
	m := &dexpreoptSystemserverCheck{}
	android.InitAndroidArchModule(m, android.DeviceSupported, android.MultilibCommon)
	return m
}

func getInstallPath(ctx android.ModuleContext, location string) android.InstallPath {
	return android.PathForModuleInPartitionInstall(
		ctx, "", strings.TrimPrefix(location, "/")).ToMakePath()
}

// Creates a build rule that depends on all the compilation artifacts of system server JARs, with
// will fail at ninja phase if any of the artifacts is missing. When this rule fails, it means that
// dexpreopting is not working for some system server JARs and needs to be fixed.
func (m *dexpreoptSystemserverCheck) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	m.outputFileOnHost = android.PathForOutput(ctx, "dexpreopt_systemserver_check.stamp")
	m.installFileOnDevice = android.PathForModuleInPartitionInstall(
		ctx, "system", "framework", m.outputFileOnHost.Base())

	global := dexpreopt.GetGlobalConfig(ctx)
	targets := ctx.Config().Targets[android.Android]
	var artifacts android.Paths
	if !global.DisablePreopt && len(targets) > 0 {
		systemServerJars := dexpreopt.AllSystemServerJars(ctx, global)
		for _, jar := range systemServerJars.CopyOfJars() {
			dexLocation := dexpreopt.GetSystemServerDexLocation(global, jar)
			odexLocation := dexpreopt.ToOdexPath(dexLocation, targets[0].Arch.ArchType)
			odexPath := getInstallPath(ctx, odexLocation)
			vdexPath := getInstallPath(ctx, pathtools.ReplaceExtension(odexLocation, "vdex"))
			artifacts = append(artifacts, odexPath, vdexPath)
		}
	}

	ctx.Build(pctx, android.BuildParams{
		Rule:      android.Touch,
		Output:    m.outputFileOnHost,
		Implicits: artifacts,
	})
}

// dexpreoptSystemserverCheck does not need any information from modules, so rules can be generated
// in GenerateAndroidBuildActions, and this method can be empty.
func (m *dexpreoptSystemserverCheck) GenerateSingletonBuildActions(ctx android.SingletonContext) {
}

// Creates a make entry to install the output file to the device. The file is empty and not used by
// anyone. It is added for creating a dependency from the system image to the file, so that the
// check is always run when a system image is being built.
func (m *dexpreoptSystemserverCheck) AndroidMkEntries() []android.AndroidMkEntries {
	installPath := m.installFileOnDevice.ToMakePath()

	return []android.AndroidMkEntries{android.AndroidMkEntries{
		Class:      "ETC",
		OutputFile: android.OptionalPathForPath(m.outputFileOnHost),
		ExtraEntries: []android.AndroidMkExtraEntriesFunc{
			func(ctx android.AndroidMkExtraEntriesContext, entries *android.AndroidMkEntries) {
				entries.SetString("LOCAL_MODULE_PATH", filepath.Dir(installPath.String()))
				entries.SetString("LOCAL_INSTALLED_MODULE_STEM", installPath.Base())
			},
		},
	}}
}
