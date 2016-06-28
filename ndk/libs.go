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

package ndk

import (
	"fmt"
	"strconv"

	"github.com/google/blueprint"

	"android/soong/android"
	"android/soong/cc"
)

var (
	toolPath = "build/soong/ndk/gen_stub_libs.py"

	genStubSrc = pctx.StaticRule("genStubSrc",
		blueprint.RuleParams{
			Command:     toolPath + " --arch $arch --api $apiLevel $in $out",
			Description: "genStubSrc $out",
		}, "arch", "apiLevel")

	genStubLib = pctx.StaticRule("genStubLib",
		blueprint.RuleParams{
			Command:     "$ccCmd $flags -o $out $in",
			Description: "genStubSrc $out",
		}, "ccCmd", "flags")

	HostPrebuiltTag = pctx.VariableConfigMethod("HostPrebuiltTag", android.Config.PrebuiltOS)
)

// TODO(danalbert): In true Go style, I've copy pasted this from cc.go. Stop doing this.
// I have no idea how to do this the right way though. Going to have to ask dwillemsen...
func init() {
	pctx.SourcePathVariable("clangDefaultBase", "prebuilts/clang/host")
	pctx.VariableFunc("clangBase", func(config interface{}) (string, error) {
		if override := config.(android.Config).Getenv("LLVM_PREBUILTS_BASE"); override != "" {
			return override, nil
		}
		return "${clangDefaultBase}", nil
	})
	pctx.VariableFunc("clangVersion", func(config interface{}) (string, error) {
		if override := config.(android.Config).Getenv("LLVM_PREBUILTS_VERSION"); override != "" {
			return override, nil
		}
		return "clang-2812033", nil
	})
	pctx.StaticVariable("clangPath", "${clangBase}/${HostPrebuiltTag}/${clangVersion}")
	pctx.StaticVariable("clangBin", "${clangPath}/bin")
}

// Creates a stub shared library based on the provided version file.
//
// Example:
//
// ndk_library {
//     name: "ndk_libfoo.current",
//     libname: "libfoo",
//     symbol_file: "libfoo.map.txt",
//     first_version: "9",
// }
//
type libraryProperies struct {
	// Name of the library to be generated, not including the extension. E.g.
	// "libfoo".
	Libname string

	// Relative path to the symbol map.
	// An sxample file can be seen here: TODO(danalbert): Make an example.
	Symbol_file string

	// The first API level a library was available. A library will be generated
	// for every API level beginning with this one.
	First_version string
}

type libraryModule struct {
	android.ModuleBase

	properties libraryProperies

	installPaths []string
}

func (m *libraryModule) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	minVersion := 9  // Minimum version supported by the NDK.
	maxVersion := 24 // TODO(danalbert): Find a real definition of this.
	firstArchVersion := map[string]int{
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
	firstVersion, err := strconv.Atoi(m.properties.First_version)
	if err != nil {
		ctx.ModuleErrorf("Invalid first_version value (must be int): %q",
			m.properties.First_version)
	}
	if firstVersion < minVersion {
		firstVersion = minVersion
	}

	for version := firstVersion; version <= maxVersion; version++ {
		arch := ctx.Arch().String()
		if firstArchVersion[arch] > version {
			continue
		}

		fileBase := fmt.Sprintf("%s.%s.%d", m.properties.Libname, arch, version)
		stubSrcName := fileBase + ".c"
		stubSrcPath := android.PathForModuleOut(ctx, stubSrcName)
		versionScriptName := fileBase + ".map"
		versionScriptPath := android.PathForModuleOut(ctx, versionScriptName)
		symbolFilePath := android.PathForModuleSrc(ctx, m.properties.Symbol_file)
		ctx.Build(pctx, blueprint.BuildParams{
			Rule:      genStubSrc,
			Outputs:   []string{stubSrcPath.String(), versionScriptPath.String()},
			Inputs:    []string{symbolFilePath.String()},
			Implicits: []string{toolPath},
			Args: map[string]string{
				"arch":     arch,
				"apiLevel": strconv.Itoa(version),
			},
		})

		libPath := doGenStubLib(ctx, m.properties.Libname, arch, version, stubSrcPath,
			versionScriptPath)
		m.installPaths = append(m.installPaths, libPath.String())
	}
}

func doGenStubLib(ctx android.ModuleContext, libName string, arch string, version int,
	stubSrcPath android.WritablePath, versionScript android.WritablePath) android.WritablePath {

	libBasename := fmt.Sprintf("%s.so.%d", libName, version)
	libPath := getNdkSysrootBase(ctx).Join(ctx, "usr/lib", arch, libBasename)

	toolchain := cc.GetToolchain(ctx)

	flags := fmt.Sprintf("-target %s -Wl,-shared,-Bsymbolic -Wl,-soname,%s.so -nostdlib -w "+
		"-Wl,--exclude-libs,libgcc.a -Wl,--version-script=%s -Wl,--no-undefined-version",
		toolchain.ClangTriple(), libName, versionScript)

	// TODO(danalbert): Actually create the damn thing.
	ctx.Build(pctx, blueprint.BuildParams{
		Rule:      genStubLib,
		Outputs:   []string{libPath.String()},
		Inputs:    []string{stubSrcPath.String()},
		Implicits: []string{stubSrcPath.String()},
		Args: map[string]string{
			"ccCmd": "${clangBin}/clang",
			"flags": flags,
		},
	})

	return libPath
}

func ndkLibraryFactory() (blueprint.Module, []interface{}) {
	module := &libraryModule{}
	return android.InitAndroidArchModule(module, android.HostSupported, android.MultilibBoth,
		&module.properties)
}
