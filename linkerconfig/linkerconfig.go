// Copyright (C) 2020 The Android Open Source Project
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

package linkerconfig

import (
	"android/soong/android"
	"android/soong/etc"
	"strconv"

	"github.com/google/blueprint"
)

type linkerConfigProperties struct {
	// source linker configuration property file
	Src *string `android:"path"`

	// if module is installable (default is true)
	// installable should be marked as false for APEX configuration to avoid
	// conflicts of configuration on /system/etc directory.
	Installable *bool
}

type linkerConfig struct {
	android.ModuleBase
	properties linkerConfigProperties

	outputFilePath android.OutputPath
	installDirPath android.InstallPath
}

// Implement PrebuiltEtcModule interface to fit in APEX prebuilt list.
var _ etc.PrebuiltEtcModule = &linkerConfig{}

func (l *linkerConfig) BaseDir() string {
	return "etc"
}

func (l *linkerConfig) SubDir() string {
	return ""
}

func (l *linkerConfig) OutputFile() android.OutputPath {
	return l.outputFilePath
}

var (
	pctx = android.NewPackageContext("android/soong/linkerconfig")
)

func init() {
	pctx.HostBinToolVariable("conv_linker_config", "conv_linker_config")
	android.RegisterModuleType("linker_config", linkerConfigFactory)
}

var (
	linkerConfigRule = pctx.AndroidStaticRule("linkerConfigRule",
		blueprint.RuleParams{
			Command:     `rm -rf $out && ${conv_linker_config} proto -s $in -o $out`,
			CommandDeps: []string{"${conv_linker_config}"},
			Description: "convert ${in} => ${out}",
		})
)

func (l *linkerConfig) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	inputFile := android.PathForModuleSrc(ctx, android.String(l.properties.Src))
	l.outputFilePath = android.PathForModuleOut(ctx, "linker.config.pb").OutputPath
	l.installDirPath = android.PathForModuleInstall(ctx, "etc")
	ctx.Build(pctx, android.BuildParams{
		Rule:   linkerConfigRule,
		Input:  inputFile,
		Output: l.outputFilePath,
	})
}

func linkerConfigFactory() android.Module {
	m := &linkerConfig{}
	m.AddProperties(&m.properties)
	android.InitAndroidArchModule(m, android.HostAndDeviceSupported, android.MultilibFirst)
	return m
}

func (l *linkerConfig) AndroidMkEntries() []android.AndroidMkEntries {
	installable := l.properties.Installable == nil || android.Bool(l.properties.Installable)
	return []android.AndroidMkEntries{android.AndroidMkEntries{
		Class:      "ETC",
		OutputFile: android.OptionalPathForPath(l.outputFilePath),
		ExtraEntries: []android.AndroidMkExtraEntriesFunc{
			func(entries *android.AndroidMkEntries) {
				entries.SetString("LOCAL_MODULE_PATH", l.installDirPath.ToMakePath().String())
				entries.SetString("LOCAL_INSTALLED_MODULE_STEM", l.outputFilePath.Base())
				entries.SetString("LOCAL_UNINSTALLABLE_MODULE", strconv.FormatBool(!installable))
			},
		},
	}}
}
