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

package cc

import (
	"android/soong/android"
)

func init() {
	RegisterLibraryStubBuildComponents(android.InitRegistrationContext)
}

func RegisterLibraryStubBuildComponents(ctx android.RegistrationContext) {
	// cc_api_stub_library shares a lot of ndk_library, and this will be refactored later
	ctx.RegisterModuleType("cc_api_stub_library", CcApiStubLibraryFactory)
}

func CcApiStubLibraryFactory() android.Module {
	module, decorator := NewLibrary(android.DeviceSupported)
	apiStubDecorator := &apiStubDecorator{
		libraryDecorator: decorator,
	}
	apiStubDecorator.BuildOnlyShared()

	module.compiler = apiStubDecorator
	module.linker = apiStubDecorator
	module.installer = nil
	module.library = apiStubDecorator
	module.Properties.HideFromMake = true // TODO: remove

	android.InitAndroidArchModule(module, android.DeviceSupported, android.MultilibBoth)
	module.AddProperties(&module.Properties,
		&apiStubDecorator.properties,
		&apiStubDecorator.MutatedProperties,
		&apiStubDecorator.apiStubLibraryProperties)
	return module
}

type apiStubLiraryProperties struct {
	Imported_includes []string `android:"path"`
}

type apiStubDecorator struct {
	*libraryDecorator
	properties               libraryProperties
	apiStubLibraryProperties apiStubLiraryProperties
}

func (compiler *apiStubDecorator) stubsVersions(ctx android.BaseMutatorContext) []string {
	firstVersion := String(compiler.properties.First_version)
	return ndkLibraryVersions(ctx, android.ApiLevelOrPanic(ctx, firstVersion))
}

func (decorator *apiStubDecorator) compile(ctx ModuleContext, flags Flags, deps PathDeps) Objects {
	if decorator.stubsVersion() == "" {
		decorator.setStubsVersion("current")
	} // TODO: fix
	symbolFile := String(decorator.properties.Symbol_file)
	nativeAbiResult := parseNativeAbiDefinition(ctx, symbolFile,
		android.ApiLevelOrPanic(ctx, decorator.stubsVersion()),
		"")
	return compileStubLibrary(ctx, flags, nativeAbiResult.stubSrc)
}

func (decorator *apiStubDecorator) link(ctx ModuleContext, flags Flags, deps PathDeps, objects Objects) android.Path {
	decorator.reexportDirs(android.PathsForModuleSrc(ctx, decorator.apiStubLibraryProperties.Imported_includes)...)
	return decorator.libraryDecorator.link(ctx, flags, deps, objects)
}
