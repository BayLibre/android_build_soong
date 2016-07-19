// Copyright 2016 Google Inc. All rights reserved.
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
	"fmt"
	"strconv"

	"github.com/google/blueprint"

	"android/soong/android"
)

var (
	toolPath = "build/soong/cc/gen_stub_libs.py"

	genStubSrc = pctx.StaticRule("genStubSrc",
		blueprint.RuleParams{
			Command:     toolPath + " --arch $arch --api $apiLevel $in $out",
			Description: "genStubSrc $out",
		}, "arch", "apiLevel")
)

func init() {
}

// Creates a stub shared library based on the provided version file.
//
// Example:
//
// ndk_library {
//     name: "libfoo.ndk",
//     libname: "libfoo",
//     symbol_file: "libfoo.map.txt",
//     first_version: "9",
// }
//
type libraryProperties struct {
	// Name of the library to be generated, not including the extension. E.g.
	// "libfoo".
	Libname string

	// Relative path to the symbol map.
	// An sxample file can be seen here: TODO(danalbert): Make an example.
	Symbol_file string

	// The first API level a library was available. A library will be generated
	// for every API level beginning with this one.
	First_version string

	// Private property for use by the mutator that splits per-API level.
	ApiLevel int `blueprint:"mutated"`
}

type stubCompiler struct {
	baseCompiler

	properties libraryProperties
}

// OMG GO
func intMin(a int, b int) int {
	if a < b {
		return a
	} else {
		return b
	}
}

func generateStubApiVariants(mctx android.BottomUpMutatorContext, c *stubCompiler) {
	minVersion := 9 // Minimum version supported by the NDK.
	maxVersion := mctx.AConfig().PlatformSdkVersion()
	firstArchVersions := map[string]int{
		"arm":    9,
		"arm64":  21,
		"mips":   16,
		"mips64": 21,
		"x86":    9,
		"x86_64": 21,
	}

	// If the NDK drops support for a platform version, we don't want to have to
	// fix up every module that was using it as its minimum version. Clip to the
	// supported version here instead.
	firstVersion, err := strconv.Atoi(c.properties.First_version)
	if err != nil {
		mctx.ModuleErrorf("Invalid first_version value (must be int): %q",
			c.properties.First_version)
	}
	if firstVersion < minVersion {
		firstVersion = minVersion
	}

	arch := mctx.Arch().ArchType.String()
	firstArchVersion, ok := firstArchVersions[arch]
	if !ok {
		panic(fmt.Sprintf("Arch %q not found in firstArchVersions", arch))
	}
	firstGenVersion := intMin(firstVersion, firstArchVersion)
	versionStrs := make([]string, maxVersion-firstGenVersion+1)
	for version := firstGenVersion; version <= maxVersion; version++ {
		versionStrs[version-firstGenVersion] = strconv.Itoa(version)
	}

	modules := mctx.CreateVariations(versionStrs...)
	for i, module := range modules {
		module.(*Module).compiler.(*stubCompiler).properties.ApiLevel = firstGenVersion + i
	}
}

func apiMutator(mctx android.BottomUpMutatorContext) {
	if m, ok := mctx.Module().(*Module); ok {
		if m.compiler != nil {
			if compiler, ok := m.compiler.(*stubCompiler); ok {
				generateStubApiVariants(mctx, compiler)
			}
		}
	}
}

func (c *stubCompiler) compile(ctx ModuleContext, flags Flags, deps PathDeps) android.Paths {
	arch := ctx.Arch().ArchType.String()

	fileBase := fmt.Sprintf("%s.%s.%d", c.properties.Libname, arch, c.properties.ApiLevel)
	stubSrcName := fileBase + ".c"
	stubSrcPath := android.PathForModuleGen(ctx, stubSrcName)
	versionScriptName := fileBase + ".map"
	versionScriptPath := android.PathForModuleGen(ctx, versionScriptName)
	symbolFilePath := android.PathForModuleSrc(ctx, c.properties.Symbol_file)
	ctx.Build(pctx, blueprint.BuildParams{
		Rule:      genStubSrc,
		Outputs:   []string{stubSrcPath.String(), versionScriptPath.String()},
		Inputs:    []string{symbolFilePath.String()},
		Implicits: []string{toolPath},
		Args: map[string]string{
			"arch":     arch,
			"apiLevel": strconv.Itoa(c.properties.ApiLevel),
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
	srcs := []string{}
	excludeSrcs := []string{}
	extraSrcs := []android.Path{stubSrcPath}
	extraDeps := []android.Path{}
	return c.baseCompiler.compileObjs(ctx, flags, subdir, srcs, excludeSrcs,
		extraSrcs, extraDeps)
}

func (c *stubCompiler) AndroidMk(ctx AndroidMkContext, ret *android.AndroidMkData) {
	ret.SubName = "." + strconv.Itoa(c.properties.ApiLevel)
}

type stubLinker struct {
	libraryLinker
}

func (linker *stubLinker) deps(ctx BaseModuleContext, deps Deps) Deps {
	return Deps{}
}

func newStubLibrary() *Module {
	module := newModule(android.DeviceSupported, android.MultilibBoth)
	module.stl = nil

	linker := &stubLinker{}
	linker.dynamicProperties.BuildShared = true
	linker.dynamicProperties.BuildStatic = false
	module.linker = linker

	module.compiler = &stubCompiler{}
	module.installer = &baseInstaller{
		dir:   "lib",
		dir64: "lib64",
	}

	return module
}

func ndkLibraryFactory() (blueprint.Module, []interface{}) {
	module := newStubLibrary()
	return android.InitAndroidArchModule(module, android.DeviceSupported, android.MultilibBoth,
		&module.compiler.(*stubCompiler).properties)
}
