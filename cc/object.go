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

	"android/soong/android"
	"android/soong/bazel"
)

//
// Objects (for crt*.o)
//

func init() {
	android.RegisterModuleType("cc_object", ObjectFactory)
	android.RegisterSdkMemberType(ccObjectSdkMemberType)

	android.RegisterBp2BuildMutator("cc_object", ObjectBp2Build)
}

var ccObjectSdkMemberType = &librarySdkMemberType{
	SdkMemberTypeBase: android.SdkMemberTypeBase{
		PropertyName: "native_objects",
		SupportsSdk:  true,
	},
	prebuiltModuleType: "cc_prebuilt_object",
	linkTypes:          nil,
}

type objectLinker struct {
	*baseLinker
	Properties ObjectLinkerProperties
}

type ObjectLinkerProperties struct {
	// list of modules that should only provide headers for this module.
	Header_libs []string `android:"arch_variant,variant_prepend"`

	// names of other cc_object modules to link into this module using partial linking
	Objs []string `android:"arch_variant"`

	// if set, add an extra objcopy --prefix-symbols= step
	Prefix_symbols *string

	// if set, the path to a linker script to pass to ld -r when combining multiple object files.
	Linker_script *string `android:"path,arch_variant"`

	// Indicates that this module is a CRT object. CRT objects will be split
	// into a variant per-API level between min_sdk_version and current.
	Crt *bool
}

func newObject() *Module {
	module := newBaseModule(android.HostAndDeviceSupported, android.MultilibBoth)
	module.sanitize = &sanitize{}
	module.stl = &stl{}
	return module
}

// cc_object runs the compiler without running the linker. It is rarely
// necessary, but sometimes used to generate .s files from .c files to use as
// input to a cc_genrule module.
func ObjectFactory() android.Module {
	module := newObject()
	module.linker = &objectLinker{
		baseLinker: NewBaseLinker(module.sanitize),
	}
	module.compiler = NewBaseCompiler()

	// Clang's address-significance tables are incompatible with ld -r.
	module.compiler.appendCflags([]string{"-fno-addrsig"})

	module.sdkMemberTypes = []android.SdkMemberType{ccObjectSdkMemberType}

	module.AddProperties(&module.bazelProperties)

	return module.Init()
}

type bazelHeaderLibraryAttributes struct {
	Hdrs                 bazel.LabelList
	Includes             []string
	Include_prefix       string
	Strip_include_prefix string
}

type bazelHeaderLibrary struct {
	android.BazelTargetModuleBase
	bazelHeaderLibraryAttributes
}

func (m *bazelHeaderLibrary) Name() string {
	return m.BaseModuleName()
}

func (m *bazelHeaderLibrary) GenerateAndroidBuildActions(ctx android.ModuleContext) {}

func BazelHeaderLibraryFactory() android.Module {
	module := &bazelHeaderLibrary{}
	module.AddProperties(&module.bazelHeaderLibraryAttributes)
	android.InitBazelTargetModule(module)
	return module
}

type bazelObjectAttributes struct {
	Hdrs  bazel.LabelList
	Srcs  bazel.LabelList
	Copts []string
	Deps  []string
}

type bazelObject struct {
	android.BazelTargetModuleBase
	bazelObjectAttributes
}

func (m *bazelObject) Name() string {
	return m.BaseModuleName()
}

func (m *bazelObject) GenerateAndroidBuildActions(ctx android.ModuleContext) {}

func BazelObjectFactory() android.Module {
	module := &bazelObject{}
	module.AddProperties(&module.bazelObjectAttributes)
	android.InitBazelTargetModule(module)
	return module
}

func ObjectBp2Build(ctx android.TopDownMutatorContext) {
	m, ok := ctx.Module().(*Module)
	if !ok || !m.bazelProperties.Bazel_module.Bp2build_available {
		return
	}

	if m.compiler == nil {
		return
	}

	var copts []string
	var srcs []string
	var localIncludeDirs []string
	for _, props := range m.compiler.compilerProps() {
		if baseCompilerProps, ok := props.(*BaseCompilerProperties); ok {
			copts = baseCompilerProps.Cflags
			srcs = baseCompilerProps.Srcs
			localIncludeDirs = baseCompilerProps.Local_include_dirs
			break
		}
	}

	var deps []string
	for _, include := range localIncludeDirs {
		depName := m.Name() + "__" + include
		deps = append(deps, depName)

		props := bazel.NewBazelTargetModuleProperties(
			depName,
			"cc_library",
			"//build/bazel/rules:cc_library.bzl",
		)

		attrs := &bazelHeaderLibraryAttributes{
			Hdrs:                 android.BazelLabelForModuleSrc(ctx, []string{include + "/**/*.h"}),
			Includes:             []string{include},
			Include_prefix:       include,
			Strip_include_prefix: include,
		}

		ctx.CreateBazelTargetModule(BazelHeaderLibraryFactory, props, attrs)
	}

	attrs := &bazelObjectAttributes{
		Srcs:  android.BazelLabelForModuleSrc(ctx, srcs),
		Copts: copts,
		Deps:  deps,
	}

	props := bazel.NewBazelTargetModuleProperties(
		m.Name(),
		"cc_object",
		"//build/bazel/rules:cc_object.bzl",
	)

	// Create the BazelTargetModule.
	ctx.CreateBazelTargetModule(BazelObjectFactory, props, attrs)
}

func (object *objectLinker) appendLdflags(flags []string) {
	panic(fmt.Errorf("appendLdflags on objectLinker not supported"))
}

func (object *objectLinker) linkerProps() []interface{} {
	return []interface{}{&object.Properties}
}

func (*objectLinker) linkerInit(ctx BaseModuleContext) {}

func (object *objectLinker) linkerDeps(ctx DepsContext, deps Deps) Deps {
	deps.HeaderLibs = append(deps.HeaderLibs, object.Properties.Header_libs...)
	deps.ObjFiles = append(deps.ObjFiles, object.Properties.Objs...)
	return deps
}

func (object *objectLinker) linkerFlags(ctx ModuleContext, flags Flags) Flags {
	flags.Global.LdFlags = append(flags.Global.LdFlags, ctx.toolchain().ToolchainClangLdflags())

	if lds := android.OptionalPathForModuleSrc(ctx, object.Properties.Linker_script); lds.Valid() {
		flags.Local.LdFlags = append(flags.Local.LdFlags, "-Wl,-T,"+lds.String())
		flags.LdFlagsDeps = append(flags.LdFlagsDeps, lds.Path())
	}
	return flags
}

func (object *objectLinker) link(ctx ModuleContext,
	flags Flags, deps PathDeps, objs Objects) android.Path {

	objs = objs.Append(deps.Objs)

	var outputFile android.Path
	builderFlags := flagsToBuilderFlags(flags)

	if len(objs.objFiles) == 1 && String(object.Properties.Linker_script) == "" {
		outputFile = objs.objFiles[0]

		if String(object.Properties.Prefix_symbols) != "" {
			output := android.PathForModuleOut(ctx, ctx.ModuleName()+objectExtension)
			transformBinaryPrefixSymbols(ctx, String(object.Properties.Prefix_symbols), outputFile,
				builderFlags, output)
			outputFile = output
		}
	} else {
		output := android.PathForModuleOut(ctx, ctx.ModuleName()+objectExtension)
		outputFile = output

		if String(object.Properties.Prefix_symbols) != "" {
			input := android.PathForModuleOut(ctx, "unprefixed", ctx.ModuleName()+objectExtension)
			transformBinaryPrefixSymbols(ctx, String(object.Properties.Prefix_symbols), input,
				builderFlags, output)
			output = input
		}

		transformObjsToObj(ctx, objs.objFiles, builderFlags, output, flags.LdFlagsDeps)
	}

	ctx.CheckbuildFile(outputFile)
	return outputFile
}

func (object *objectLinker) unstrippedOutputFilePath() android.Path {
	return nil
}

func (object *objectLinker) nativeCoverage() bool {
	return true
}

func (object *objectLinker) coverageOutputFilePath() android.OptionalPath {
	return android.OptionalPath{}
}

func (object *objectLinker) object() bool {
	return true
}

func (object *objectLinker) isCrt() bool {
	return Bool(object.Properties.Crt)
}
