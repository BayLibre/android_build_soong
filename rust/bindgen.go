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
	"github.com/google/blueprint/proptools"

	"android/soong/android"
)

var ()

func init() {
	android.RegisterModuleType("rust_bindgen_rlib", RustBindgenFactory)
}

type BindgenProperties struct {
	// Rust file that invokes bindgen as a library. This file will be built
	// into a binary and called as `$tool [input files] -- [extra cflags]`,
	// and it should write a Rust bindings to standard output.
	Srcs []string `android:"path"`

	// path to the header file(s) to generate bindings from
	Headers []string `android:"path,arch_variant"`
}

type bindgenDecorator struct {
	*libraryDecorator

	Properties           BindgenProperties
	unstrippedOutputFile android.Path
}

// rust_bindgen builds Rust bindings to C/C++ using rust-bindgen
func RustBindgenFactory() android.Module {
	module, libraryDecorator := NewRustLibrary(android.HostAndDeviceSupported)
	libraryDecorator.BuildOnlyRlib()

	bindgen := &bindgenDecorator{
		libraryDecorator: libraryDecorator,
	}
	module.compiler = bindgen
	module.Init()

	android.AddLoadHook(module, func(ctx android.LoadHookContext) { bindgen.createGeneratorModule(ctx) })
	return module
}

func (bindgen *bindgenDecorator) generatorToolName(moduleName string) string {
	return moduleName + "-bindgen"
}

func (bindgen *bindgenDecorator) createGeneratorModule(mctx android.LoadHookContext) {
	if module, ok := mctx.Module().(*Module); ok {
		props := struct {
			Name  *string
			Srcs  []string
			Rlibs []string
		}{}
		props.Name = proptools.StringPtr(bindgen.generatorToolName(module.BaseModuleName()))
		props.Srcs = bindgen.Properties.Srcs
		props.Rlibs = []string{"libbindgen"}
		mctx.CreateModule(RustBinaryHostFactory, &props)
	}
}

func (bindgen *bindgenDecorator) compilerProps() []interface{} {
	return append(bindgen.baseCompiler.compilerProps(),
		&bindgen.libraryDecorator.MutatedProperties,
		&bindgen.Properties)
}

func (bindgen *bindgenDecorator) compilerFlags(ctx ModuleContext, flags Flags) Flags {
	flags = bindgen.libraryDecorator.compilerFlags(ctx, flags)
	// Ignore style warnings in generated code
	flags.RustFlags = append(flags.RustFlags, "-A nonstandard-style")
	return flags
}

func (bindgen *bindgenDecorator) compile(ctx ModuleContext, flags Flags, deps PathDeps) android.Path {
	baseModuleName := ctx.baseModuleName()
	bindingsFile := android.PathForModuleGen(ctx, "bindings.rs")

	includes := make([]string, len(deps.IncludeDirs))
	for i, v := range deps.IncludeDirs {
		includes[i] = v.String()
	}

	headers := android.PathsForModuleSrc(ctx, bindgen.Properties.Headers)
	ctx.Build(pctx, android.BuildParams{
		Rule:        bindingsGenerator,
		Description: "bindgen " + baseModuleName,
		Inputs:      headers,
		Output:      bindingsFile,
		Args: map[string]string{
			"generator":  ctx.Config().HostToolPath(ctx, bindgen.generatorToolName(baseModuleName)).String(),
			"extraFlags": android.JoinWithPrefix(includes, "-I"),
		},
	})

	// We can't call libraryDecorator.compile() here because it expects all
	// source paths to be under the source tree, not in out/
	flags.RustFlags = append(flags.RustFlags, deps.depFlags...)

	fileName := bindgen.getStem(ctx) + ctx.toolchain().RlibSuffix()
	outputFile := android.PathForModuleOut(ctx, fileName)

	outputs := TransformSrctoRlib(ctx, bindingsFile, deps, flags, outputFile, deps.linkDirs)
	bindgen.coverageFile = outputs.coverageFile
	bindgen.unstrippedOutputFile = outputFile
	return outputFile
}
