// Copyright 2024 The Android Open Source Project
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

package rust

import (
	"android/soong/android"
	"android/soong/cc"
	"fmt"
	"strings"
)

func init() {
	cc.SetRustStaticLibraryFactory(RustFFIStaticFactory)
}

var _ SourceProvider = (*generatedLibraryDecorator)(nil)

type generatedLibraryProperties struct {
	// Static_rlibs is here to match the cc property so cc_defaults which set it
	// can be property ingested as this module inherits the default modules from
	// a cc_* module.
	Static_rlibs []string
	Target       struct {
		// This should mirror the related set of properties
		// defined in LinkerProperties
		Vendor, Product, Recovery, Ramdisk struct {
			// Target specific
			Static_rlibs []string
		}
	}
}

type generatedLibraryDecorator struct {
	*BaseSourceProvider

	Properties        generatedLibraryProperties
	crateIncludeNames []string
}

func (library *generatedLibraryDecorator) GenerateSource(ctx ModuleContext, deps PathDeps) android.Path {
	stem := library.BaseSourceProvider.getStem(ctx)
	stemFile := android.PathForModuleOut(ctx, "mod_"+stem)

	// Collect crate names
	ctx.VisitDirectDeps(func(dep android.Module) {
		depTag := ctx.OtherModuleDependencyTag(dep)

		if rustDep, ok := dep.(*Module); ok {
			switch depTag {
			case rlibDepTag:
				// We should only have rlib deps for these generated libraries.
				if rustDep.Name() != "libstd" {
					library.crateIncludeNames = append(library.crateIncludeNames, rustDep.CrateName())
				}
			}
		}
	})

	// Generate file and write to disk.
	android.WriteFileRule(ctx, stemFile, library.genModFileContents())

	// stemFile must be first here as the first path in
	// BaseSourceProvider.OutputFiles is the library entry-point.
	library.BaseSourceProvider.OutputFiles = android.Paths{stemFile}

	// return the entry-point for our library modules.
	return stemFile
}

func (library *generatedLibraryDecorator) genModFileContents() string {
	lines := []string{
		"// @Soong generated Source",
	}
	for _, crate := range library.crateIncludeNames {
		lines = append(lines, fmt.Sprintf("extern crate %s;", crate))
	}
	return strings.Join(lines, "\n")
}

func (library *generatedLibraryDecorator) SourceProviderProps() []interface{} {
	return append(library.BaseSourceProvider.SourceProviderProps(), &library.Properties)
}

func (library *generatedLibraryDecorator) SourceProviderDeps(ctx DepsContext, deps Deps) Deps {
	deps = library.BaseSourceProvider.SourceProviderDeps(ctx, deps)
	deps.Rlibs = append(deps.Rlibs, library.Properties.Static_rlibs...)
	return deps
}

// produces a static library (Rust crate type "staticlib").
func RustFFIStaticFactory() android.Module {
	module, _ := NewGeneratedLibrary(android.HostAndDeviceSupported)
	module.compiler.(*libraryDecorator).BuildOnlyStatic()
	return module.Init()
}

func NewGeneratedLibrary(hod android.HostOrDeviceSupported) (*Module, *generatedLibraryDecorator) {
	generatedLibrary := &generatedLibraryDecorator{
		BaseSourceProvider: NewSourceProvider(),
		Properties:         generatedLibraryProperties{},
	}

	// Static library
	module := NewGeneratedLibraryModule(hod, generatedLibrary)

	return module, generatedLibrary
}

func NewGeneratedLibraryModule(hod android.HostOrDeviceSupported, sourceProvider SourceProvider) *Module {
	module, library := NewRustLibrary(hod)

	library.sourceProvider = sourceProvider

	module.sourceProvider = sourceProvider
	module.compiler = library

	library.disableLints()
	module.disableClippy()

	return module
}
