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
			Description: "build par $out",
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

	fileListSpecs []fileListSpec
}

func (p parSpec) soongParArgs() string {
	ret := "-P " + p.rootPrefix

	for _, spec := range p.fileListSpecs {
		ret += " -C " + spec.relativeRoot + " -l " + spec.fileList.String()
	}

	return ret
}

func registerBuildActionForModuleFileList(ctx android.ModuleContext,
	name string, files android.Paths) android.Path {
	fileList := android.PathForModuleOut(ctx, name+".list")

	content := []string{}
	for _, file := range files {
		content = append(content, file.String())
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
	interpreter, main, parName string, newPyPkgs []string, parSpecs []parSpec) android.Path {

	initFile := android.PathForModuleOut(ctx, initFileName).String()

	template := android.PathForSource(ctx, stubTemplateHost)

	stub := android.PathForModuleOut(ctx, mainFileName).String()

	parFile := android.PathForModuleOut(ctx, parName)

	implicits := android.Paths{template}

	for _, p := range parSpecs {
		for _, f := range p.fileListSpecs {
			implicits = append(implicits, f.fileList)
		}
	}

	var parArgs string
	parArgs += "-C " + strings.TrimSuffix(stub, mainFileName) + " -f " + stub
	parArgs += " "
	parArgs += "-C " + strings.TrimSuffix(initFile, initFileName) + " -f " + initFile
	parArgs += " "
	for _, pkg := range newPyPkgs {
		parArgs += " -P " + pkg + " -f " + initFile
	}
	parArgs += " "
	for _, p := range parSpecs {
		parArgs += p.soongParArgs()
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

	return parFile
}
