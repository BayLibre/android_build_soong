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

package android

import (
	"path/filepath"

	"github.com/google/blueprint"
)

// Contains support for adding license modules to an sdk.

func init() {
	RegisterSdkMemberType(licenseModuleSdkMemberType)
}

// licenseSdkMemberType determines how a license module is added to the sdk.
type licenseSdkMemberType struct {
	SdkMemberTypeBase
}

func (l *licenseSdkMemberType) AddDependencies(mctx BottomUpMutatorContext, dependencyTag blueprint.DependencyTag, names []string) {
	// Add dependencies onto the license module from the sdk module.
	mctx.AddDependency(mctx.Module(), dependencyTag, names...)
}

func (l *licenseSdkMemberType) IsInstance(module Module) bool {
	// Verify that the module being added is compatible with this module type.
	_, ok := module.(*licenseModule)
	return ok
}

func (l *licenseSdkMemberType) AddPrebuiltModule(ctx SdkMemberContext, member SdkMember) BpModule {
	// Add the basics of a prebuilt module.
	return ctx.SnapshotBuilder().AddPrebuiltModule(member, "license")
}

func (l *licenseSdkMemberType) CreateVariantPropertiesStruct() SdkMemberProperties {
	// Create the structure into which the properties of the license module that need to be output to
	// the snapshot will be placed. The structure may be populated with information from a variant or
	// may be used as the destination for properties that are common to a set of variants.
	return &licenseSdkMemberProperties{}
}

// licenseModuleSdkMemberType is the instance of licenseSdkMemberType
var licenseModuleSdkMemberType = &licenseSdkMemberType{
	SdkMemberTypeBase{
		// This should never be added directly to an sdk/module_exports so the property should never be
		// used but it can be useful for debugging.
		// TODO(b/181569894): Prevent this being added to the sdk's properties as it is not needed and
		//  will conflict with the common property.
		PropertyName: "licenses",
		SupportsSdk:  true,
	},
}

var _ SdkMemberType = (*licenseSdkMemberType)(nil)

// licenseSdkMemberProperties is the set of properties that need to be added to the license module
// in the snapshot.
type licenseSdkMemberProperties struct {
	SdkMemberPropertiesBase

	// The kinds of licenses provided by the module.
	License_kinds []string

	// The source paths to the files containing license text.
	License_text Paths
}

func (p *licenseSdkMemberProperties) PopulateFromVariant(_ SdkMemberContext, variant Module) {
	// Populate the properties from the variant.
	l := variant.(*licenseModule)
	p.License_kinds = l.properties.License_kinds
	p.License_text = l.licenseTextPaths
}

func (p *licenseSdkMemberProperties) AddToPropertySet(ctx SdkMemberContext, propertySet BpPropertySet) {
	// Just pass any specified license_kinds straight through.
	if len(p.License_kinds) > 0 {
		propertySet.AddProperty("license_kinds", p.License_kinds)
	}

	// Copy any license test files to the snapshot into a module specific location.
	if len(p.License_text) > 0 {
		dests := []string{}
		for _, path := range p.License_text {
			// The destination path only uses the path of the license file in the source not the license
			// module name. That ensures that if the same license file is used by multiple license modules
			// that it only gets copied once as the snapshot builder will dedup copies where the source
			// and destination match.
			dest := filepath.Join("licenses", path.String())
			dests = append(dests, dest)
			ctx.SnapshotBuilder().CopyToSnapshot(path, dest)
		}
		propertySet.AddProperty("license_text", dests)
	}
}

var _ SdkMemberProperties = (*licenseSdkMemberProperties)(nil)
