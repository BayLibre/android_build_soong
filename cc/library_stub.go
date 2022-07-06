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
	"github.com/google/blueprint/proptools"

	"android/soong/android"
)

func init() {
	RegisterStubLibraryBuildComponents(android.InitRegistrationContext)
}

func RegisterStubLibraryBuildComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("cc_api_stub_library", CcApiStubLibraryFactory)
}

func CcApiStubLibraryFactory() android.Module {
	// TODO: This needs to be cleaned up
	module, decorator := NewLibrary(android.DeviceSupported)
	apiStubDecorator := &apiStubDecorator{
		libraryDecorator: decorator,
	}
	module.compiler = apiStubDecorator
	module.linker = apiStubDecorator
	module.library = apiStubDecorator
	module.installer = nil

	apiStubDecorator.BuildOnlyShared()

	stubModule := &ApiStubLibrary{
		ccModule: &ccModule{
			Module: module,
		},
	}

	android.InitAndroidArchModule(stubModule, android.DeviceSupported, android.MultilibBoth)
	stubModule.AddProperties(&apiStubDecorator.properties, &decorator.MutatedProperties)
	return stubModule
}

// TODO: desc why this is necessary
// Or better yet cleanup LinkableInterface and remove this altogether
type ccModule struct {
	*Module
}

type ApiStubLibrary struct {
	*ccModule
}

var _ LinkableInterface = (*ApiStubLibrary)(nil)

func (s *ApiStubLibrary) Module() android.Module {
	return s.ccModule.Module
}

var _ android.ApiSurfaceStubLibrary = (*ApiStubLibrary)(nil)

func (stubLibrary *ApiStubLibrary) Stem() string {
	return proptools.String(stubLibrary.compiler.(*apiStubDecorator).properties.Stem)
}

func (stubLibrary *ApiStubLibrary) Version() string {
	return proptools.String(stubLibrary.compiler.(*apiStubDecorator).properties.Version)
}

func (stubLibrary *ApiStubLibrary) SourceApiDomain() android.ApiDomain {
	return android.SystemApiDomain
	apiDomainName := proptools.String(stubLibrary.compiler.(*apiStubDecorator).properties.Api_Domain)
	return android.GetApiDomain(apiDomainName)
}

func (stubLibrary *ApiStubLibrary) ApiSurfaceName() string {
	return proptools.String(stubLibrary.compiler.(*apiStubDecorator).properties.Api_Surface)
}

func (stubLibrary *ApiStubLibrary) LibraryFactory() android.ModuleFactory {
	return LibrarySharedFactory
}

type apiStubDecorator struct {
	*libraryDecorator
	properties apiStubProperties
}

// TODO: Split this more structs
// Also this might be blueprint mutated if we create an api_surface module type
type apiStubProperties struct {
	// The base name of the stub library (e.g. libc)
	Stem *string

	// Relative path to the symbol file
	Symbol_file *string `android:"path"`

	// Version of the stub library
	// This will be used to restrict access to symbols introduced in versions >= N + 1
	Version *string

	// The API domain that hosts this contribution on the device
	// TODO: Make this required
	// TODO: Pick a better name
	Api_Domain *string

	// The API surface that this contribution belongs to
	// TODO: Even though this Android.bp file will be generated, add some validation
	Api_Surface *string
}

// TODO: Add extra flags (e.g. #apex)
func (decorator *apiStubDecorator) compile(ctx ModuleContext, flags Flags, deps PathDeps) Objects {
	if decorator.properties.Symbol_file == nil {
		ctx.PropertyErrorf("symbol_file", "symbol_file is a required field")
	}
	if decorator.properties.Version == nil {
		ctx.PropertyErrorf("version", "version is a required field")
	}
	symbolFile := String(decorator.properties.Symbol_file)
	version := android.ApiLevelOrPanic(ctx, String(decorator.properties.Version))
	nativeAbiResult := parseNativeAbiDefinition(ctx, symbolFile, version, "") // TODO: This should not be empty
	return compileStubLibrary(ctx, flags, nativeAbiResult.stubSrc)
}

// fix soname
func (decorator *apiStubDecorator) link(ctx ModuleContext, flags Flags, deps PathDeps, objects Objects) android.Path {
	return decorator.libraryDecorator.link(ctx, flags, deps, objects)
}

// TODO: add desc
func (decorator *apiStubDecorator) linkerDeps(ctx DepsContext, deps Deps) Deps {
	return deps
}

// TODO: Add desc
var _ android.ImageInterface = (*ApiStubLibrary)(nil)

func (s *ApiStubLibrary) ImageMutatorBegin(ctx android.BaseModuleContext) {}

func (s *ApiStubLibrary) CoreVariantNeeded(ctx android.BaseModuleContext) bool {
	return true //todo
}

func (s *ApiStubLibrary) RamdiskVariantNeeded(ctx android.BaseModuleContext) bool {
	return false //todo
}

func (s *ApiStubLibrary) VendorRamdiskVariantNeeded(ctx android.BaseModuleContext) bool {
	return false // false //todo
}

func (s *ApiStubLibrary) DebugRamdiskVariantNeeded(ctx android.BaseModuleContext) bool {
	return false //todo
}

func (s *ApiStubLibrary) RecoveryVariantNeeded(ctx android.BaseModuleContext) bool {
	return false
}

func (s *ApiStubLibrary) ExtraImageVariations(ctx android.BaseModuleContext) []string {
	return []string{"vendor.29"}
}

func (s *ApiStubLibrary) SetImageVariation(ctx android.BaseModuleContext, variation string, module android.Module) {
}
