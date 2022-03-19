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

package bcid

import (
	"path/filepath"

	"android/soong/android"
	"android/soong/apex"
	"android/soong/java"
	"android/soong/provenance/provenance_metadata_proto"
	"android/soong/shared"
	"github.com/google/blueprint/proptools"
)

const provenanceMetadataFile = "provenance_metadata.textproto"

func init() {
	RegisterBcidSingleton(android.InitRegistrationContext)
}

func RegisterBcidSingleton(ctx android.RegistrationContext) {
	ctx.RegisterSingletonType("bcid_provenance_info_singleton", bcidProvenanceInfoSingletonFactory)
}

var PrepareForTestWithBcidSingleton = android.FixtureRegisterWithContext(RegisterBcidSingleton)

func bcidProvenanceInfoSingletonFactory() android.Singleton {
	return &bcidProvenanceInfoSingleton{}
}

type bcidProvenanceInfoSingleton struct {
}

func (b *bcidProvenanceInfoSingleton) GenerateBuildActions(context android.SingletonContext) {
	var provenanceMetadataList = provenance_metadata_proto.ProvenanceMetaDataList{}
	context.VisitAllModulesIf(moduleFilter, func(module android.Module) {
		provenanceMetada := provenance_metadata_proto.ProvenanceMetaData{}
		provenanceMetada.ModuleName = module.Name()
		provenanceMetada.ArtifactPath = getArtifactPath(context, module)
		provenanceMetada.ArtifactSha256 = android.Sha256(context.Config(), provenanceMetada.ArtifactPath)
		provenanceMetada.ArtifactInstallPath = getInstallPath(module)
		provenanceMetadataList.Metadata = append(provenanceMetadataList.Metadata, &provenanceMetada)
	})
	saveToTextProto(context, provenanceMetadataList)
}

func moduleFilter(module android.Module) bool {
	if !module.Enabled() || module.IsSkipInstall() {
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

func saveToTextProto(context android.SingletonContext, provenanceMetadataList provenance_metadata_proto.ProvenanceMetaDataList) {
	config := context.Config()
	header := []string{
		"# proto-file: build/soong/bcid/proto/provenance_metadata.proto",
		"# proto-message: ProvenanceMetaDataList",
		"",
	}
	metadataFile := filepath.Join(config.Getenv("TOP"), config.OutDir(), provenanceMetadataFile)
	if err := shared.SaveTextProto(&provenanceMetadataList, metadataFile, header); err != nil {
		panic(err)
	}
}
