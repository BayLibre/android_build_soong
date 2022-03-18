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
	"sort"
	"strings"

	"android/soong/android"
)

func init() {
	// Use singleton type to gather all generated soong modules.
	android.RegisterSingletonType("stublibraries", stubLibrariesSingleton)
}

type stubLibraries struct {
	stubLibraryMap map[string]bool

	apiListCoverageXmlPaths []string
}

// Check if the module defines stub, or itself is stub
func IsStubTarget(m *Module) bool {
	return m.IsStubs() || m.HasStubsVariants()
}

// Get target file name to be installed from this module
func getInstalledFileName(m *Module) string {
	for _, ps := range m.PackagingSpecs() {
		if name := ps.FileName(); name != "" {
			return name
		}
	}
	return ""
}

func (s *stubLibraries) GenerateBuildActions(ctx android.SingletonContext) {
	// Visit all generated soong modules and store stub library file names.
	ctx.VisitAllModules(func(module android.Module) {
		if m, ok := module.(*Module); ok {
			if IsStubTarget(m) {
				if name := getInstalledFileName(m); name != "" {
					s.stubLibraryMap[name] = true
				}
			}
			if m.library != nil {
				if p := m.library.getAPIListCoverageXMLPath().String(); p != "" {
					s.apiListCoverageXmlPaths = append(s.apiListCoverageXmlPaths, p)
				}
			}
		}
	})
}

func stubLibrariesSingleton() android.Singleton {
	return &stubLibraries{
		stubLibraryMap: make(map[string]bool),
	}
}

func (s *stubLibraries) MakeVars(ctx android.MakeVarsContext) {
	// Convert stub library file names into Makefile variable.
	ctx.Strict("STUB_LIBRARIES", strings.Join(android.SortedStringKeys(s.stubLibraryMap), " "))

	// Export the list of API XML files to Make.
	sort.Strings(s.apiListCoverageXmlPaths)
	ctx.Strict("SOONG_CC_API_XML", strings.Join(s.apiListCoverageXmlPaths, " "))
}

func init() {
	android.RegisterModuleType("cc_api_stub_library", CcApiStubLibraryFactory)
}

// cc_api_stub_library creates a stub library from map.txt files
// These are ABI stable libraries and are provided by a CC api_surface
func CcApiStubLibraryFactory() android.Module {
	module, stub := newStubLibrary()
	decorator := &apiStubDecorator{
		stubDecorator: stub,
	}
	// cc_stub_library has different compile actions than an ndk_library
	// Specifically, it cannot dumpAbi since the source might not be present in the consuming API domain
	decorator.setCanDumpAbi(false)
	module.compiler = decorator
	module.linker = decorator
	module.installer = nil // do not install stubs
	// Create non-sdk variant of cc_api_stub_library as well
	// To consuming API domains, this library should be "just another shared library"
	module.Properties.AlwaysSdk = false
	module.AddProperties(&decorator.apiSurfaceProperties)
	android.InitAndroidArchModule(module, android.DeviceSupported, android.MultilibBoth)
	return module
}

type apiStubDecorator struct {
	*stubDecorator
	apiSurfaceProperties ApiSurfaceProperties
}

type ApiSurfaceProperties struct {
	// Name of the api surface this library contributes to (e.g. ndk cc_api_surface)
	// This property will be used to determine the set of APIs available in the generated stubs
	Api_surface_name string
}

var (
	importSuffix = "import"
)

// Prevent name collisions using suffixes. This is necessary since the importing tree can contain the source variant of the library before branch mininmization is complete
// TODO(sdas): Create a mutator to replace references to foo with foo.ndk.external for vendor libraries
func (this *apiStubDecorator) Name(name string) string {
	// Add imoprtSuffix since importing tree can contain an ndk_library with the same name
	return strings.Join([]string{name, this.apiSurfaceProperties.Api_surface_name, importSuffix}, ".")
}

// TODO(spandandas): write tests
func (this *apiStubDecorator) stubGenFlags() string {
	if this.apiSurfaceProperties.Api_surface_name == "apex" {
		return "--apex"
	} else if this.apiSurfaceProperties.Api_surface_name == "llndk" {
		return "--lndk"
	}
	return "" // default
}
