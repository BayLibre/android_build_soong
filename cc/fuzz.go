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
	"path/filepath"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"

	"android/soong/android"
	"android/soong/cc/config"
)

type FuzzProperties struct {
	// Optional list of seed files to be installed to the fuzz target's output
	// directory.
	Corpus []string `android:"path"`
	// Optional dictionary to be installed to the fuzz target's output directory.
	Dictionary *string `android:"path"`
}

func init() {
	android.RegisterModuleType("cc_fuzz", FuzzFactory)
	android.RegisterSingletonType("cc_fuzz_packaging", fuzzPackagingFactory)
}

// cc_fuzz creates a host/device fuzzer binary. Host binaries can be found at
// $ANDROID_HOST_OUT/fuzz/, and device binaries can be found at /data/fuzz on
// your device, or $ANDROID_PRODUCT_OUT/data/fuzz in your build tree.
func FuzzFactory() android.Module {
	module := NewFuzz(android.HostAndDeviceSupported)
	return module.Init()
}

func NewFuzzInstaller() *baseInstaller {
	return NewBaseInstaller("fuzz", "fuzz", InstallInData)
}

type fuzzBinary struct {
	*binaryDecorator
	*baseCompiler

	Properties FuzzProperties
	corpus     android.Paths
	dictionary android.Path
}

func (fuzz *fuzzBinary) linkerProps() []interface{} {
	props := fuzz.binaryDecorator.linkerProps()
	props = append(props, &fuzz.Properties)
	return props
}

func (fuzz *fuzzBinary) linkerInit(ctx BaseModuleContext) {
	// Add ../lib[64] to rpath so that out/host/linux-x86/fuzz/<fuzzer> can
	// find out/host/linux-x86/lib[64]/library.so
	runpaths := []string{"../lib"}
	for _, runpath := range runpaths {
		if ctx.toolchain().Is64Bit() {
			runpath += "64"
		}
		fuzz.binaryDecorator.baseLinker.dynamicProperties.RunPaths = append(
			fuzz.binaryDecorator.baseLinker.dynamicProperties.RunPaths, runpath)
	}

	// add "" to rpath so that fuzzer binaries can find libraries in their own fuzz directory
	fuzz.binaryDecorator.baseLinker.dynamicProperties.RunPaths = append(
		fuzz.binaryDecorator.baseLinker.dynamicProperties.RunPaths, "")

	fuzz.binaryDecorator.linkerInit(ctx)
}

func (fuzz *fuzzBinary) linkerDeps(ctx DepsContext, deps Deps) Deps {
	deps.StaticLibs = append(deps.StaticLibs,
		config.LibFuzzerRuntimeLibrary(ctx.toolchain()))
	deps = fuzz.binaryDecorator.linkerDeps(ctx, deps)
	return deps
}

func (fuzz *fuzzBinary) linkerFlags(ctx ModuleContext, flags Flags) Flags {
	flags = fuzz.binaryDecorator.linkerFlags(ctx, flags)
	return flags
}

func (fuzz *fuzzBinary) install(ctx ModuleContext, file android.Path) {
	fuzz.binaryDecorator.baseInstaller.dir = filepath.Join(
		"fuzz", ctx.Target().Arch.ArchType.String(), ctx.ModuleName())
	fuzz.binaryDecorator.baseInstaller.dir64 = filepath.Join(
		"fuzz", ctx.Target().Arch.ArchType.String(), ctx.ModuleName())
	fuzz.binaryDecorator.baseInstaller.install(ctx, file)

	fuzz.corpus = android.PathsForModuleSrc(ctx, fuzz.Properties.Corpus)
	if fuzz.Properties.Dictionary != nil {
		fuzz.dictionary = android.PathForModuleSrc(ctx, *fuzz.Properties.Dictionary)
		if fuzz.dictionary.Ext() != ".dict" {
			ctx.PropertyErrorf("dictionary",
				"Fuzzer dictionary %q does not have '.dict' extension",
				fuzz.dictionary.String())
		}
	}
}

func NewFuzz(hod android.HostOrDeviceSupported) *Module {
	module, binary := NewBinary(hod)

	binary.baseInstaller = NewFuzzInstaller()
	module.sanitize.SetSanitizer(fuzzer, true)

	fuzz := &fuzzBinary{
		binaryDecorator: binary,
		baseCompiler:    NewBaseCompiler(),
	}
	module.compiler = fuzz
	module.linker = fuzz
	module.installer = fuzz

	// The fuzzer runtime is not present for darwin host modules, disable cc_fuzz modules when targeting darwin.
	android.AddLoadHook(module, func(ctx android.LoadHookContext) {
		disableDarwinAndLinuxBionic := struct {
			Target struct {
				Darwin struct {
					Enabled *bool
				}
				Linux_bionic struct {
					Enabled *bool
				}
			}
		}{}
		disableDarwinAndLinuxBionic.Target.Darwin.Enabled = BoolPtr(false)
		disableDarwinAndLinuxBionic.Target.Linux_bionic.Enabled = BoolPtr(false)
		ctx.AppendProperties(&disableDarwinAndLinuxBionic)
	})

	// Statically link the STL. This allows fuzz target deployment to not have to
	// include the STL.
	android.AddLoadHook(module, func(ctx android.LoadHookContext) {
		staticStlLinkage := struct {
			Stl *string
		}{}

		staticStlLinkage.Stl = proptools.StringPtr("libc++_static")
		ctx.AppendProperties(&staticStlLinkage)
	})

	return module
}

// Responsible for generating GNU Make rules that package fuzz targets into
// their architecture & target/host specific zip file.
type fuzzPackager struct {
}

func fuzzPackagingFactory() android.Singleton {
	return &fuzzPackager{}
}

func (s *fuzzPackager) GenerateBuildActions(ctx android.SingletonContext) {
	// Map between each architecture + host/device combination, and the list of implicit
	// dependencies that need to be made.
	archDirs := make(map[android.OutputPath]android.Paths)

	ctx.VisitAllModules(func(module android.Module) {
		ccModule, ok := module.(*Module)
		if !ok {
			return
		}

		fuzzModule, ok := ccModule.compiler.(*fuzzBinary)
		if !ok {
			return
		}

		// Discard vendor-NDK-linked modules, they're duplicates of fuzz targets
		// we're going to package anyway.
		if ccModule.useVndk() || !ccModule.Enabled() {
			return
		}

		hostOrTargetString := "target"
		if ccModule.Host() {
			hostOrTargetString = "host"
		}

		// Mark this host/target + architecture specific directory as ripe for
		// packaging.
		archString := ccModule.Arch().ArchType.String()
		archDir := android.PathForIntermediates(ctx, "fuzz", hostOrTargetString, archString)

		// Map between source -> subdirectory in destination.
		filesToCopy := make(map[android.Path]string)

		// The executable.
		filesToCopy[ccModule.outputFile.Path()] = ccModule.outputFile.Path().Base()

		// The corpora.
		for _, corpusEntry := range fuzzModule.corpus {
			filesToCopy[corpusEntry] = "corpus/" + corpusEntry.Base()
		}

		// The dictionary.
		if fuzzModule.dictionary != nil {
			filesToCopy[fuzzModule.dictionary] = fuzzModule.dictionary.Base()
		}

		builder := android.NewRuleBuilder()
		for src, relativeDir := range filesToCopy {
			destination := android.PathForIntermediates(
				ctx, "fuzz", hostOrTargetString, archString, ctx.ModuleName(module),
				relativeDir)
			builder.Command().Text("cp").Input(src).Output(destination)
			archDirs[archDir] = append(archDirs[archDir], destination)
		}

		builder.Build(pctx, ctx, ctx.ModuleName(module)+"-"+ctx.ModuleSubDir(ccModule),
			"Package the "+ctx.ModuleName(module)+" fuzz target for "+ctx.ModuleSubDir(ccModule))
	})

	// List of architecture + host/device specific packages to build via. 'make fuzz'.
	var archDirTargets android.Paths

	builder := android.NewRuleBuilder()
	for archDir, implicits := range archDirs {
		// Ninja will complain that there is no rule that creates the archDir, even
		// though the immediate dependencies will create it as a parent directory.
		// Create a rule here that instantiates the parent dir explicitly.
		builder.Command().Text("mkdir").FlagWithOutput("-p ", archDir)

		outputFile := android.PathForOutput(ctx, "fuzz-"+archDir.Base()+".zip")
		builder.Command().BuiltTool(ctx, "soong_zip").
			FlagWithInput("-C ", archDir).
			FlagWithOutput("-o ", outputFile).
			FlagWithInput("-D ", archDir).
			Implicits(implicits)
		archDirTargets = append(archDirTargets, outputFile)
	}

	builder.Build(pctx, ctx, "create-fuzz-packages",
		"Create architecture + host/device specific fuzz target packages.")

	ctx.Build(pctx, android.BuildParams{
		Rule:        blueprint.Phony,
		Output:      android.PathForPhony(ctx, "fuzz"),
		Implicits:   archDirTargets,
		Description: "Build all Android fuzz targets, and create packages.",
	})
}
