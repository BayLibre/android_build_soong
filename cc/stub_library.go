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
	"os"
	"strings"

	"android/soong/android"
)

func (m *Module) isStubLibrary(ctx android.SingletonContext) bool {
	if !m.Enabled() {
		return false
	}

	if pm, ok := ctx.PrimaryModule(m).(*Module); ok {
		return (pm.HasStubsVariants() || pm.IsStubs()) &&
			(!android.DirectlyInAnyApex(&notOnHostContext{}, pm.BaseModuleName()) || android.DirectlyInApex("com.android.runtime", pm.BaseModuleName()))
	}

	return false
}

func (m *Module) getInstalledFileName() (string, bool) {
	if library, ok := m.linker.(*libraryDecorator); ok {
		return library.getLibNameHelper(m.BaseModuleName(), true) + ".so", true
	}
	if prebuilt, ok := m.linker.(*prebuiltLibraryLinker); ok {
		return prebuilt.libraryDecorator.getLibNameHelper(m.BaseModuleName(), true) + ".so", true
	}
	return "", false
}

func init() {
	android.RegisterSingletonType("stub_libraries_text", stubLibrariesTextFactory)
}

type stubLibrariesText struct {
	output android.OutputPath
}

func (s *stubLibrariesText) GenerateBuildActions(ctx android.SingletonContext) {
	s.output = android.PathForOutput(ctx, "stub.libraries.txt")
	stubLibraries := make(map[string]string)

	ctx.VisitAllModules(func(module android.Module) {
		if m, ok := module.(*Module); ok {

			if m.isStubLibrary(ctx) {
				if filename, ok := m.getInstalledFileName(); ok {
					stubLibraries[m.BaseModuleName()] = filename
				}
			}
		}
	})

	if f, err := os.OpenFile(s.output.String(), os.O_CREATE|os.O_WRONLY, 0444); err == nil {
		f.WriteString(strings.Join(android.SortedStringMapValues(stubLibraries), "\n"))
	}
}

func stubLibrariesTextFactory() android.Singleton {
	return &stubLibrariesText{}
}

func (s *stubLibrariesText) MakeVars(ctx android.MakeVarsContext) {
	ctx.Strict("STUB_LIBRARIES_FILE", s.output.String())
}
