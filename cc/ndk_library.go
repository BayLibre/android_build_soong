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
	"strings"
	"sync"

	"github.com/google/blueprint"

	"android/soong/android"
)

func init() {
	pctx.HostBinToolVariable("ndkStubGenerator", "ndkstubgen")
	pctx.HostBinToolVariable("ndk_api_coverage_parser", "ndk_api_coverage_parser")
	pctx.SourcePathVariable("abidiff", "prebuilts/abigail/abidiff")
	pctx.SourcePathVariable("abidw", "prebuilts/abigail/abidw")
}

var (
	genStubSrc = pctx.AndroidStaticRule("genStubSrc",
		blueprint.RuleParams{
			Command: "$ndkStubGenerator --arch $arch --api $apiLevel " +
				"--api-map $apiMap $flags $in $out",
			CommandDeps: []string{"$ndkStubGenerator"},
		}, "arch", "apiLevel", "apiMap", "flags")

	parseNdkApiRule = pctx.AndroidStaticRule("parseNdkApiRule",
		blueprint.RuleParams{
			Command:     "$ndk_api_coverage_parser $in $out --api-map $apiMap",
			CommandDeps: []string{"$ndk_api_coverage_parser"},
		}, "apiMap")

	abidw = pctx.AndroidStaticRule("abidw",
		blueprint.RuleParams{
			Command: "$abidw --load-all-types --type-id-style hash " +
				"--no-corpus-path --no-comp-dir-path -w $symbolList " +
				"--out-file $out $in",
			CommandDeps: []string{"$abidw"},
		}, "symbolList")

	ndkLibrarySuffix = ".ndk"

	// Added as a variation dependency via depsMutator.
	ndkKnownLibs = []string{}
	// protects ndkKnownLibs writes during parallel BeginMutator.
	ndkKnownLibsLock sync.Mutex
)

// The First_version and Unversioned_until properties of this struct should not
// be used directly, but rather through the ApiLevel returning methods
// firstVersion() and unversionedUntil().

// Creates a stub shared library based on the provided version file.
//
// Example:
//
// ndk_library {
//     name: "libfoo",
//     symbol_file: "libfoo.map.txt",
//     first_version: "9",
// }
//
type libraryProperties struct {
	// Relative path to the symbol map.
	// An example file can be seen here: TODO(danalbert): Make an example.
	Symbol_file *string

	// The first API level a library was available. A library will be generated
	// for every API level beginning with this one.
	First_version *string

	// The first API level that library should have the version script applied.
	// This defaults to the value of first_version, and should almost never be
	// used. This is only needed to work around platform bugs like
	// https://github.com/android-ndk/ndk/issues/265.
	Unversioned_until *string

	// Use via apiLevel on the stubDecorator.
	ApiLevel string `blueprint:"mutated"`

	// True if this API is not yet ready to be shipped in the NDK. It will be
	// available in the platform for testing, but will be excluded from the
	// sysroot provided to the NDK proper.
	Draft bool
}

type stubDecorator struct {
	*libraryDecorator

	properties libraryProperties

	versionScriptPath     android.ModuleGenPath
	parsedCoverageXmlPath android.ModuleOutPath
	installPath           android.Path
	abidumpPath           android.ModuleOutPath

	apiLevel         android.ApiLevel
	firstVersion     android.ApiLevel
	unversionedUntil android.ApiLevel
}

func shouldUseVersionScript(ctx BaseModuleContext, stub *stubDecorator) bool {
	return stub.apiLevel.GreaterThanOrEqualTo(stub.unversionedUntil)
}

func generatePerApiVariants(ctx android.BottomUpMutatorContext, m *Module,
	from android.ApiLevel, perSplit func(*Module, android.ApiLevel)) {

	var versions []android.ApiLevel
	versionStrs := []string{}
	for _, version := range ctx.Config().AllSupportedApiLevels() {
		if version.GreaterThanOrEqualTo(from) {
			versions = append(versions, version)
			versionStrs = append(versionStrs, version.String())
		}
	}
	versions = append(versions, android.FutureApiLevel)
	versionStrs = append(versionStrs, android.FutureApiLevel.String())

	modules := ctx.CreateVariations(versionStrs...)
	for i, module := range modules {
		perSplit(module.(*Module), versions[i])
	}
}

func NdkApiMutator(ctx android.BottomUpMutatorContext) {
	if m, ok := ctx.Module().(*Module); ok {
		if m.Enabled() {
			if compiler, ok := m.compiler.(*stubDecorator); ok {
				if ctx.Os() != android.Android {
					// These modules are always android.DeviceEnabled only, but
					// those include Fuchsia devices, which we don't support.
					ctx.Module().Disable()
					return
				}
				if ctx.Target().NativeBridge == android.NativeBridgeEnabled {
					ctx.Module().Disable()
					return
				}
				firstVersion, err := nativeApiLevelFromUser(ctx,
					String(compiler.properties.First_version))
				if err != nil {
					ctx.PropertyErrorf("first_version", err.Error())
					return
				}
				generatePerApiVariants(ctx, m, firstVersion,
					func(m *Module, version android.ApiLevel) {
						m.compiler.(*stubDecorator).properties.ApiLevel =
							version.String()
					})
			} else if m.SplitPerApiLevel() && m.IsSdkVariant() {
				if ctx.Os() != android.Android {
					return
				}
				from, err := nativeApiLevelFromUser(ctx, m.MinSdkVersion())
				if err != nil {
					ctx.PropertyErrorf("min_sdk_version", err.Error())
					return
				}
				generatePerApiVariants(ctx, m, from,
					func(m *Module, version android.ApiLevel) {
						m.Properties.Sdk_version = StringPtr(version.String())
					})
			}
		}
	}
}

func (this *stubDecorator) initializeProperties(ctx BaseModuleContext) bool {
	this.apiLevel = nativeApiLevelOrPanic(ctx, this.properties.ApiLevel)

	var err error
	this.firstVersion, err = nativeApiLevelFromUser(ctx,
		String(this.properties.First_version))
	if err != nil {
		ctx.PropertyErrorf("first_version", err.Error())
		return false
	}

	this.unversionedUntil, err = nativeApiLevelFromUserWithDefault(ctx,
		String(this.properties.Unversioned_until), "minimum")
	if err != nil {
		ctx.PropertyErrorf("unversioned_until", err.Error())
		return false
	}

	return true
}

func (c *stubDecorator) compilerInit(ctx BaseModuleContext) {
	c.baseCompiler.compilerInit(ctx)

	name := ctx.baseModuleName()
	if strings.HasSuffix(name, ndkLibrarySuffix) {
		ctx.PropertyErrorf("name", "Do not append %q manually, just use the base name", ndkLibrarySuffix)
	}

	ndkKnownLibsLock.Lock()
	defer ndkKnownLibsLock.Unlock()
	for _, lib := range ndkKnownLibs {
		if lib == name {
			return
		}
	}
	ndkKnownLibs = append(ndkKnownLibs, name)
}

func addStubLibraryCompilerFlags(flags Flags) Flags {
	flags.Global.CFlags = append(flags.Global.CFlags,
		// We're knowingly doing some otherwise unsightly things with builtin
		// functions here. We're just generating stub libraries, so ignore it.
		"-Wno-incompatible-library-redeclaration",
		"-Wno-incomplete-setjmp-declaration",
		"-Wno-builtin-requires-header",
		"-Wno-invalid-noreturn",
		"-Wall",
		"-Werror",
		// These libraries aren't actually used. Don't worry about unwinding
		// (avoids the need to link an unwinder into a fake library).
		"-fno-unwind-tables",
	)
	// All symbols in the stubs library should be visible.
	if inList("-fvisibility=hidden", flags.Local.CFlags) {
		flags.Local.CFlags = append(flags.Local.CFlags, "-fvisibility=default")
	}
	return flags
}

func (stub *stubDecorator) compilerFlags(ctx ModuleContext, flags Flags, deps PathDeps) Flags {
	flags = stub.baseCompiler.compilerFlags(ctx, flags, deps)
	return addStubLibraryCompilerFlags(flags)
}

type ndkApiOutputs struct {
	stubSrc       android.ModuleGenPath
	versionScript android.ModuleGenPath
	symbolList    android.ModuleGenPath
}

func parseNativeAbiDefinition(ctx ModuleContext, symbolFile string,
	apiLevel android.ApiLevel, genstubFlags string) ndkApiOutputs {

	stubSrcPath := android.PathForModuleGen(ctx, "stub.c")
	versionScriptPath := android.PathForModuleGen(ctx, "stub.map")
	symbolFilePath := android.PathForModuleSrc(ctx, symbolFile)
	symbolListPath := android.PathForModuleGen(ctx, "abi_symbol_list.txt")
	apiLevelsJson := android.GetApiLevelsJson(ctx)
	ctx.Build(pctx, android.BuildParams{
		Rule:        genStubSrc,
		Description: "generate stubs " + symbolFilePath.Rel(),
		Outputs: []android.WritablePath{stubSrcPath, versionScriptPath,
			symbolListPath},
		Input:     symbolFilePath,
		Implicits: []android.Path{apiLevelsJson},
		Args: map[string]string{
			"arch":     ctx.Arch().ArchType.String(),
			"apiLevel": apiLevel.String(),
			"apiMap":   apiLevelsJson.String(),
			"flags":    genstubFlags,
		},
	})

	return ndkApiOutputs{
		stubSrc:       stubSrcPath,
		versionScript: versionScriptPath,
		symbolList:    symbolListPath,
	}
}

func compileStubLibrary(ctx ModuleContext, flags Flags, src android.Path) Objects {
	return compileObjs(ctx, flagsToBuilderFlags(flags), "",
		android.Paths{src}, nil, nil)
}

func parseSymbolFileForCoverage(ctx ModuleContext, symbolFile string) android.ModuleOutPath {
	apiLevelsJson := android.GetApiLevelsJson(ctx)
	symbolFilePath := android.PathForModuleSrc(ctx, symbolFile)
	outputFileName := strings.Split(symbolFilePath.Base(), ".")[0]
	parsedApiCoveragePath := android.PathForModuleOut(ctx, outputFileName+".xml")
	ctx.Build(pctx, android.BuildParams{
		Rule:        parseNdkApiRule,
		Description: "parse ndk api symbol file for api coverage: " + symbolFilePath.Rel(),
		Outputs:     []android.WritablePath{parsedApiCoveragePath},
		Input:       symbolFilePath,
		Implicits:   []android.Path{apiLevelsJson},
		Args: map[string]string{
			"apiMap": apiLevelsJson.String(),
		},
	})
	return parsedApiCoveragePath
}

func (this *stubDecorator) findImplementationLibrary(ctx ModuleContext) android.Path {
	var impl *Module
	ctx.VisitDirectDeps(func(baseDep android.Module) {
		dep, ok := baseDep.(*Module)
		if !ok {
			return
		}
		if baseDep.Name()+ndkLibrarySuffix == ctx.ModuleName() {
			if impl != nil {
				panic(fmt.Sprintf("found multiple implementations: %s and %s",
					impl, dep))
			}
			impl = dep
		}
	})

	output := impl.UnstrippedOutputFile()
	if output == nil {
		ctx.ModuleErrorf("implementation module (%s) has no output", impl)
		return nil
	}

	return output
}

func canDumpAbi(ctx ModuleContext) bool {
	if ctx.ModuleName() == "libandroid.ndk" {
		// http://b/160625946
		return false
	}
	return true
}

func (this *stubDecorator) dumpAbi(ctx ModuleContext, symbolList android.Path) {
	implementationLibrary := this.findImplementationLibrary(ctx)
	this.abidumpPath = android.PathForModuleOut(ctx, "abi.xml")
	ctx.Build(pctx, android.BuildParams{
		Rule:        abidw,
		Description: fmt.Sprintf("abidw %s", implementationLibrary),
		Output:      this.abidumpPath,
		Input:       implementationLibrary,
		Implicits:   android.Paths{symbolList},
		Args: map[string]string{
			"symbolList": symbolList.String(),
		},
	})
}

func (c *stubDecorator) compile(ctx ModuleContext, flags Flags, deps PathDeps) Objects {
	if !strings.HasSuffix(String(c.properties.Symbol_file), ".map.txt") {
		ctx.PropertyErrorf("symbol_file", "must end with .map.txt")
	}

	if !c.initializeProperties(ctx) {
		// Emits its own errors, so we don't need to.
		return Objects{}
	}

	symbolFile := String(c.properties.Symbol_file)
	nativeAbiResult := parseNativeAbiDefinition(ctx, symbolFile, c.apiLevel, "")
	objs := compileStubLibrary(ctx, flags, nativeAbiResult.stubSrc)
	c.versionScriptPath = nativeAbiResult.versionScript
	if canDumpAbi(ctx) {
		c.dumpAbi(ctx, nativeAbiResult.symbolList)
		// TODO: Handle ABI checking.
	}
	if c.apiLevel.IsCurrent() && ctx.PrimaryArch() {
		c.parsedCoverageXmlPath = parseSymbolFileForCoverage(ctx, symbolFile)
	}
	return objs
}

func (linker *stubDecorator) linkerDeps(ctx DepsContext, deps Deps) Deps {
	return Deps{}
}

func (linker *stubDecorator) Name(name string) string {
	return name + ndkLibrarySuffix
}

func (stub *stubDecorator) linkerFlags(ctx ModuleContext, flags Flags) Flags {
	stub.libraryDecorator.libName = ctx.baseModuleName()
	return stub.libraryDecorator.linkerFlags(ctx, flags)
}

func (stub *stubDecorator) link(ctx ModuleContext, flags Flags, deps PathDeps,
	objs Objects) android.Path {

	if shouldUseVersionScript(ctx, stub) {
		linkerScriptFlag := "-Wl,--version-script," + stub.versionScriptPath.String()
		flags.Local.LdFlags = append(flags.Local.LdFlags, linkerScriptFlag)
		flags.LdFlagsDeps = append(flags.LdFlagsDeps, stub.versionScriptPath)
	}

	return stub.libraryDecorator.link(ctx, flags, deps, objs)
}

func (stub *stubDecorator) nativeCoverage() bool {
	return false
}

func (stub *stubDecorator) install(ctx ModuleContext, path android.Path) {
	arch := ctx.Target().Arch.ArchType.Name
	// arm64 isn't actually a multilib toolchain, so unlike the other LP64
	// architectures it's just installed to lib.
	libDir := "lib"
	if ctx.toolchain().Is64Bit() && arch != "arm64" {
		libDir = "lib64"
	}

	installDir := getNdkInstallBase(ctx).Join(ctx, fmt.Sprintf(
		"platforms/android-%s/arch-%s/usr/%s", stub.apiLevel, arch, libDir))
	stub.installPath = ctx.InstallFile(installDir, path.Base(), path)
}

func newStubLibrary() *Module {
	module, library := NewLibrary(android.DeviceSupported)
	library.BuildOnlyShared()
	module.stl = nil
	module.sanitize = nil
	library.disableStripping()

	stub := &stubDecorator{
		libraryDecorator: library,
	}
	module.compiler = stub
	module.linker = stub
	module.installer = stub

	module.Properties.AlwaysSdk = true
	module.Properties.Sdk_version = StringPtr("current")

	module.AddProperties(&stub.properties, &library.MutatedProperties)

	return module
}

// ndk_library creates a library that exposes a stub implementation of functions
// and variables for use at build time only.
func NdkLibraryFactory() android.Module {
	module := newStubLibrary()
	android.InitAndroidArchModule(module, android.DeviceSupported, android.MultilibBoth)
	module.ModuleBase.EnableNativeBridgeSupportByDefault()
	return module
}
