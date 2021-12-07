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
	_ "fmt"
	_ "strings"

	"android/soong/android"

	"github.com/google/blueprint"
)

func init() {
	RegisterLibraryStubBuildComponents(android.InitRegistrationContext)
}

func RegisterLibraryStubBuildComponents(ctx android.RegistrationContext) {
	// cc_api_stub_library shares a lot of ndk_library, and this will be refactored later
	ctx.RegisterModuleType("cc_api_stub_library", CcApiStubLibraryFactory)
	ctx.RegisterModuleType("cc_api_contribution", CcApiContributionFactory)
	ctx.RegisterModuleType("api_surface", ApiSurfaceFactory) // TODO: Move this to android package
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
		&apiStubDecorator.MutatedProperties)
	return module
}

type apiStubDecorator struct {
	*libraryDecorator
	properties libraryProperties
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

func init() {
	pctx.HostBinToolVariable("gen_api_surface_build_files", "gen_api_surface_build_files")
}

var (
	genApiSurfaceBuildFiles = pctx.AndroidStaticRule("genApSurfaceBuildFiles",
		blueprint.RuleParams{
			Command:     "$gen_api_surface_build_files $name $symbol_file $first_version > $out",
			CommandDeps: []string{"$gen_api_surface_build_files"},
		}, "name", "symbol_file", "first_version")

	cpDirRule = pctx.AndroidStaticRule("cpDir",
		blueprint.RuleParams{
			Command: "rm -rf $out && cp -R $in $out",
		})
)

type CcApiContribution struct {
	android.ModuleBase
	properties ccApiContributionProperties
}

type ccApiContributionProperties struct {
	Symbol_file        *string
	First_version      *string
	Export_include_dir *string
}

func CcApiContributionFactory() android.Module {
	module := &CcApiContribution{}
	module.AddProperties(&module.properties)
	android.InitAndroidModule(module)
	return module
}

func (contrib *CcApiContribution) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	if contrib.properties.Symbol_file == nil {
		ctx.PropertyErrorf("symbol_file", "%v does not have symbol file", ctx.ModuleName())
	}
	if contrib.properties.First_version == nil {
		ctx.PropertyErrorf("first_version", "%v does not have first_version for stub variants", ctx.ModuleName())
	}
	genAndroidBp := outPathApiSurface(ctx, "Android.bp")
	genMapTxt := outPathApiSurface(ctx, String(contrib.properties.Symbol_file))
	genIncludeDir := outPathApiSurface(ctx, "include")

	// generate Android.bp
	ctx.Build(pctx, android.BuildParams{
		Rule:        genApiSurfaceBuildFiles,
		Description: "generate API surface build files",
		Outputs:     []android.WritablePath{genAndroidBp},
		Args: map[string]string{
			"name":          ctx.ModuleName() + ".api_surface",
			"symbol_file":   String(contrib.properties.Symbol_file),
			"first_version": String(contrib.properties.First_version),
		},
	})

	// copy map.txt for now
	// hardlinks cannot be created since nsjail creates a different mountpoint for out/
	ctx.Build(pctx, android.BuildParams{
		Rule:        android.Cp,
		Description: "import map.txt file",
		Input:       android.PathForModuleSrc(ctx, String(contrib.properties.Symbol_file)),
		Output:      genMapTxt,
	})

	// TODO: Input/Output should be files in dir, and not dir
	// copy include dir for now
	// hardlinks cannot be created since nsjail creates a different mountpoint for out/
	ctx.Build(pctx, android.BuildParams{
		Rule:        cpDirRule,
		Description: "import include dir",
		Input:       android.PathForModuleSrc(ctx, String(contrib.properties.Export_include_dir)),
		Output:      genIncludeDir,
	})

	inputs := []android.Path{genAndroidBp, genMapTxt}
	if contrib.properties.Export_include_dir != nil {
		inputs = append(inputs, genIncludeDir)
	}

	// phony target
	ctx.Build(pctx, android.BuildParams{
		Rule:   blueprint.Phony,
		Output: android.PathForPhony(ctx, ctx.ModuleName()),
		Inputs: inputs,
	})
}

// Path is out/soong/.export/ but will be different in final multi-tree layout
func outPathApiSurface(ctx android.ModuleContext, pathComponent string) android.OutputPath {
	return android.PathForOutput(ctx, ".export", ctx.ModuleName(), pathComponent)
}

type ApiSurface struct {
	android.ModuleBase
	properties apiSurfaceProperties
}

type apiSurfaceProperties struct {
	Contributions []string
}

func ApiSurfaceFactory() android.Module {
	module := &ApiSurface{}
	module.AddProperties(&module.properties)
	android.InitAndroidModule(module)
	return module
}

// TODO: create *api_contribution variant for every single api_surface it contributes to?
func (surface *ApiSurface) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	contributions := surface.properties.Contributions
	inputs := []android.Path{}
	for _, contribution := range contributions {
		inputs = append(inputs, android.PathForPhony(ctx, contribution))
	}

	// phony target
	ctx.Build(pctx, android.BuildParams{
		Rule:   blueprint.Phony,
		Output: android.PathForPhony(ctx, ctx.ModuleName()),
		Inputs: inputs,
	})
}
