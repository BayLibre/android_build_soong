/*
 * Copyright (C) 2022 The Android Open Source Project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package provenance

import (
	"path/filepath"

	"android/soong/android"
	"android/soong/apex"
	"android/soong/java"
	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

var (
	pctx = android.NewPackageContext("android/soong/provenance")
	rule = pctx.HostBinToolVariable("gen_provenance_metadata", "gen_provenance_metadata")

	genProvenanceMetaData = pctx.AndroidStaticRule("genProvenanceMetaData",
		blueprint.RuleParams{
			Command: `rm -rf "$out" && ` +
				`${gen_provenance_metadata} --module_name=${module_name} ` +
				`--artifact_path=$in --install_path=${install_path} --metadata_path=$out`,
			CommandDeps: []string{"${gen_provenance_metadata}"},
		}, "module_name", "install_path")

	mergeProvenanceMetaData = pctx.AndroidStaticRule("mergeProvenanceMetaData",
		blueprint.RuleParams{
			Command: `rm -rf $out && ` +
				`echo -e "# proto-file: build/soong/provenance/proto/provenance_metadata.proto\n# proto-message: ProvenanceMetaDataList" > $out && ` +
				`cat $in | grep -v "^#.*" >> $out`,
		})
)

func init() {
	RegisterProvenanceSingleton(android.InitRegistrationContext)
}

func RegisterProvenanceSingleton(ctx android.RegistrationContext) {
	ctx.RegisterSingletonType("provenance_metadata_singleton", provenanceInfoSingletonFactory)
}

var PrepareForTestWithProvenanceSingleton = android.FixtureRegisterWithContext(RegisterProvenanceSingleton)

func provenanceInfoSingletonFactory() android.Singleton {
	return &provenanceInfoSingleton{}
}

type provenanceInfoSingleton struct {
}

func (b *provenanceInfoSingleton) GenerateBuildActions(context android.SingletonContext) {
	allMetaDataFiles := make([]android.Path, 0)
	context.VisitAllModulesIf(moduleFilter, func(module android.Module) {
		artifactPath := getArtifactPath(context, module)
		artifactInstallPath := getInstallPath(module)
		artifactMetaDataFile := android.PathForIntermediates(context, "provenance_metadata", context.ModuleDir(module), module.Name(), "provenance_metadata.textproto")
		allMetaDataFiles = append(allMetaDataFiles, artifactMetaDataFile)

		context.Build(pctx, android.BuildParams{
			Rule:        genProvenanceMetaData,
			Description: "generate artifact provenance metadata",
			Inputs:      android.PathsForSource(context, []string{artifactPath}),
			Output:      artifactMetaDataFile,
			Args: map[string]string{
				"module_name":  module.Name(),
				"install_path": artifactInstallPath,
			}})
	})
	mergedMetaDataFile := android.PathForOutput(context, "provenance_metadata.textproto")
	context.Build(pctx, android.BuildParams{
		Rule:        mergeProvenanceMetaData,
		Description: "merge provenance metadata",
		Inputs:      allMetaDataFiles,
		Output:      mergedMetaDataFile,
	})

	context.Build(pctx, android.BuildParams{
		Rule:        blueprint.Phony,
		Description: "phony rule of merge provenance metadata",
		Inputs:      []android.Path{mergedMetaDataFile},
		Output:      android.PathForPhony(context, "provenance_metadata"),
	})
}

func moduleFilter(module android.Module) bool {
	if !module.Enabled() || module.IsSkipInstall() {
		return false
	}
	if getInstallPath(module) == "" {
		return false
	}

	// prebuilt_apex
	if apexPrebuilt, ok := module.(*apex.Prebuilt); ok {
		for _, p := range apexPrebuilt.GetProperties() {
			if prebuiltProp, ok := p.(*apex.PrebuiltProperties); ok {
				if !proptools.BoolDefault(prebuiltProp.Installable, true) {
					return false
				}
			}
		}
		return true
	}

	// android_app_import
	if appImport, ok := module.(*java.AndroidAppImport); ok {
		if !appImport.IsInstallable() {
			return false
		}
		return true
	}

	return false
}

func getArtifactPath(context android.SingletonContext, module android.Module) string {
	// prebuilt_apex
	if apexPrebuilt, ok := module.(*apex.Prebuilt); ok {
		return apexPrebuilt.InputApex.String()
	}

	// android_app_import
	if appImport, ok := module.(*java.AndroidAppImport); ok {
		for _, props := range appImport.GetProperties() {
			if appImportProp, ok := props.(*java.AndroidAppImportProperties); ok {
				return filepath.Join(context.ModuleDir(module), proptools.String(appImportProp.Apk))
			}
		}
	}
	return ""
}

func getInstallPath(module android.Module) string {
	// prebuilt_apex
	if apexPrebuilt, ok := module.(*apex.Prebuilt); ok {
		return apexPrebuilt.InstalledFileOnDevicePath()
	}

	if appImport, ok := module.(*java.AndroidAppImport); ok {
		return appImport.InstalledFileOnDevicePath()
	}
	return ""
}
