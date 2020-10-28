// Copyright 2017 Google Inc. All rights reserved.
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

package android

import (
	"strings"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

// TODO(ccross): protos are often used to communicate between multiple modules.  If the only
// way to convert a proto to source is to reference it as a source file, and external modules cannot
// reference source files in other modules, then every module that owns a proto file will need to
// export a library for every type of external user (lite vs. full, c vs. c++ vs. java).  It would
// be better to support a proto module type that exported a proto file along with some include dirs,
// and then external modules could depend on the proto module but use their own settings to
// generate the source.

type ProtoFlags struct {
	Flags                 []string
	CanonicalPathFromRoot bool
	Dir                   ModuleGenPath
	SubDir                ModuleGenPath
	OutTypeFlag           string
	OutParams             []string
	Deps                  Paths
}

type protoDependencyTag struct {
	blueprint.BaseDependencyTag
	name string
}

var ProtoPluginDepTag = protoDependencyTag{name: "plugin"}

func ProtoDeps(ctx BottomUpMutatorContext, p *ProtoProperties) {
	if String(p.Proto.Plugin) != "" && String(p.Proto.Type) != "" {
		ctx.ModuleErrorf("only one of proto.type and proto.plugin can be specified.")
	}

	if plugin := String(p.Proto.Plugin); plugin != "" {
		ctx.AddFarVariationDependencies(ctx.Config().BuildOSTarget.Variations(),
			ProtoPluginDepTag, "protoc-gen-"+plugin)
	}
}

func GetProtoFlags(ctx ModuleContext, p *ProtoProperties) ProtoFlags {
	var flags []string
	var deps Paths

	if len(p.Proto.Local_include_dirs) > 0 {
		localProtoIncludeDirs := PathsForModuleSrc(ctx, p.Proto.Local_include_dirs)
		flags = append(flags, JoinWithPrefix(localProtoIncludeDirs.Strings(), "-I"))
	}
	if len(p.Proto.Include_dirs) > 0 {
		rootProtoIncludeDirs := PathsForSource(ctx, p.Proto.Include_dirs)
		flags = append(flags, JoinWithPrefix(rootProtoIncludeDirs.Strings(), "-I"))
	}

	ctx.VisitDirectDepsWithTag(ProtoPluginDepTag, func(dep Module) {
		if hostTool, ok := dep.(HostToolProvider); !ok || !hostTool.HostToolPath().Valid() {
			ctx.PropertyErrorf("proto.plugin", "module %q is not a host tool provider",
				ctx.OtherModuleName(dep))
		} else {
			plugin := String(p.Proto.Plugin)
			deps = append(deps, hostTool.HostToolPath().Path())
			flags = append(flags, "--plugin=protoc-gen-"+plugin+"="+hostTool.HostToolPath().String())
		}
	})

	var protoOutFlag string
	if plugin := String(p.Proto.Plugin); plugin != "" {
		protoOutFlag = "--" + plugin + "_out"
	}

	return ProtoFlags{
		Flags:                 flags,
		Deps:                  deps,
		OutTypeFlag:           protoOutFlag,
		CanonicalPathFromRoot: proptools.BoolDefault(p.Proto.Canonical_path_from_root, true),
		Dir:                   PathForModuleGen(ctx, "proto"),
		SubDir:                PathForModuleGen(ctx, "proto", ctx.ModuleDir()),
	}
}

type ProtoProperties struct {
	Proto struct {
		// Proto generator type.  C++: full or lite.  Java: micro, nano, stream, or lite.
		Type *string `android:"arch_variant"`

		// Proto plugin to use as the generator.  Must be a cc_binary_host module.
		Plugin *string `android:"arch_variant"`

		// list of directories that will be added to the protoc include paths.
		Include_dirs []string

		// list of directories relative to the bp file that will
		// be added to the protoc include paths.
		Local_include_dirs []string

		// whether to identify the proto files from the root of the
		// source tree (the original method in Android, useful for
		// android-specific protos), or relative from where they were
		// specified (useful for external/third party protos).
		//
		// This defaults to true today, but is expected to default to
		// false in the future.
		Canonical_path_from_root *bool
	} `android:"arch_variant"`
}

func MultiProtoRule(ctx ModuleContext, rule *RuleBuilder, protoFiles Paths, flags ProtoFlags, deps Paths,
	outDir WritablePath, depFile WritablePath, outputs WritablePaths) {
	GenProtoRule(ctx, rule, protoFiles, flags, deps, outDir, depFile, outputs)
}

func ProtoRule(ctx ModuleContext, rule *RuleBuilder, protoFile Path, flags ProtoFlags, deps Paths,
	outDir WritablePath, depFile WritablePath, outputs WritablePaths) {
	GenProtoRule(ctx, rule, Paths{protoFile}, flags, deps, outDir, depFile, outputs)
}

func GenProtoRule(ctx ModuleContext, rule *RuleBuilder, protoFiles Paths, flags ProtoFlags, deps Paths,
	outDir WritablePath, depFile WritablePath, outputs WritablePaths) {

	var protoBases []string
	if flags.CanonicalPathFromRoot {
		protoBases = append(protoBases, ".")
	} else {
		for _, proto := range protoFiles {
			rel := proto.Rel()
			protoBases = append(protoBases, strings.TrimSuffix(proto.String(), rel))
		}
		protoBases = FirstUniqueStrings(protoBases)
	}

	cmd := rule.Command()

	cmd.BuiltTool(ctx, "aprotoc").
		FlagWithArg(flags.OutTypeFlag+"=", strings.Join(flags.OutParams, ",")+":"+outDir.String()).
		FlagForEachArg("-I ", protoBases).
		Flags(flags.Flags).
		Inputs(protoFiles).
		Implicits(deps).
		ImplicitOutputs(outputs)

	if len(protoFiles) == 1 {
		// dependency_out only supports a single input, so only pass this flag when we have a single input.
		// Error message: Can only process one input file when using --dependency_out=FILE.
		cmd.FlagWithDepFile("--dependency_out=", depFile)
		rule.Command().
			BuiltTool(ctx, "dep_fixer").Flag(depFile.String())
	}
}
