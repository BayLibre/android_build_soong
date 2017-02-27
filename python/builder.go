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

package python

// This file contains Ninja build actions for building Python program.

import (
	"strings"

	"android/soong/android"
	"path/filepath"

	"github.com/google/blueprint"
	_ "github.com/google/blueprint/bootstrap"
)

var (
	pctx = android.NewPackageContext("android/soong/python")

	par = pctx.AndroidStaticRule("par",
		blueprint.RuleParams{
			Command: `touch $initFile && ` +
				`sed -e 's/%interpreter%/$interp/g' -e 's/%main%/$main/g' $template | cat > $stub && ` +
				`$parCmd -o $out $parArgs && rm -f $initFile && rm -f $stub`,
			CommandDeps: []string{"$parCmd"},
			Description: "par $out",
		},
		"initFile", "interp", "main", "template", "stub", "parCmd", "parArgs")
)

func init() {
	pctx.Import("github.com/google/blueprint/bootstrap")
	pctx.Import("android/soong/common")

	pctx.StaticVariable("parCmd", filepath.Join("${bootstrap.ToolDir}", "soong_zip"))
}

type fileListSpec struct {
	fileList     android.Path
	relativeRoot string
}

type parSpec struct {
	rootPrefix string

	fileListSpec []fileListSpec
}

func (p parSpec) soongParArgs() string {
	ret := "-P " + p.rootPrefix

	for _, ele := range p.fileListSpec {
		ret += " -C " + ele.relativeRoot + " -l " + ele.fileList.String()
	}

	return ret
}

func registerBuildActionForModuleFileList(ctx android.ModuleContext,
	name string, files android.Paths) android.Path {
	fileList := android.PathForModuleOut(ctx, name+".list")

	content := []string{}
	for _, ele := range files {
		content = append(content, ele.String())
	}

	ctx.ModuleBuild(pctx, android.ModuleBuildParams{
		Rule:      android.WriteFile,
		Output:    fileList,
		Implicits: files,
		Args: map[string]string{
			"content": strings.Join(content, "\n"),
		},
	})

	return fileList
}

func registerBuildActionForParFile(ctx android.ModuleContext,
	interpreter, main string, newPyPkgs []string, parSpecs []parSpec) {

	initFile := android.PathForModuleOut(ctx, "__init__.py").String()

	template := android.PathForSource(ctx,
		"build/soong/python/scripts/stub_template_host.txt")

	stub := android.PathForModuleOut(ctx, "__main__.py").String()

	parFile := android.PathForModuleOut(ctx, ctx.ModuleName()+".zip")

	implicits := android.Paths{template}

	for _, p := range parSpecs {
		for _, f := range p.fileListSpec {
			implicits = append(implicits, f.fileList)
		}
	}

	var parArgs string
	for _, p := range parSpecs {
		parArgs += p.soongParArgs()
	}
	parArgs += " -C " + strings.TrimSuffix(initFile, "__init__.py") +
		" -P " + " -f " + initFile
	for _, ele := range newPyPkgs {
		parArgs += " -P " + ele + " -f " + initFile
	}

	ctx.ModuleBuild(pctx, android.ModuleBuildParams{
		Rule:      par,
		Output:    parFile,
		Implicits: implicits,
		Args: map[string]string{
			"initFile": initFile,
			"interp":   strings.Replace(interpreter, "/", "\\/", -1),
			"main":     strings.Replace(main, "/", "\\/", -1),
			"template": template.String(),
			"stub":     stub,
			"parArgs":  parArgs,
		},
	})
}
