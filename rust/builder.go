// Copyright 2019 The Android Open Source Project
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
	"path/filepath"
	"strings"

	"github.com/google/blueprint"

	"android/soong/android"
	cc_config "android/soong/cc/config"
	"android/soong/rust/config"
)

var (
	_     = pctx.SourcePathVariable("rustcCmd", "${config.RustBin}/rustc")
	_     = pctx.SourcePathVariable("mkcraterspCmd", "build/soong/scripts/mkcratersp.py")
	rustc = pctx.AndroidStaticRule("rustc",
		blueprint.RuleParams{
			Command: "$envVars $rustcCmd " +
				"-C linker=$mkcraterspCmd " +
				"--emit link -o $out --emit dep-info=$out.d.raw $in ${libFlags} $rustcFlags" +
				" && grep \"^$out:\" $out.d.raw > $out.d",
			CommandDeps: []string{"$rustcCmd", "$mkcraterspCmd"},
			// Rustc deps-info writes out make compatible dep files: https://github.com/rust-lang/rust/issues/7633
			// Rustc emits unneeded dependency lines for the .d and input .rs files.
			// Those extra lines cause ninja warning:
			//     "warning: depfile has multiple output paths"
			// For ninja, we keep/grep only the dependency rule for the rust $out file.
			Deps:    blueprint.DepsGCC,
			Depfile: "$out.d",
		},
		"rustcFlags", "libFlags", "envVars")
	rustLink = pctx.AndroidStaticRule("rustLink",
		blueprint.RuleParams{
			Command: "${config.RustLinker} -o $out ${crtBegin} ${config.RustLinkerArgs} @$in ${linkFlags} ${crtEnd}",
		},
		"linkFlags", "crtBegin", "crtEnd")

	_       = pctx.SourcePathVariable("rustdocCmd", "${config.RustBin}/rustdoc")
	rustdoc = pctx.AndroidStaticRule("rustdoc",
		blueprint.RuleParams{
			Command: "$envVars $rustdocCmd $rustdocFlags $in -o $outDir && " +
				"touch $out",
			CommandDeps: []string{"$rustdocCmd"},
		},
		"rustdocFlags", "outDir", "envVars")

	_            = pctx.SourcePathVariable("clippyCmd", "${config.RustBin}/clippy-driver")
	clippyDriver = pctx.AndroidStaticRule("clippy",
		blueprint.RuleParams{
			Command: "$envVars $clippyCmd " +
				// Because clippy-driver uses rustc as backend, we need to have some output even during the linting.
				// Use the metadata output as it has the smallest footprint.
				"--emit metadata -o $out --emit dep-info=$out.d.raw $in ${libFlags} " +
				"$rustcFlags $clippyFlags" +
				" && grep \"^$out:\" $out.d.raw > $out.d",
			CommandDeps: []string{"$clippyCmd"},
			Deps:        blueprint.DepsGCC,
			Depfile:     "$out.d",
		},
		"rustcFlags", "libFlags", "clippyFlags", "envVars")

	zip = pctx.AndroidStaticRule("zip",
		blueprint.RuleParams{
			Command:        "cat $out.rsp | tr ' ' '\\n' | tr -d \\' | sort -u > ${out}.tmp && ${SoongZipCmd} -o ${out} -C $$OUT_DIR -l ${out}.tmp",
			CommandDeps:    []string{"${SoongZipCmd}"},
			Rspfile:        "$out.rsp",
			RspfileContent: "$in",
		})

	cp = pctx.AndroidStaticRule("cp",
		blueprint.RuleParams{
			Command:        "cp --parents `cat $outDir.rsp` $outDir",
			Rspfile:        "${outDir}.rsp",
			RspfileContent: "$in",
		},
		"outDir")

	realcp = pctx.AndroidStaticRule("realcp",
		blueprint.RuleParams{
			Command:     "rm -f $out && cp $in $out",
			Description: "cp $out",
		})

	symlink = pctx.AndroidStaticRule("symlink",
		blueprint.RuleParams{
			Command:     "rm -f $out && ln -s $in $out",
			Description: "symlink $in $out",
		})

	// Cross-referencing:
	_ = pctx.SourcePathVariable("rustExtractor",
		"prebuilts/build-tools/${config.HostPrebuiltTag}/bin/rust_extractor")
	_ = pctx.VariableFunc("kytheCorpus",
		func(ctx android.PackageVarContext) string { return ctx.Config().XrefCorpusName() })
	_ = pctx.VariableFunc("kytheCuEncoding",
		func(ctx android.PackageVarContext) string { return ctx.Config().XrefCuEncoding() })
	_            = pctx.SourcePathVariable("kytheVnames", "build/soong/vnames.json")
	kytheExtract = pctx.AndroidStaticRule("kythe",
		blueprint.RuleParams{
			Command: `KYTHE_CORPUS=${kytheCorpus} ` +
				`KYTHE_OUTPUT_FILE=$out ` +
				`KYTHE_VNAMES=$kytheVnames ` +
				`KYTHE_KZIP_ENCODING=${kytheCuEncoding} ` +
				`KYTHE_CANONICALIZE_VNAME_PATHS=prefer-relative ` +
				`$rustExtractor $envVars ` +
				`$rustcCmd ` +
				`-C linker=true ` +
				`$in ${libFlags} $rustcFlags`,
			CommandDeps:    []string{"$rustExtractor", "$kytheVnames"},
			Rspfile:        "${out}.rsp",
			RspfileContent: "$in",
		},
		"rustcFlags", "libFlags", "envVars")
)

type buildOutput struct {
	outputFile android.Path
	kytheFile  android.Path
}

func init() {
	pctx.HostBinToolVariable("SoongZipCmd", "soong_zip")
}

func TransformSrcToBinary(ctx ModuleContext, mainSrc android.Path, deps PathDeps, flags Flags,
	outputFile android.WritablePath) buildOutput {
	flags.GlobalRustFlags = append(flags.GlobalRustFlags, "-C lto=thin")

	return transformSrctoCrate(ctx, nil, mainSrc, deps, flags, outputFile, "bin")
}

func TransformSrctoRlib(ctx ModuleContext, c compiler, mainSrc android.Path, deps PathDeps, flags Flags,
	outputFile android.WritablePath) buildOutput {
	return transformSrctoCrate(ctx, c, mainSrc, deps, flags, outputFile, "rlib")
}

func TransformSrctoDylib(ctx ModuleContext, mainSrc android.Path, deps PathDeps, flags Flags,
	outputFile android.WritablePath) buildOutput {
	flags.GlobalRustFlags = append(flags.GlobalRustFlags, "-C lto=thin")

	return transformSrctoCrate(ctx, nil, mainSrc, deps, flags, outputFile, "dylib")
}

func TransformSrctoStatic(ctx ModuleContext, mainSrc android.Path, deps PathDeps, flags Flags,
	outputFile android.WritablePath) buildOutput {
	flags.GlobalRustFlags = append(flags.GlobalRustFlags, "-C lto=thin")
	return transformSrctoCrate(ctx, nil, mainSrc, deps, flags, outputFile, "staticlib")
}

func TransformSrctoShared(ctx ModuleContext, mainSrc android.Path, deps PathDeps, flags Flags,
	outputFile android.WritablePath) buildOutput {
	flags.GlobalRustFlags = append(flags.GlobalRustFlags, "-C lto=thin")
	return transformSrctoCrate(ctx, nil, mainSrc, deps, flags, outputFile, "cdylib")
}

func TransformSrctoProcMacro(ctx ModuleContext, mainSrc android.Path, deps PathDeps,
	flags Flags, outputFile android.WritablePath) buildOutput {
	return transformSrctoCrate(ctx, nil, mainSrc, deps, flags, outputFile, "proc-macro")
}

func rustLibsToPaths(libs RustLibraries) android.Paths {
	var paths android.Paths
	for _, lib := range libs {
		paths = append(paths, lib.Path)
	}
	return paths
}

func makeLibFlags(deps PathDeps, ruleCmd *android.RuleBuilderCommand) []string {
	var libFlags []string

	// Collect library/crate flags
	//for _, lib := range deps.RLibs {
	//	libPath := ruleCmd.PathForInput(lib.Path)
	//	libFlags = append(libFlags, "--extern "+lib.CrateName+"="+libPath)
	//}
	for _, lib := range deps.DyLibs {
		libPath := ruleCmd.PathForInput(lib.Path)
		libFlags = append(libFlags, "--extern "+lib.CrateName+"="+libPath)
	}
	for _, procMacro := range deps.ProcMacros {
		procMacroPath := ruleCmd.PathForInput(procMacro.Path)
		libFlags = append(libFlags, "--extern "+procMacro.CrateName+"="+procMacroPath)
	}
	for lib, _ := range deps.TransitiveRlibs {
		libPath := ruleCmd.PathForInput(lib.Path)
		libFlags = append(libFlags, "-L "+android.PathDirname(libPath))
		libFlags = append(libFlags, "--extern "+lib.CrateName+"="+libPath)
	}

	for _, path := range deps.linkDirs {
		libFlags = append(libFlags, "-L "+path)
	}

	return libFlags
}

func makeDepPaths(deps PathDeps) android.Paths {
	depPaths := android.Paths{}
	depPaths = append(depPaths, deps.Rustc)
	for _, d := range deps.Stdlibs {
		depPaths = append(depPaths, d)
	}
	for _, d := range deps.RustcLibs {
		depPaths = append(depPaths, d)
	}
	for _, d := range deps.DyLibs {
		depPaths = append(depPaths, d.Path)
	}
	//fmt.Println(deps.TransitiveRlibs)
	for d, _ := range deps.TransitiveRlibs {
		depPaths = append(depPaths, d.Path)
	}
	//for _, d := range deps.RLibs {
	//	depPaths = append(depPaths, d.Path)
	//}
	for _, d := range deps.LibDeps {
		depPaths = append(depPaths, d)
	}
	for _, d := range deps.WholeStaticLibs {
		depPaths = append(depPaths, d)
	}
	for _, d := range deps.ProcMacros {
		depPaths = append(depPaths, d.Path)
	}
	for _, d := range deps.AfdoProfiles {
		depPaths = append(depPaths, d)
	}

	for _, d := range deps.linkObjects {
		depPaths = append(depPaths, d)
	}

	//for _, d := range deps.depIncludePaths {
	//	depPaths = append(depPaths, d)
	//}
	for _, d := range deps.depGeneratedHeaders {
		depPaths = append(depPaths, d)
	}
	//for _, d := range deps.depSystemIncludePaths {
	//	depPaths = append(depPaths, d)
	//}

	for _, d := range deps.SrcDeps {
		depPaths = append(depPaths, d)
	}
	for _, d := range deps.srcProviderFiles {
		depPaths = append(depPaths, d)
	}
	return depPaths
}

func rustEnvVars(ctx ModuleContext, deps PathDeps) []string {
	var envVars []string

	var libDirs []string
	for _, lib := range deps.RustcLibs {
		if strings.Contains(lib.String(), "prebuilts/rust") {
			libDirs = append(libDirs, "__SBOX_SANDBOX_DIR__/"+android.PathDirname(lib.String()))
		}
	}
	for _, lib := range deps.LibDeps {
		libDirs = append(libDirs, "__SBOX_SANDBOX_DIR__/"+android.PathDirname(lib.String()))
	}
	envVars = append(envVars, fmt.Sprintf(
		"LD_LIBRARY_PATH=%s:$${LD_LIBRARY_PATH}",
		strings.Join(android.FirstUniqueStrings(libDirs), ":")),
	)

	// libstd requires a specific environment variable to be set. This is
	// not officially documented and may be removed in the future. See
	// https://github.com/rust-lang/rust/blob/master/library/std/src/env.rs#L866.
	if ctx.RustModule().CrateName() == "std" {
		envVars = append(envVars, "STD_ENV_ARCH="+config.StdEnvArch[ctx.RustModule().Arch().ArchType])
	}

	if len(deps.SrcDeps) > 0 {
		moduleGenDir := ctx.RustModule().compiler.CargoOutDir()
		// We must calculate an absolute path for OUT_DIR since Rust's include! macro (which normally consumes this)
		// assumes that paths are relative to the source file.
		var outDirPrefix string
		if !filepath.IsAbs(moduleGenDir.String()) {
			// If OUT_DIR is not absolute, we use $$PWD to generate an absolute path (os.Getwd() returns '/')
			outDirPrefix = "$$PWD/"
		} else {
			// If OUT_DIR is absolute, then moduleGenDir will be an absolute path, so we don't need to set this to anything.
			outDirPrefix = ""
		}
		envVars = append(envVars, "OUT_DIR="+filepath.Join(outDirPrefix, moduleGenDir.String()))
	} else {
		// TODO(pcc): Change this to "OUT_DIR=" after fixing crates to not rely on this value.
		envVars = append(envVars, "OUT_DIR=out")
	}

	envVars = append(envVars, "ANDROID_RUST_VERSION="+config.GetRustVersion(ctx))

	if ctx.RustModule().compiler.CargoEnvCompat() {
		if bin, ok := ctx.RustModule().compiler.(*binaryDecorator); ok {
			envVars = append(envVars, "CARGO_BIN_NAME="+bin.getStem(ctx))
		}
		envVars = append(envVars, "CARGO_CRATE_NAME="+ctx.RustModule().CrateName())
		envVars = append(envVars, "CARGO_PKG_NAME="+ctx.RustModule().CrateName())
		pkgVersion := ctx.RustModule().compiler.CargoPkgVersion()
		if pkgVersion != "" {
			envVars = append(envVars, "CARGO_PKG_VERSION="+pkgVersion)
		}
	}

	envVars = append(envVars, "AR=${cc_config.ClangBin}/llvm-ar")

	return envVars
}

func transformSrctoCrate(ctx ModuleContext, comp compiler, main android.Path, deps PathDeps, flags Flags,
	outputFile android.WritablePath, crateType string) buildOutput {

	var inputs android.Paths
	var implicits, linkImplicits, linkOrderOnly android.Paths
	var output buildOutput
	var rustcFlags, linkFlags []string

	output.outputFile = outputFile
	crateName := ctx.RustModule().CrateName()
	targetTriple := ctx.toolchain().RustTriple()

	envVars := rustEnvVars(ctx, deps)

	inputs = append(inputs, main)

	// Collect rustc flags
	rustcFlags = append(rustcFlags, flags.GlobalRustFlags...)
	rustcFlags = append(rustcFlags, flags.RustFlags...)
	rustcFlags = append(rustcFlags, "--crate-type="+crateType)
	if crateName != "" {
		rustcFlags = append(rustcFlags, "--crate-name="+crateName)
	}
	if targetTriple != "" {
		rustcFlags = append(rustcFlags, "--target="+targetTriple)
		linkFlags = append(linkFlags, "-target "+targetTriple)
	}

	// Suppress an implicit sysroot
	rustcFlags = append(rustcFlags, "--sysroot=/dev/null")

	// Enable incremental compilation if requested by user
	if ctx.Config().IsEnvTrue("SOONG_RUSTC_INCREMENTAL") {
		incrementalPath := android.PathForOutput(ctx, "rustc").String()
		rustcFlags = append(rustcFlags, "-Cincremental="+incrementalPath)
	}

	// Disallow experimental features
	modulePath := android.PathForModuleSrc(ctx).String()
	if !(android.IsThirdPartyPath(modulePath) || strings.HasPrefix(modulePath, "prebuilts")) {
		rustcFlags = append(rustcFlags, "-Zallow-features=\"\"")
	}

	// Collect linker flags
	linkFlags = append(linkFlags, flags.GlobalLinkFlags...)
	linkFlags = append(linkFlags, flags.LinkFlags...)

	// Check if this module needs to use the bootstrap linker
	if ctx.RustModule().Bootstrap() && !ctx.RustModule().InRecovery() && !ctx.RustModule().InRamdisk() && !ctx.RustModule().InVendorRamdisk() {
		dynamicLinker := "-Wl,-dynamic-linker,/system/bin/bootstrap/linker"
		if ctx.toolchain().Is64Bit() {
			dynamicLinker += "64"
		}
		linkFlags = append(linkFlags, dynamicLinker)
	}

	// Collect dependencies
	implicits = append(implicits, rustLibsToPaths(deps.RLibs)...)
	implicits = append(implicits, rustLibsToPaths(deps.DyLibs)...)
	implicits = append(implicits, rustLibsToPaths(deps.ProcMacros)...)
	implicits = append(implicits, deps.AfdoProfiles...)
	implicits = append(implicits, deps.srcProviderFiles...)
	implicits = append(implicits, deps.WholeStaticLibs...)

	linkImplicits = append(linkImplicits, deps.LibDeps...)
	linkImplicits = append(linkImplicits, deps.CrtBegin...)
	linkImplicits = append(linkImplicits, deps.CrtEnd...)

	linkOrderOnly = append(linkOrderOnly, deps.linkObjects...)

	if len(deps.SrcDeps) > 0 {
		moduleGenDir := ctx.RustModule().compiler.CargoOutDir()
		var outputs android.WritablePaths

		for _, genSrc := range deps.SrcDeps {
			outfile := filepath.Join(genSubDir, genSrc.String())
			if android.SuffixInList(outputs.Strings(), outfile) {
				ctx.PropertyErrorf("srcs", "multiple source providers generate the same filename output: "+genSrc.String())
			}
			outfilePath := android.PathForModuleOut(ctx, outfile)
			outputs = append(outputs, outfilePath)
		}

		ctx.Build(pctx, android.BuildParams{
			Rule:        cp,
			Description: "cp " + moduleGenDir.Path().Base(),
			Outputs:     outputs,
			Inputs:      deps.SrcDeps,
			Args: map[string]string{
				"outDir": moduleGenDir.String(),
			},
		})
		implicits = append(implicits, outputs.Paths()...)
	}

	if comp != nil {
		implicits = append(implicits, comp.compileData(ctx)...)
	}

	envVars = append(envVars, "AR="+cc_config.ClangPath(ctx, "bin/llvm-ar").String())
	envVars = append(envVars, "ANDROID_RUST_VERSION="+config.GetRustVersion(ctx))

	if ctx.RustModule().compiler.CargoEnvCompat() {
		if _, ok := ctx.RustModule().compiler.(*binaryDecorator); ok {
			envVars = append(envVars, "CARGO_BIN_NAME="+strings.TrimSuffix(outputFile.Base(), outputFile.Ext()))
		}
		envVars = append(envVars, "CARGO_CRATE_NAME="+ctx.RustModule().CrateName())
		pkgVersion := ctx.RustModule().compiler.CargoPkgVersion()
		if pkgVersion != "" {
			envVars = append(envVars, "CARGO_PKG_VERSION="+pkgVersion)
		}
	}

	libFlags := makeLibFlags(deps, nil)
	if flags.Clippy {
		clippyFile := android.PathForModuleOut(ctx, outputFile.Base()+".clippy")
		ctx.Build(pctx, android.BuildParams{
			Rule:        clippyDriver,
			Description: "clippy " + main.Rel(),
			Output:      clippyFile,
			Inputs:      inputs,
			Implicits:   implicits,
			Args: map[string]string{
				"rustcFlags":  strings.Join(rustcFlags, " "),
				"libFlags":    strings.Join(libFlags, " "),
				"clippyFlags": strings.Join(flags.ClippyFlags, " "),
				"envVars":     strings.Join(envVars, " "),
			},
		})
		// Declare the clippy build as an implicit dependency of the original crate.
		implicits = append(implicits, clippyFile)
	}

	usesLinker := crateType == "bin" || crateType == "dylib" || crateType == "cdylib" || crateType == "proc-macro"
	if comp != nil && comp.crateRoot(ctx) != nil {
		compileInSandbox(
			ctx,
			comp,
			main,
			deps,
			flags,
			outputFile,
			crateType,
			envVars,
			inputs,
			implicits,
			rustcFlags,
			linkFlags,
			linkImplicits,
			linkOrderOnly,
			usesLinker,
		)
	} else {
		compile(
			ctx,
			comp,
			main,
			deps,
			flags,
			outputFile,
			crateType,
			envVars,
			inputs,
			implicits,
			rustcFlags,
			linkFlags,
			linkImplicits,
			linkOrderOnly,
			usesLinker,
			outputFile,
			libFlags,
		)
	}

	if flags.EmitXrefs {
		kytheFile := android.PathForModuleOut(ctx, outputFile.Base()+".kzip")
		ctx.Build(pctx, android.BuildParams{
			Rule:        kytheExtract,
			Description: "Xref Rust extractor " + main.Rel(),
			Output:      kytheFile,
			Inputs:      inputs,
			Implicits:   implicits,
			Args: map[string]string{
				"rustcFlags": strings.Join(rustcFlags, " "),
				"libFlags":   strings.Join(libFlags, " "),
				"envVars":    strings.Join(envVars, " "),
			},
		})
		output.kytheFile = kytheFile
	}
	return output
}

func compile(
	ctx ModuleContext,
	comp compiler,
	main android.Path,
	deps PathDeps,
	flags Flags,
	outputFile android.WritablePath,
	crateType string,
	envVars []string,
	inputs android.Paths,
	implicits android.Paths,
	rustcFlags []string,
	linkFlags []string,
	linkImplicits android.Paths,
	linkOrderOnly android.Paths,
	usesLinker bool,
	rustcOutputFile android.WritablePath,
	libFlags []string,
) {
	if usesLinker {
		rustcOutputFile = android.PathForModuleOut(ctx, outputFile.Base()+".rsp")
	}
	ctx.Build(pctx, android.BuildParams{
		Rule:        rustc,
		Description: "rustc " + main.Rel(),
		Output:      rustcOutputFile,
		Inputs:      inputs,
		Implicits:   implicits,
		Args: map[string]string{
			"rustcFlags": strings.Join(rustcFlags, " "),
			"libFlags":   strings.Join(libFlags, " "),
			"envVars":    strings.Join(envVars, " "),
		},
	})
	if usesLinker {
		ctx.Build(pctx, android.BuildParams{
			Rule:        rustLink,
			Description: "rustLink " + main.Rel(),
			Output:      outputFile,
			Inputs:      android.Paths{rustcOutputFile},
			Implicits:   linkImplicits,
			OrderOnly:   linkOrderOnly,
			Args: map[string]string{
				"linkFlags": strings.Join(linkFlags, " "),
				"crtBegin":  strings.Join(deps.CrtBegin.Strings(), " "),
				"crtEnd":    strings.Join(deps.CrtEnd.Strings(), " "),
			},
		})
	}
}

func compileInSandbox(
	ctx ModuleContext,
	comp compiler,
	main android.Path,
	deps PathDeps,
	flags Flags,
	outputFile android.WritablePath,
	crateType string,
	envVars []string,
	inputs android.Paths,
	implicits android.Paths,
	rustcFlags []string,
	linkFlags []string,
	linkImplicits android.Paths,
	linkOrderOnly android.Paths,
	usesLinker bool,
) {
	clangBinPath := cc_config.ClangPath(ctx, "bin")

	sboxDirectory := "rustc"
	rustcSboxOutputFile := android.PathForModuleOut(ctx, sboxDirectory, outputFile.Base())
	depFile := android.PathForModuleOut(ctx, sboxDirectory, rustcSboxOutputFile.Base()+".d")
	depInfoFile := android.PathForModuleOut(ctx, sboxDirectory, rustcSboxOutputFile.Base()+".d.raw")
	var rustcImplicitOutputs android.WritablePaths

	if usesLinker {
		rustcSboxOutputFile = android.PathForModuleOut(ctx, sboxDirectory, rustcSboxOutputFile.Base()+".rsp")
		rustcImplicitOutputs = android.WritablePaths{
			android.PathForModuleOut(ctx, sboxDirectory, rustcSboxOutputFile.Base()+".whole.a"),
			android.PathForModuleOut(ctx, sboxDirectory, rustcSboxOutputFile.Base()+".a"),
		}
	}

	mkCrateRspPy := android.PathForSource(ctx, "build", "soong", "scripts", "mkcratersp.py")
	rustcRule := android.NewRuleBuilder(pctx, ctx).
		Sbox(
			android.PathForModuleOut(ctx, sboxDirectory),
			android.PathForModuleOut(ctx, sboxDirectory+".sbox.textproto"),
		).
		SandboxInputs()

	rustcCmd := rustcRule.Command()
	envVars = append(envVars, "AR="+rustcCmd.PathForTool(cc_config.ClangPath(ctx, "bin/llvm-ar")))
	libFlags := makeLibFlags(deps, rustcCmd)

	rustcCmd.
		Flags(envVars).
		Flag(
			fmt.Sprintf(
				"PATH=$${PATH}:__SBOX_SANDBOX_DIR__/tools/src/%s",
				android.PathDirname(clangBinPath.String()),
			),
		).
		Tool(config.RustPath(ctx, "bin/rustc")).
		ImplicitTool(clangBinPath.Join(ctx, "llvm-ar")).
		FlagWithInput("-C linker=", mkCrateRspPy).
		Flag("--emit link").
		Flag("-o").
		Output(rustcSboxOutputFile).
		FlagWithOutput("--emit dep-info=", depInfoFile).
		Inputs(inputs).
		Flags(libFlags).
		Implicits(makeDepPaths(deps)).
		Implicits(implicits).
		Flags(rustcFlags).
		ImplicitOutputs(rustcImplicitOutputs)

	depfileCreationCmd := rustcRule.Command()
	depfileCreationCmd.
		Flag(fmt.Sprintf(
			`grep "^%s:" %s >`,
			depfileCreationCmd.PathForOutput(rustcSboxOutputFile),
			depfileCreationCmd.PathForOutput(depInfoFile),
		)).
		DepFile(depFile)

	//depFileValidationCmd := rustcRule.Command()
	//depFileValidationCmd.
	//	Flag(fmt.Sprintf(
	//		`echo "%s" > srcs.inputs && \
	//		[[ -z $$(\
	//			comm -13 \
	//			<(cat srcs.inputs | xargs realpath | sort -u) \
	//			<(cat %s | awk -F ":" '{print $$2}' | tr " " "\n" | xargs realpath | sort -u) | \
	//			grep "\.rs$$") \
	//		]]`,
	//		strings.Join(android.Map(deps.SrcDeps, depFileValidationCmd.PathForInput), "\n"),
	//		depFileValidationCmd.PathForOutput(depInfoFile),
	//	))

	if usesLinker {
		sboxOutputFile := android.PathForModuleOut(ctx, sboxDirectory, outputFile.Base())
		rustLinkCmd := rustcRule.Command()
		rustLinkCmd.
			Flag(
				fmt.Sprintf(
					"PATH=$${PATH}:__SBOX_SANDBOX_DIR__/tools/src/%s",
					android.PathDirname(clangBinPath.String()),
				),
			).
			Tool(clangBinPath.Join(ctx, "clang++")).
			ImplicitTool(clangBinPath.Join(ctx, "clang++.real")).
			ImplicitTool(clangBinPath.Join(ctx, "lld")).
			ImplicitTool(clangBinPath.Join(ctx, "ld.lld")).
			Flag("-o").
			Output(sboxOutputFile).
			Inputs(deps.CrtBegin).
			Flag("${config.RustLinkerArgs}").
			FlagWithInput("@", rustcSboxOutputFile).
			Flags(linkFlags).
			Inputs(deps.CrtEnd).
			Implicits(rustcImplicitOutputs.Paths()).
			Implicits(makeDepPaths(deps)).
			Implicits(linkImplicits).
			OrderOnlys(linkOrderOnly)
		ctx.Build(pctx, android.BuildParams{
			Rule:   realcp,
			Input:  sboxOutputFile,
			Output: outputFile,
		})
	} else {
		ctx.Build(pctx, android.BuildParams{
			Rule:   realcp,
			Input:  rustcSboxOutputFile,
			Output: outputFile,
		})
	}

	rustcRule.BuildWithNinjaVars("rustcSandbox", "rustcSandbox "+main.Rel(), ctx, pctx)
}

func Rustdoc(ctx ModuleContext, main android.Path, deps PathDeps,
	flags Flags) android.ModuleOutPath {

	rustdocFlags := append([]string{}, flags.RustdocFlags...)
	rustdocFlags = append(rustdocFlags, "--sysroot=/dev/null")

	// Build an index for all our crates. -Z unstable options is required to use
	// this flag.
	rustdocFlags = append(rustdocFlags, "-Z", "unstable-options", "--enable-index-page")

	targetTriple := ctx.toolchain().RustTriple()

	// Collect rustc flags
	if targetTriple != "" {
		rustdocFlags = append(rustdocFlags, "--target="+targetTriple)
	}

	crateName := ctx.RustModule().CrateName()
	rustdocFlags = append(rustdocFlags, "--crate-name "+crateName)

	rustdocFlags = append(rustdocFlags, makeLibFlags(deps, nil)...)
	docTimestampFile := android.PathForModuleOut(ctx, "rustdoc.timestamp")

	// Silence warnings about renamed lints for third-party crates
	modulePath := android.PathForModuleSrc(ctx).String()
	if android.IsThirdPartyPath(modulePath) {
		rustdocFlags = append(rustdocFlags, " -A warnings")
	}

	// Yes, the same out directory is used simultaneously by all rustdoc builds.
	// This is what cargo does. The docs for individual crates get generated to
	// a subdirectory named for the crate, and rustdoc synchronizes writes to
	// shared pieces like the index and search data itself.
	// https://github.com/rust-lang/rust/blob/master/src/librustdoc/html/render/write_shared.rs#L144-L146
	docDir := android.PathForOutput(ctx, "rustdoc")

	ctx.Build(pctx, android.BuildParams{
		Rule:        rustdoc,
		Description: "rustdoc " + main.Rel(),
		Output:      docTimestampFile,
		Input:       main,
		Implicit:    ctx.RustModule().UnstrippedOutputFile(),
		Args: map[string]string{
			"rustdocFlags": strings.Join(rustdocFlags, " "),
			"outDir":       docDir.String(),
			"envVars":      strings.Join(rustEnvVars(ctx, deps), " "),
		},
	})

	return docTimestampFile
}
