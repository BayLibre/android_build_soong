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

	"github.com/google/blueprint/proptools"
)

func init() {
	RegisterLibraryHeadersBuildComponents(android.InitRegistrationContext)

	// Register sdk member types.
	android.RegisterSdkMemberType(headersLibrarySdkMemberType)

	android.Bp2BuildMutators(func(ctx android.RegisterMutatorsContext) {
		ctx.TopDown("cc_library_headers_bp2build", ccLibraryHeadersModuleToTargetMutator).Parallel()
	})
}

var headersLibrarySdkMemberType = &librarySdkMemberType{
	SdkMemberTypeBase: android.SdkMemberTypeBase{
		PropertyName:    "native_header_libs",
		SupportsSdk:     true,
		HostOsDependent: true,
	},
	prebuiltModuleType: "cc_prebuilt_library_headers",
	noOutputFiles:      true,
}

func RegisterLibraryHeadersBuildComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("cc_library_headers", LibraryHeaderFactory)
	ctx.RegisterModuleType("cc_prebuilt_library_headers", prebuiltLibraryHeaderFactory)
}

// cc_library_headers contains a set of c/c++ headers which are imported by
// other soong cc modules using the header_libs property. For best practices,
// use export_include_dirs property or LOCAL_EXPORT_C_INCLUDE_DIRS for
// Make.
func LibraryHeaderFactory() android.Module {
	module, library := NewLibrary(android.HostAndDeviceSupported)
	library.HeaderOnly()
	module.sdkMemberTypes = []android.SdkMemberType{headersLibrarySdkMemberType}
	return module.Init()
}

// cc_prebuilt_library_headers is a prebuilt version of cc_library_headers
func prebuiltLibraryHeaderFactory() android.Module {
	module, library := NewPrebuiltLibrary(android.HostAndDeviceSupported)
	library.HeaderOnly()
	return module.Init()
}

type bazelCcLibraryHeadersAttributes struct {
	Name     *string
	Includes []string
	Hdrs     android.BazelGlob
	Deps     []string
}

type bazelCcLibraryHeaders struct {
	android.ModuleBase
	bazelCcLibraryHeadersAttributes
	android.Bp2BuildProperties
}

func BazelCcLibraryHeadersFactory() android.Module {
	module := &bazelCcLibraryHeaders{}
	module.AddProperties(&module.bazelCcLibraryHeadersAttributes)
	module.AddProperties(&module.Bp2BuildProperties)
	android.InitAndroidModule(module)
	module.ConvertToBazel()
	return module

}

func ccLibraryHeadersModuleToTargetMutator(ctx android.TopDownMutatorContext) {
	if m, ok := ctx.Module().(*Module); ok {
		libIntf, ok := m.linker.(libraryInterface)
		if !ok {
			return
		}
		if libIntf.buildShared() || libIntf.buildStatic() {
			// not a cc_header_library library
			return
		}

		// FIXME(jingwen): this should use select + arch mutator
		exportHeaderLibHeaders := m.linker.linkerProps()[0].(*BaseLinkerProperties).Export_header_lib_headers
		var exportedIncludeDirs []string
		if libDecorator, ok := m.linker.(*libraryDecorator); ok {
			exportedIncludeDirs = libDecorator.flagExporter.Properties.Export_system_include_dirs
			exportedIncludeDirs = append(exportedIncludeDirs, libDecorator.flagExporter.Properties.Export_include_dirs...)
		}

		// exportSystemIncludeDirs := m.linker.linkerProps()[0].(*BaseLinkerProperties).Export_system_include_dirs

		var globHeaders []string
		for _, dir := range android.FirstUniqueStrings(exportedIncludeDirs) {
			globHeaders = append(globHeaders, dir+"/**/*.h")
		}

		name := "__bp2build__" + m.Name()
		// Replace in and out variables with $< and $@
		ctx.CreateModule(BazelCcLibraryHeadersFactory, &bazelCcLibraryHeadersAttributes{
			Name:     proptools.StringPtr(name),
			Deps:     exportHeaderLibHeaders,
			Includes: exportedIncludeDirs,
			Hdrs: android.BazelGlob{
				Include: globHeaders,
			},
		}, &android.Bp2BuildProperties{
			Rule_class: "cc_library",
		})
	}
}

func (m *bazelCcLibraryHeaders) Name() string {
	return m.BaseModuleName()
}

func (m *bazelCcLibraryHeaders) GenerateAndroidBuildActions(ctx android.ModuleContext) {}
