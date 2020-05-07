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

package rust

import (
	"fmt"
	"os"
	"strings"

	"android/soong/android"
	"android/soong/rust/config"
	"github.com/google/blueprint"
)

var (
	// This is basically the android.WriteFile static rule, but it ensures
	// the output directory exists and allows shell expansions in the
	// content.
	writeFile = pctx.AndroidStaticRule("writeFile",
		blueprint.RuleParams{
			Command: `mkdir -p $$(dirname $out) && ` +
				`/bin/bash -c 'echo -e $$0 > $out' "$content"`,
			Description: "writing file $out",
		},
		"content")
)

// When the environment variable SOONG_GEN_CARGO_MANIFEST=1 is set, this
// singleton creates the `cargo` build target which generates Cargo.toml build
// files from Android rust modules for use by RLS, cargo doc, and other cargo
// based tools. Cargo.toml files are generated in a separate folder structure
// (see variable cargoOutputCratesDirectory for root).

func init() {
	android.RegisterSingletonType("cargo_manifest_generator", cargoManifestGeneratorSingletonFactory)

	pctx.Import("android/soong/android")
}

func cargoManifestGeneratorSingletonFactory() android.Singleton {
	return &cargoManifestGeneratorSingleton{}
}

type cargoManifestGeneratorSingleton struct{}

const (
	cargoManifestFilename      = "Cargo.toml"
	cargoBuildScriptFilename   = "build.rs"
	cargoOutputCratesDirectory = "development" + string(os.PathSeparator) + "ide" + string(os.PathSeparator) + "cargo"

	// Environment variables used to modify behavior of this singleton.
	envVariableGenerateCargoManifest = "SOONG_GEN_CARGO_MANIFEST"
	envVariableTrue                  = "1"
)

func (c *cargoManifestGeneratorSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	// Using android.Config.Getenv instead of os.getEnv to guarantee soong will
	// re-run in case this environment variable changes.
	if ctx.Config().Getenv(envVariableGenerateCargoManifest) != envVariableTrue {
		return
	}

	// Track which projects have already had Cargo.toml generated to keep the first
	// variant for each project.
	seenProjects := map[string]bool{}
	var files []android.Path

	ctx.VisitAllModules(func(module android.Module) {
		if rustModule, ok := module.(*Module); ok {
			if cargoFile := generateCargoManifest(ctx, rustModule, seenProjects); cargoFile != nil {
				files = append(files, cargoFile)
			}
		}
	})

	ctx.Build(pctx, android.BuildParams{
		Rule:   blueprint.Phony,
		Output: android.PathForPhony(ctx, "cargo"),
		Inputs: files,
	})
}

func generateCargoManifest(ctx android.SingletonContext, rustModule *Module, seenProjects map[string]bool) android.Path {
	// We only handle library crates for now. We check for the interface
	// rather than using CcLibrary() because we want to include
	// bindgen-generated interface crates.
	if !rustModule.CcLibraryInterface() {
		return nil
	}

	// Only write Cargo.toml for the first variant of each architecture of each module
	cargoManifestDir := cargoManifestDirForModule(ctx, rustModule)
	if seenProjects[cargoManifestDir.String()] {
		return nil
	}

	seenProjects[cargoManifestDir.String()] = true

	version := fmt.Sprintf(`\"0.%s.0\"`, ctx.Config().PlatformSdkVersion())
	var features []string
	edition := config.DefaultEdition
	var rustCfgs []string
	if compiler, ok := rustModule.compiler.(compilerInterface); ok {
		edition = compiler.getEdition()
		features = compiler.getFeatures()
		for _, flag := range compiler.getFlags() {
			// We currently only handle --cfg flags.
			if strings.HasPrefix(flag, "--cfg ") {
				rustCfgs = append(rustCfgs, strings.TrimPrefix(flag, "--cfg "))
			}
		}
	} else {
		ctx.ModuleErrorf(rustModule, "Could not retrieve baseCompiler")
	}

	// Implicit dependencies
	var deps []android.Path

	// Create build.rs. We use a build script to pass --cfg flags to rustc,
	// because we can't specify these in the manifest file.
	needsBuildRs := false
	if len(rustCfgs) > 0 {
		needsBuildRs = true
		buildRsFile := cargoManifestDir.Join(ctx, cargoBuildScriptFilename)
		var f strings.Builder

		f.WriteString(`// THIS FILE WAS AUTOMATICALLY GENERATED!\n`)
		f.WriteString(`// ANY MODIFICATION WILL BE OVERWRITTEN!\n\n`)
		f.WriteString(`fn main() {\n`)
		for _, cfg := range rustCfgs {
			fmt.Fprintf(&f, `    println!(\"cargo:rustc-cfg=%s\");\n`, cfg)
		}
		f.WriteString(`}\n`)

		ctx.Build(pctx, android.BuildParams{
			Rule:        writeFile,
			Description: "Write build.rs for " + rustModule.CrateName(),
			Output:      buildRsFile,
			Args: map[string]string{
				"content": f.String(),
			},
		})
		deps = append(deps, buildRsFile)
	}

	// Create Cargo.toml
	cargoManifestFile := cargoManifestDir.Join(ctx, cargoManifestFilename)
	var f strings.Builder

	// Header.
	f.WriteString(`# THIS FILE WAS AUTOMATICALLY GENERATED!\n`)
	f.WriteString(`# ANY MODIFICATION WILL BE OVERWRITTEN!\n\n`)
	f.WriteString(`[package]\n`)
	// Required Fields
	fmt.Fprintf(&f, `name = \"%s\"\n`, rustModule.CrateName())
	f.WriteString(`# version is set to platform SDK version number\n`)
	fmt.Fprintf(&f, `version = %s\n`, version)
	f.WriteString(`publish = false\n`)
	fmt.Fprintf(&f, `edition = \"%s\"\n`, edition)
	if needsBuildRs {
		fmt.Fprintf(&f, `build = \"%s\"\n`, cargoBuildScriptFilename)
	}
	// TODO: crate-type
	f.WriteString(`\n`)

	f.WriteString(`[features]\n`)
	var quotedFeatures []string
	for _, feature := range features {
		fmt.Fprintf(&f, `%s = []\n`, feature)
		quotedFeatures = append(quotedFeatures, `\"`+feature+`\"`)
	}
	fmt.Fprintf(&f, `default = [%s]\n`, strings.Join(quotedFeatures, ", "))
	f.WriteString(`\n`)

	f.WriteString(`[lib]\n`)
	if srcPath := rustModule.compiler.srcPath(); srcPath != nil {
		fmt.Fprintf(&f, `path = \"$${PWD}/%s\"\n`, srcPath)
	}
	f.WriteString(`\n`)

	for _, dep := range rlibDeps(ctx, rustModule) {
		depDir := cargoManifestDirForModule(ctx, dep)
		deps = append(deps, depDir.Join(ctx, cargoManifestFilename))

		fmt.Fprintf(&f, `[dependencies.%s]\n`, dep.CrateName())
		fmt.Fprintf(&f, `version = %s\n`, version)
		fmt.Fprintf(&f, `path = \"$${PWD}/%s\"\n`, depDir)
		f.WriteString(`\n`)
	}

	ctx.Build(pctx, android.BuildParams{
		Rule:        writeFile,
		Description: "Write Cargo.toml for " + rustModule.CrateName(),
		Output:      cargoManifestFile,
		Args: map[string]string{
			"content": f.String(),
		},
		Implicits: deps,
	})

	return cargoManifestFile
}

func rlibDeps(ctx android.SingletonContext, mod *Module) []*Module {
	var rlibs []*Module
	ctx.VisitDirectDeps(mod.Module(), func(dep android.Module) {
		if rustDep, ok := dep.(*Module); ok {
			if lib, ok := rustDep.compiler.(libraryInterface); ok {
				if lib.rlib() {
					rlibs = append(rlibs, rustDep)
				}
			}
		}
	})
	return rlibs
}

func cargoManifestDirForModule(ctx android.SingletonContext, module *Module) android.OutputPath {
	return android.PathForOutput(ctx,
		cargoOutputCratesDirectory,
		module.CrateName())
}
