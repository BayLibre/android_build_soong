// Copyright 2024 Google Inc. All rights reserved.
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

package golang

import (
	"android/soong/android"
	"github.com/google/blueprint"
	"github.com/google/blueprint/bootstrap"
)

func init() {
	// Wrap the blueprint Go module types with Soong ones that interoperate with the rest of the Soong modules.
	bootstrap.GoModuleTypesAreWrapped()
	RegisterGoModuleTypes(android.InitRegistrationContext)
}

func RegisterGoModuleTypes(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("bootstrap_go_package", goPackageModuleFactory)
	ctx.RegisterModuleType("blueprint_go_binary", goBinaryModuleFactory)
}

// A GoPackage is a module for building Go packages.
type GoPackage struct {
	android.ModuleBase
	bootstrap.GoPackage
}

func goPackageModuleFactory() android.Module {
	module := &GoPackage{}
	module.AddProperties(module.Properties()...)
	android.InitAndroidArchModule(module, android.HostSupported, android.MultilibFirst)
	return module
}

func (g *GoPackage) GenerateBuildActions(ctx blueprint.ModuleContext) {
	g.ModuleBase.GenerateBuildActions(ctx)
}

func (g *GoPackage) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	g.GoPackage.GenerateBuildActions(ctx.BlueprintModuleContext())
}

// A GoBinary is a module for building executable binaries from Go sources.
type GoBinary struct {
	android.ModuleBase
	bootstrap.GoBinary

	outputFile  android.Path
	installPath android.Path
}

func goBinaryModuleFactory() android.Module {
	module := &GoBinary{}
	module.AddProperties(module.Properties()...)
	android.InitAndroidArchModule(module, android.HostSupportedNoCross, android.MultilibFirst)
	return module
}

func (g *GoBinary) GenerateBuildActions(ctx blueprint.ModuleContext) {
	g.ModuleBase.GenerateBuildActions(ctx)
}

func (g *GoBinary) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	// Install the file in Soong instead of blueprint so that Soong knows about the install rules.
	g.GoBinary.SetSkipInstall()

	// Run the build actions from the wrapped blueprint bootstrap module.
	g.GoBinary.GenerateBuildActions(ctx.BlueprintModuleContext())

	// Translate the bootstrap module's string path into a Path
	outputFile := android.PathForArbitraryOutput(ctx, android.Rel(ctx, ctx.Config().OutDir(), g.IntermediateFile()))
	g.outputFile = outputFile

	installPath := ctx.InstallFile(android.PathForModuleInstall(ctx, "bin"), ctx.ModuleName(), outputFile)
	g.installPath = installPath

	ctx.SetOutputFiles(android.Paths{outputFile}, "")
}

func (g *GoBinary) HostToolPath() android.OptionalPath {
	return android.OptionalPathForPath(g.outputFile)
}

func (g *GoBinary) AndroidMkEntries() []android.AndroidMkEntries {
	return []android.AndroidMkEntries{
		{
			Class:      "EXECUTABLES",
			OutputFile: android.OptionalPathForPath(g.outputFile),
			Include:    "$(BUILD_SYSTEM)/soong_cc_rust_prebuilt.mk",
		},
	}
}
