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
	"strings"

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

	android.InitAndroidArchModule(module, android.DeviceSupported, android.MultilibBoth)
	module.AddProperties(&apiStubDecorator.properties, &decorator.MutatedProperties)
	return module
}

type apiStubDecorator struct {
	*libraryDecorator
	properties apiStubProperties
}

// TODO: Split this more structs
// Also this might be blueprint mutated if we create an api_surface module type
type apiStubProperties struct {
	// Relative path to the symbol file
	Symbol_file *string `android:"path"`

	// Version of the stub library
	// This will be used to restrict access to symbols introduced in versions >= N + 1
	Version *string

	// The API surface that this contribution belongs to
	Api_surface *string

	// The unique module name of this stub library
	StubName *string `blueprint:"mutated"`
}

// Additional args passed to ndkstubgen depending on the API surface
var (
	ndkStubGenFlags = map[string]string{
		android.PublicApi.String(): "",
		android.VendorApi.String(): "--llndk",
		android.SystemApi.String(): "--apex",
	}
)

func (decorator *apiStubDecorator) compile(ctx ModuleContext, flags Flags, deps PathDeps) Objects {
	if decorator.properties.Symbol_file == nil {
		ctx.PropertyErrorf("symbol_file", "symbol_file is a required field")
	}
	if decorator.properties.Version == nil {
		ctx.PropertyErrorf("version", "version is a required field")
	}
	if decorator.properties.Api_surface == nil {
		ctx.PropertyErrorf("api_surface", "api_surface is a required field")
	}
	symbolFile := String(decorator.properties.Symbol_file)
	version := android.ApiLevelOrPanic(ctx, decorator.Version())
	apiSurfaceName := decorator.ApiSurfaceName()
	if _, ok := ndkStubGenFlags[apiSurfaceName]; !ok {
		ctx.PropertyErrorf("api_surface", "%v is not a recognized API Surface for generating stubs. Add this API Surface to `cc.ndkStubgenFlags` map", apiSurfaceName)
	}
	stubGenFlags := ndkStubGenFlags[apiSurfaceName]
	nativeAbiResult := parseNativeAbiDefinition(ctx, symbolFile, version, stubGenFlags)
	return compileStubLibrary(ctx, flags, nativeAbiResult.stubSrc)
}

// TODO: add desc
func (decorator *apiStubDecorator) linkerDeps(ctx DepsContext, deps Deps) Deps {
	return deps
}

var _ android.ApiSurfaceStubLibrary = (*apiStubDecorator)(nil)

func (stubLibrary *apiStubDecorator) Name(stem string) string {
	if stubLibrary.properties.StubName != nil {
		return *stubLibrary.properties.StubName
	}
	stubName := stubModuleName(stem, stubLibrary.ApiSurfaceName(), stubLibrary.Version())
	stubLibrary.properties.StubName = &stubName
	return stubName
}

// Helper function that generates a fully qualified name for an imported stub library
// This is necessary since the same stem library can be present in many API surfaces
// e.g libc --> libc.vendor.33
func stubModuleName(stem string, apiSurfaceName string, version string) string {
	components := []string{stem, apiSurfaceName, version}
	return strings.Join(components[:], ".")
}

func (stubLibrary *apiStubDecorator) Version() string {
	return proptools.String(stubLibrary.properties.Version)
}

func (stubLibrary *apiStubDecorator) ApiSurfaceName() string {
	return proptools.String(stubLibrary.properties.Api_surface)
}
