// Copyright 2015 Google Inc. All rights reserved.
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

package genrule

import (
	"path/filepath"

	"github.com/google/blueprint"
	"github.com/google/blueprint/pathtools"

	"android/soong/common"
)

var (
	pctx = blueprint.NewPackageContext("android/soong/genrule")
)

func init() {
	pctx.VariableConfigMethod("srcDir", common.Config.SrcDir)
	pctx.VariableFunc("path", func(config interface{}) (string, error) {
		aconfig := config.(common.Config)
		return aconfig.Getenv("PATH") + string(filepath.ListSeparator) + aconfig.HostBin(), nil
	})
}

type SourceFileGenerator interface {
	GeneratedSourceFiles() []string
}

type HostToolProvider interface {
	HostToolPath() string
}

func GenSrcsFactory() (blueprint.Module, []interface{}) {
	module := &genSrcs{}

	return common.InitAndroidModule(module, &module.properties)
}

type genSrcs struct {
	common.AndroidModuleBase

	properties struct {
		// cmd: command to run on each input file.  Available variables for substitution:
		// $in: an input file
		// $out: the corresponding output file
		// $srcDir: the root directory of the source tree
		// The host bin directory will be in the path, any binaries used must be listed in deps
		Cmd string

		// srcs: list of input files
		Srcs []string

		// output_extension: extension that will be substituted for each output file
		Output_extension string
	}
	outputFiles []string
}

func (g *genSrcs) GenerateAndroidBuildActions(ctx common.AndroidModuleContext) {
	rule := ctx.Rule(pctx, "genSrcs", blueprint.RuleParams{
		Command: "PATH=$path " + g.properties.Cmd,
	})

	deps := collectDeps(ctx)
	srcFiles := common.ExpandSources(ctx, g.properties.Srcs)

	g.outputFiles = make([]string, 0, len(srcFiles))

	for _, in := range srcFiles {
		out := pathtools.ReplaceExtension(in, g.properties.Output_extension)
		out = filepath.Join(common.ModuleGenDir(ctx), out)
		g.outputFiles = append(g.outputFiles, out)
		ctx.Build(pctx, blueprint.BuildParams{
			Rule:      rule,
			Inputs:    []string{in},
			Implicits: deps,
			Outputs:   []string{out},
			// TODO: visit dependencies to add implicit dependencies on required tools
		})
	}
}

var _ SourceFileGenerator = (*genSrcs)(nil)

func (g *genSrcs) GeneratedSourceFiles() []string {
	return g.outputFiles
}

func GenRuleFactory() (blueprint.Module, []interface{}) {
	module := &genRule{}

	return common.InitAndroidModule(module, &module.properties)
}

type genRule struct {
	common.AndroidModuleBase

	properties struct {
		// cmd: command to run on each input file.  Available variables for substitution:
		// $in: an input file
		// $out: the corresponding output file
		// $srcDir: the root directory of the source tree
		// The host bin directory will be in the path, any binaries used must be listed in deps
		Cmd string

		// out: output file that will be generated
		Out string
	}
	outputFiles []string
}

func (g *genRule) GenerateAndroidBuildActions(ctx common.AndroidModuleContext) {
	rule := ctx.Rule(pctx, "genRule", blueprint.RuleParams{
		Command: "PATH=$path " + g.properties.Cmd,
	})

	deps := collectDeps(ctx)
	out := filepath.Join(common.ModuleGenDir(ctx), g.properties.Out)
	g.outputFiles = []string{out}

	ctx.Build(pctx, blueprint.BuildParams{
		Rule:      rule,
		Implicits: deps,
		Outputs:   []string{out},
	})
}

var _ SourceFileGenerator = (*genRule)(nil)

func (g *genRule) GeneratedSourceFiles() []string {
	return g.outputFiles
}

func collectDeps(ctx common.AndroidModuleContext) []string {
	var deps []string
	ctx.VisitDirectDeps(func(module blueprint.Module) {
		if t, ok := module.(HostToolProvider); ok {
			p := t.HostToolPath()
			if p != "" {
				deps = append(deps, t.HostToolPath())
			} else {
				ctx.ModuleErrorf("host tool %q missing output file", ctx.OtherModuleName(module))
			}
		} else {
			ctx.ModuleErrorf("unknown dependency %q", ctx.OtherModuleName(module))
		}
	})

	return deps
}
