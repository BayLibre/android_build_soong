// Copyright 2017 Google Inc. All rights reserved.
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

	"github.com/google/blueprint"

	"android/soong/android"
)

var (
	llndkLibrarySuffix = ".llndk"
)

// Creates a stub shared library based on the provided version file.
//
// The name of the generated file will be based on the module name by stripping
// the ".llndk" suffix from the module name. Module names must end with ".llndk"
// (as a convention to allow soong to guess the LL-NDK name of a dependency when
// needed). "libfoo.llndk" will generate "libfoo.so.
//
// Example:
//
// llndk_library {
//     name: "libfoo.llndk",
//     symbol_file: "libfoo.map.txt",
//     export_include_dirs: ["include_vndk"],
// }
//
type llndkLibraryProperties struct {
	// Relative path to the symbol map.
	// An example file can be seen here: TODO(danalbert): Make an example.
	Symbol_file string
}

type llndkStubDecorator struct {
	*libraryDecorator

	properties llndkLibraryProperties

	versionScriptPath android.ModuleGenPath
}

func (c *llndkStubDecorator) compile(ctx ModuleContext, flags Flags, deps PathDeps) Objects {
	arch := ctx.Arch().ArchType.String()

	if !strings.HasSuffix(ctx.ModuleName(), llndkLibrarySuffix) {
		ctx.ModuleErrorf("llndk_library modules names must be suffixed with %q\n",
			llndkLibrarySuffix)
	}
	stubSrcPath := android.PathForModuleGen(ctx, "stub.c")
	c.versionScriptPath = android.PathForModuleGen(ctx, "stub.map")
	symbolFilePath := android.PathForModuleSrc(ctx, c.properties.Symbol_file)
	ctx.ModuleBuild(pctx, android.ModuleBuildParams{
		Rule:    genStubSrc,
		Outputs: []android.WritablePath{stubSrcPath, c.versionScriptPath},
		Input:   symbolFilePath,
		Args: map[string]string{
			// TODO(dwillemsen): Should we tell gen_stub_libs about vndk?
			"arch":     arch,
			"apiLevel": "current",
		},
	})

	flags.CFlags = append(flags.CFlags,
		// We're knowingly doing some otherwise unsightly things with builtin
		// functions here. We're just generating stub libraries, so ignore it.
		"-Wno-incompatible-library-redeclaration",
		"-Wno-builtin-requires-header",
		"-Wno-invalid-noreturn",

		// These libraries aren't actually used. Don't worry about unwinding
		// (avoids the need to link an unwinder into a fake library).
		"-fno-unwind-tables",
	)

	subdir := ""
	srcs := []android.Path{stubSrcPath}
	return compileObjs(ctx, flagsToBuilderFlags(flags), subdir, srcs, nil)
}

func (linker *llndkStubDecorator) linkerDeps(ctx DepsContext, deps Deps) Deps {
	return Deps{}
}

func (stub *llndkStubDecorator) linkerFlags(ctx ModuleContext, flags Flags) Flags {
	stub.libraryDecorator.libName = strings.TrimSuffix(ctx.ModuleName(),
		llndkLibrarySuffix)
	return stub.libraryDecorator.linkerFlags(ctx, flags)
}

func (stub *llndkStubDecorator) link(ctx ModuleContext, flags Flags, deps PathDeps,
	objs Objects) android.Path {

	linkerScriptFlag := "-Wl,--version-script," + stub.versionScriptPath.String()
	flags.LdFlags = append(flags.LdFlags, linkerScriptFlag)

	return stub.libraryDecorator.link(ctx, flags, deps, objs)
}

func newLLndkStubLibrary() (*Module, []interface{}) {
	module, library := NewLibrary(android.DeviceSupported)
	library.BuildOnlyShared()
	module.stl = nil
	module.sanitize = nil
	library.StripProperties.Strip.None = true

	stub := &llndkStubDecorator{
		libraryDecorator: library,
	}
	module.compiler = stub
	module.linker = stub
	module.installer = nil

	return module, []interface{}{&stub.properties, &library.MutatedProperties, &library.flagExporter.Properties}
}

func llndkLibraryFactory() (blueprint.Module, []interface{}) {
	module, properties := newLLndkStubLibrary()
	return android.InitAndroidArchModule(module, android.DeviceSupported,
		android.MultilibBoth, properties...)
}

func init() {
	android.RegisterModuleType("llndk_library", llndkLibraryFactory)
}
