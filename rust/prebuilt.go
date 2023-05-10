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
	"android/soong/android"
)

func init() {
	android.RegisterModuleType("rust_prebuilt_library", PrebuiltLibraryFactory)
	android.RegisterModuleType("rust_prebuilt_dylib", PrebuiltDylibFactory)
	android.RegisterModuleType("rust_prebuilt_rlib", PrebuiltRlibFactory)
	android.RegisterModuleType("rust_prebuilt_proc_macro", PrebuiltProcMacroFactory)
	android.RegisterModuleType("rust_prebuilt_binary", rustPrebuiltBinaryFactory)
}

type PrebuiltProperties struct {
	// path to the prebuilt file
	Srcs []string `android:"path,arch_variant"`
	// directories containing associated rlib dependencies
	Link_dirs []string `android:"path,arch_variant"`
}

type prebuiltLibraryDecorator struct {
	android.Prebuilt

	*libraryDecorator
	Properties PrebuiltProperties
}

type prebuiltProcMacroDecorator struct {
	android.Prebuilt

	*procMacroDecorator
	Properties PrebuiltProperties
}

type Sanitized struct {
	None struct {
		Srcs []string `android:"path,arch_variant"`
	} `android:"arch_variant"`
	Address struct {
		Srcs []string `android:"path,arch_variant"`
	} `android:"arch_variant"`
	Hwaddress struct {
		Srcs []string `android:"path,arch_variant"`
	} `android:"arch_variant"`
}

type prebuiltLinkerProperties struct {
	// a prebuilt library or binary. Can reference a genrule module that generates an executable file.
	Srcs []string `android:"path,arch_variant"`

	Sanitized Sanitized `android:"arch_variant"`

	// Check the prebuilt ELF files (e.g. DT_SONAME, DT_NEEDED, resolution of undefined
	// symbols, etc), default true.
	Check_elf_files *bool

	// if set, add an extra objcopy --prefix-symbols= step
	Prefix_symbols *string

	// Optionally provide an import library if this is a Windows PE DLL prebuilt.
	// This is needed only if this library is linked by other modules in build time.
	// Only makes sense for the Windows target.
	Windows_import_lib *string `android:"path,arch_variant"`

	// MixedBuildsDisabled is true if and only if building this prebuilt is explicitly disabled in mixed builds for either
	// its static or shared version on the current build variant. This is to prevent Bazel targets for build variants with
	// which either the static or shared version is incompatible from participating in mixed buiods. Please note that this
	// is an override and does not fully determine whether Bazel or Soong will be used. For the full determination, see
	// cc.ProcessBazelQueryResponse, cc.QueueBazelCall, and cc.MixedBuildsDisabled.
	MixedBuildsDisabled bool `blueprint:"mutated"`
}

type prebuiltLinker struct {
	android.Prebuilt

	properties prebuiltLinkerProperties
}

type prebuiltBinaryDecorator struct {
	android.Prebuilt

	*binaryDecorator
	prebuiltLinker

	toolPath android.OptionalPath
}

func PrebuiltProcMacroFactory() android.Module {
	module, _ := NewPrebuiltProcMacro(android.HostSupportedNoCross)
	return module.Init()
}

type rustPrebuilt interface {
	prebuiltSrcs() []string
	prebuilt() *android.Prebuilt
}

func NewPrebuiltProcMacro(hod android.HostOrDeviceSupported) (*Module, *prebuiltProcMacroDecorator) {
	module, library := NewProcMacro(hod)
	prebuilt := &prebuiltProcMacroDecorator{
		procMacroDecorator: library,
	}
	module.compiler = prebuilt

	addSrcSupplier(module, prebuilt)

	return module, prebuilt
}

var _ compiler = (*prebuiltLibraryDecorator)(nil)
var _ exportedFlagsProducer = (*prebuiltLibraryDecorator)(nil)
var _ rustPrebuilt = (*prebuiltLibraryDecorator)(nil)

var _ compiler = (*prebuiltProcMacroDecorator)(nil)
var _ exportedFlagsProducer = (*prebuiltProcMacroDecorator)(nil)
var _ rustPrebuilt = (*prebuiltProcMacroDecorator)(nil)
var _ rustPrebuilt = (*prebuiltBinaryDecorator)(nil)

func PrebuiltLibraryFactory() android.Module {
	module, _ := NewPrebuiltLibrary(android.HostAndDeviceSupported)
	return module.Init()
}

func PrebuiltDylibFactory() android.Module {
	module, _ := NewPrebuiltDylib(android.HostAndDeviceSupported)
	return module.Init()
}

func PrebuiltRlibFactory() android.Module {
	module, _ := NewPrebuiltRlib(android.HostAndDeviceSupported)
	return module.Init()
}

func rustPrebuiltBinaryFactory() android.Module {
	module, _ := NewPrebuiltBinary(android.HostSupported)
	return module.Init()
}

func addSrcSupplier(module android.PrebuiltInterface, prebuilt rustPrebuilt) {
	srcsSupplier := func(_ android.BaseModuleContext, _ android.Module) []string {
		return prebuilt.prebuiltSrcs()
	}
	android.InitPrebuiltModuleWithSrcSupplier(module, srcsSupplier, "srcs")
}

func NewPrebuiltLibrary(hod android.HostOrDeviceSupported) (*Module, *prebuiltLibraryDecorator) {
	module, library := NewRustLibrary(hod)
	library.BuildOnlyRust()
	library.setNoStdlibs()
	prebuilt := &prebuiltLibraryDecorator{
		libraryDecorator: library,
	}
	module.compiler = prebuilt

	addSrcSupplier(module, prebuilt)

	return module, prebuilt
}

func NewPrebuiltDylib(hod android.HostOrDeviceSupported) (*Module, *prebuiltLibraryDecorator) {
	module, library := NewRustLibrary(hod)
	library.BuildOnlyDylib()
	library.setNoStdlibs()
	prebuilt := &prebuiltLibraryDecorator{
		libraryDecorator: library,
	}
	module.compiler = prebuilt

	addSrcSupplier(module, prebuilt)

	return module, prebuilt
}

func NewPrebuiltRlib(hod android.HostOrDeviceSupported) (*Module, *prebuiltLibraryDecorator) {
	module, library := NewRustLibrary(hod)
	library.BuildOnlyRlib()
	library.setNoStdlibs()
	prebuilt := &prebuiltLibraryDecorator{
		libraryDecorator: library,
	}
	module.compiler = prebuilt

	addSrcSupplier(module, prebuilt)

	return module, prebuilt
}

func NewPrebuiltBinary(hod android.HostOrDeviceSupported) (*Module, *binaryDecorator) {
	module, binary := NewRustBinary(hod)
	module.compiler = nil

	prebuilt := &prebuiltBinaryDecorator{
		binaryDecorator: binary,
	}

	module.AddProperties(&prebuilt.properties)

	module.compiler = prebuilt
	addSrcSupplier(module, prebuilt)
	return module, binary
}

func (prebuilt *prebuiltLibraryDecorator) compilerProps() []interface{} {
	return append(prebuilt.libraryDecorator.compilerProps(),
		&prebuilt.Properties)
}

func (prebuilt *prebuiltLibraryDecorator) compile(ctx ModuleContext, flags Flags, deps PathDeps) buildOutput {
	prebuilt.flagExporter.exportLinkDirs(android.PathsForModuleSrc(ctx, prebuilt.Properties.Link_dirs).Strings()...)
	prebuilt.flagExporter.setProvider(ctx)

	srcPath, paths := srcPathFromModuleSrcs(ctx, prebuilt.prebuiltSrcs())
	if len(paths) > 0 {
		ctx.PropertyErrorf("srcs", "prebuilt libraries can only have one entry in srcs (the prebuilt path)")
	}
	prebuilt.baseCompiler.unstrippedOutputFile = srcPath
	return buildOutput{outputFile: srcPath}
}

func (prebuilt *prebuiltLibraryDecorator) rustdoc(ctx ModuleContext, flags Flags,
	deps PathDeps) android.OptionalPath {

	return android.OptionalPath{}
}

func (prebuilt *prebuiltLibraryDecorator) compilerDeps(ctx DepsContext, deps Deps) Deps {
	deps = prebuilt.baseCompiler.compilerDeps(ctx, deps)
	return deps
}

func (prebuilt *prebuiltLibraryDecorator) nativeCoverage() bool {
	return false
}

func (prebuilt *prebuiltLibraryDecorator) prebuiltSrcs() []string {
	srcs := prebuilt.Properties.Srcs
	if prebuilt.rlib() {
		srcs = append(srcs, prebuilt.libraryDecorator.Properties.Rlib.Srcs...)
	}
	if prebuilt.dylib() {
		srcs = append(srcs, prebuilt.libraryDecorator.Properties.Dylib.Srcs...)
	}

	return srcs
}

func (prebuilt *prebuiltLibraryDecorator) prebuilt() *android.Prebuilt {
	return &prebuilt.Prebuilt
}

func (prebuilt *prebuiltProcMacroDecorator) prebuiltSrcs() []string {
	srcs := prebuilt.Properties.Srcs
	return srcs
}

func (prebuilt *prebuiltProcMacroDecorator) prebuilt() *android.Prebuilt {
	return &prebuilt.Prebuilt
}

func (prebuilt *prebuiltBinaryDecorator) prebuilt() *android.Prebuilt {
	return &prebuilt.Prebuilt
}

func (prebuilt *prebuiltBinaryDecorator) prebuiltSrcs() []string {
	srcs := prebuilt.prebuiltLinker.properties.Srcs
	return srcs
}

func (prebuilt *prebuiltProcMacroDecorator) compilerProps() []interface{} {
	return append(prebuilt.procMacroDecorator.compilerProps(),
		&prebuilt.Properties)
}

func (prebuilt *prebuiltProcMacroDecorator) compile(ctx ModuleContext, flags Flags, deps PathDeps) buildOutput {
	prebuilt.flagExporter.exportLinkDirs(android.PathsForModuleSrc(ctx, prebuilt.Properties.Link_dirs).Strings()...)
	prebuilt.flagExporter.setProvider(ctx)

	srcPath, paths := srcPathFromModuleSrcs(ctx, prebuilt.prebuiltSrcs())
	if len(paths) > 0 {
		ctx.PropertyErrorf("srcs", "prebuilt libraries can only have one entry in srcs (the prebuilt path)")
	}
	prebuilt.baseCompiler.unstrippedOutputFile = srcPath
	return buildOutput{outputFile: srcPath}
}

func (prebuilt *prebuiltProcMacroDecorator) rustdoc(ctx ModuleContext, flags Flags,
	deps PathDeps) android.OptionalPath {

	return android.OptionalPath{}
}

func (prebuilt *prebuiltProcMacroDecorator) compilerDeps(ctx DepsContext, deps Deps) Deps {
	deps = prebuilt.baseCompiler.compilerDeps(ctx, deps)
	return deps
}

func (prebuilt *prebuiltProcMacroDecorator) nativeCoverage() bool {
	return false
}
