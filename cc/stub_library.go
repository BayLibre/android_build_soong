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
	"strings"

	"android/soong/android"

	"github.com/google/blueprint"
)

// ApexBundleInfo from APEX
type ApexBundleInfo struct {
	Contents *android.ApexContents
}

// ApexBundleInfoProvider from APEX
var ApexBundleInfoProvider = blueprint.NewMutatorProvider(ApexBundleInfo{}, "apex_deps")

func (m *Module) isStubLibrary(ctx android.SingletonContext) bool {
	if !m.Enabled() {
		return false
	}

	if pm, ok := ctx.PrimaryModule(m).(*Module); ok {
		abInfo := ctx.ModuleProvider(m, ApexBundleInfoProvider).(ApexBundleInfo)
		isStubModule := pm.HasStubsVariants() || pm.IsStubs()

		// Stub libraries contained in APEXes will not install actual library under /system/${LIB}
		containedInAnyApex := pm.DirectlyInAnyApex()

		// Stub libraries contained in runtime APEX will create symlink of actual library under /system/${LIB}
		containedInRuntimeApex := false
		if abInfo.Contents != nil {
			containedInRuntimeApex = abInfo.Contents.DirectlyInApex("com.android.runtime")
		}

		return isStubModule && (!containedInAnyApex || containedInRuntimeApex)
	}

	return false
}

func (m *Module) getInstalledFileName() (string, bool) {
	if library, ok := m.linker.(*libraryDecorator); ok {
		return library.getLibNameHelper(m.BaseModuleName(), false) + ".so", true
	}
	if prebuilt, ok := m.linker.(*prebuiltLibraryLinker); ok {
		return prebuilt.libraryDecorator.getLibNameHelper(m.BaseModuleName(), false) + ".so", true
	}
	if llndk, ok := m.linker.(*llndkStubDecorator); ok {
		return llndk.libraryDecorator.getLibNameHelper(m.BaseModuleName(), false) + ".so", true
	}

	return "", false
}

func init() {
	android.RegisterSingletonType("stub_libraries_text", stubLibrariesTextSingleton)
}

type stubLibrariesText struct {
}

var stubLibraries = make(map[string]string)

func (s *stubLibrariesText) GenerateBuildActions(ctx android.SingletonContext) {
	ctx.VisitAllModules(func(module android.Module) {
		if m, ok := module.(*Module); ok {
			if m.isStubLibrary(ctx) {
				if filename, ok := m.getInstalledFileName(); ok {
					stubLibraries[filename] = filename
				}
			}
		}
	})
}

func stubLibrariesTextSingleton() android.Singleton {
	return &stubLibrariesText{}
}

func (s *stubLibrariesText) MakeVars(ctx android.MakeVarsContext) {
	ctx.Strict("SYSTEM_STUB_LIBRARIES", strings.Join(android.SortedStringMapValues(stubLibraries), " "))
}
