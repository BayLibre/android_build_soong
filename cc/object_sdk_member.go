// Copyright 2020 Google Inc. All rights reserved.
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

package cc

import (
	"android/soong/android"
	"path/filepath"

	"github.com/google/blueprint"
)

func init() {
	android.RegisterSdkMemberType(ccObjectSdkMemberType)
}

var ccObjectSdkMemberType = &objectSdkMemberType{
	SdkMemberTypeBase: android.SdkMemberTypeBase{
		PropertyName: "native_objects",
		SupportsSdk:  true,
	},
}

type objectSdkMemberType struct {
	android.SdkMemberTypeBase
}

func (mt *objectSdkMemberType) AddDependencies(mctx android.BottomUpMutatorContext, dependencyTag blueprint.DependencyTag, names []string) {
	targets := mctx.MultiTargets()
	for _, obj := range names {
		name, version := StubsLibNameAndVersion(obj)
		for _, target := range targets {
			if version == "" {
				version = LatestStubsVersionFor(mctx.Config(), name)
			}
			mctx.AddFarVariationDependencies(append(target.Variations(), []blueprint.Variation{
				{Mutator: "version", Variation: version},
			}...), dependencyTag, name)
		}
	}
}

func (mt *objectSdkMemberType) IsInstance(module android.Module) bool {
	if m, ok := module.(*Module); ok {
		for _, allowableMemberType := range m.sdkMemberTypes {
			if allowableMemberType == mt {
				return true
			}
		}
	}
	return false
}

func (mt *objectSdkMemberType) AddPrebuiltModule(sdkModuleContext android.ModuleContext, builder android.SnapshotBuilder, member android.SdkMember) android.BpModule {
	return builder.AddPrebuiltModule(member, "cc_prebuilt_object")
}

func (mt *objectSdkMemberType) CreateVariantPropertiesStruct() android.SdkMemberProperties {
	return &objectInfoProperties{}
}

type objectInfoProperties struct {
	android.SdkMemberPropertiesBase

	// archType is not exported as if set (to a non default value) it is always arch specific.
	// This is "" for common properties.
	archType string

	// outputFile is not exported as it is always arch specific.
	outputFile android.Path
}

func (p *objectInfoProperties) PopulateFromVariant(variant android.SdkAware) {
	ccModule := variant.(*Module)
	p.archType = ccModule.Target().Arch.ArchType.String()
	p.outputFile = ccModule.OutputFile().Path()
}

// objectPathFor returns path to the object file, relative to <sdk_root>/<api_dir>.
func objectPathFor(obj objectInfoProperties) string {
	return filepath.Join(obj.OsPrefix(), obj.archType, "lib", obj.outputFile.Base())
}

func (p *objectInfoProperties) AddToPropertySet(sdkModuleContext android.ModuleContext, builder android.SnapshotBuilder, propertySet android.BpPropertySet) {
	if p.outputFile != nil {
		propertySet.AddProperty("srcs", []string{objectPathFor(*p)})
		builder.CopyToSnapshot(p.outputFile, objectPathFor(*p))
	}
}
