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
	"path/filepath"

	"android/soong/android"

	"github.com/google/blueprint"
)

//
// Objects (for crt*.o)
//

func init() {
	android.RegisterModuleType("cc_object", ObjectFactory)
	android.RegisterSdkMemberType(ccObjectSdkMemberType)
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
}

// cc_object runs the compiler without running the linker. It is rarely
// necessary, but sometimes used to generate .s files from .c files to use as
// input to a cc_genrule module.
func ObjectFactory() android.Module {
	module := newBaseModule(android.HostAndDeviceSupported, android.MultilibBoth)
	module.sanitize = &sanitize{}
	module.linker = &objectLinker{
		baseLinker: NewBaseLinker(module.sanitize),
	}
	module.compiler = NewBaseCompiler()

	// Clang's address-significance tables are incompatible with ld -r.
	module.compiler.appendCflags([]string{"-fno-addrsig"})

	module.stl = &stl{}
	module.sdkMemberTypes = []android.SdkMemberType{ccObjectSdkMemberType}
	return module.Init()
}

func (object *objectLinker) appendLdflags(flags []string) {
	panic(fmt.Errorf("appendLdflags on objectLinker not supported"))
}

func (object *objectLinker) linkerProps() []interface{} {
	return []interface{}{&object.Properties}
}

func (*objectLinker) linkerInit(ctx BaseModuleContext) {}

func (object *objectLinker) linkerDeps(ctx DepsContext, deps Deps) Deps {
	if ctx.useVndk() && ctx.toolchain().Bionic() {
		// Needed for VNDK builds where bionic headers aren't automatically added.
		deps.LateSharedLibs = append(deps.LateSharedLibs, "libc")
	}

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
			TransformBinaryPrefixSymbols(ctx, String(object.Properties.Prefix_symbols), outputFile,
				builderFlags, output)
			outputFile = output
		}
	} else {
		output := android.PathForModuleOut(ctx, ctx.ModuleName()+objectExtension)
		outputFile = output

		if String(object.Properties.Prefix_symbols) != "" {
			input := android.PathForModuleOut(ctx, "unprefixed", ctx.ModuleName()+objectExtension)
			TransformBinaryPrefixSymbols(ctx, String(object.Properties.Prefix_symbols), input,
				builderFlags, output)
			output = input
		}

		TransformObjsToObj(ctx, objs.objFiles, builderFlags, output, flags.LdFlagsDeps)
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

// SDK member support

var ccObjectSdkMemberType = &objectSdkMemberType{
	SdkMemberTypeBase: android.SdkMemberTypeBase{
		PropertyName: "native_objects",
		SupportsSdk:  true,
	},
}

type objectSdkMemberType struct {
	android.SdkMemberTypeBase
}

func (mt *objectSdkMemberType) AddDependencies(mctx android.BottomUpMutatorContext, dependencyTag blueprint.DependencyTag, names []string) {
	targets := mctx.MultiTargets()
	for _, obj := range names {
		name, version := StubsLibNameAndVersion(obj)
		for _, target := range targets {
			if version == "" {
				version = LatestStubsVersionFor(mctx.Config(), name)
			}
			mctx.AddFarVariationDependencies(append(target.Variations(), []blueprint.Variation{
				{Mutator: "version", Variation: version},
			}...), dependencyTag, name)
		}
	}
}

func (mt *objectSdkMemberType) IsInstance(module android.Module) bool {
	if m, ok := module.(*Module); ok {
		for _, allowableMemberType := range m.sdkMemberTypes {
			if allowableMemberType == mt {
				return true
			}
		}
	}
	return false
}

func (mt *objectSdkMemberType) AddPrebuiltModule(sdkModuleContext android.ModuleContext, builder android.SnapshotBuilder, member android.SdkMember) android.BpModule {
	return builder.AddPrebuiltModule(member, "cc_prebuilt_object")
}

func (mt *objectSdkMemberType) CreateVariantPropertiesStruct() android.SdkMemberProperties {
	return &objectInfoProperties{}
}

type objectInfoProperties struct {
	android.SdkMemberPropertiesBase

	// archType is not exported as if set (to a non default value) it is always arch specific.
	// This is "" for common properties.
	archType string

	// outputFile is not exported as it is always arch specific.
	outputFile android.Path
}

func (p *objectInfoProperties) PopulateFromVariant(variant android.SdkAware) {
	ccModule := variant.(*Module)
	p.archType = ccModule.Target().Arch.ArchType.String()
	p.outputFile = ccModule.OutputFile().Path()
}

// objectPathFor returns path to the object file, relative to <sdk_root>/<api_dir>.
func objectPathFor(obj objectInfoProperties) string {
	return filepath.Join(obj.OsPrefix(), obj.archType, "lib", obj.outputFile.Base())
}

func (p *objectInfoProperties) AddToPropertySet(sdkModuleContext android.ModuleContext, builder android.SnapshotBuilder, propertySet android.BpPropertySet) {
	if p.Compile_multilib != "" {
		propertySet.AddProperty("compile_multilib", p.Compile_multilib)
	}
	if p.outputFile != nil {
		propertySet.AddProperty("srcs", []string{objectPathFor(*p)})
		builder.CopyToSnapshot(p.outputFile, objectPathFor(*p))
	}
}
