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

package java

// This file generates the final rules for compiling all C/C++.  All properties related to
// compiling should have been translated into builderFlags or another argument to the Transform*
// functions.

import (
	"path/filepath"
	"strings"

	"github.com/google/blueprint"
	"github.com/google/blueprint/pathtools"

	"android/soong/common"
)

func init() {
	pctx.VariableFunc("aidlCmd", func(c interface{}) (string, error) {
		return c.(common.Config).HostBinTool("aidl")
	})
	pctx.VariableConfigMethod("srcDir", common.Config.SrcDir)

	pctx.StaticVariable("platformAidlIncludes",
		common.JoinWithPrefix([]string{
			"frameworks/base/core/java",
			"frameworks/base/graphics/java",
			"frameworks/base/location/java",
			"frameworks/base/media/java",
			"frameworks/base/media/mca/effect/java",
			"frameworks/base/media/mca/filterfw/java",
			"frameworks/base/media/mca/filterpacks/java",
			"frameworks/base/drm/java",
			"frameworks/base/opengl/java",
			"frameworks/base/sax/java",
			"frameworks/base/telecomm/java",
			"frameworks/base/telephony/java",
			"frameworks/base/wifi/java",
			"frameworks/base/keystore/java",
			"frameworks/base/rs/java",
		}, "-I${srcDir}/"))
}

var (
	aidl = pctx.StaticRule("aidl",
		blueprint.RuleParams{
			Command:     "$aidlCmd -d$depFile $aidlFlags $in $out",
			Description: "aidl $out",
		},
		"depFile", "aidlFlags")

	aidlPreprocessRule = pctx.StaticRule("aidlPreprocess",
		blueprint.RuleParams{
			Command:     "$aidlCmd --preprocess $out $in",
			Description: "preprocess aidl $out",
		})
)

func genAidl(ctx common.AndroidModuleContext, aidlFile, aidlFlags string) string {
	javaFile := strings.TrimPrefix(aidlFile, common.ModuleSrcDir(ctx))
	javaFile = filepath.Join(common.ModuleGenDir(ctx), javaFile)
	javaFile = pathtools.ReplaceExtension(javaFile, "java")
	depFile := javaFile + ".d"

	ctx.Build(pctx, blueprint.BuildParams{
		Rule:      aidl,
		Outputs:   []string{javaFile},
		Inputs:    []string{aidlFile},
		Implicits: []string{"$aidlCmd"},
		Args: map[string]string{
			"depFile":   depFile,
			"aidlFlags": aidlFlags,
		},
	})

	return javaFile
}

func genSources(ctx common.AndroidModuleContext, srcFiles []string,
	flags javaBuilderFlags) []string {

	for i, srcFile := range srcFiles {
		switch filepath.Ext(srcFile) {
		case ".aidl":
			javaFile := genAidl(ctx, srcFile, flags.aidlFlags)
			srcFiles[i] = javaFile
		}
	}

	return srcFiles
}

func preprocessAidl(ctx common.AndroidModuleContext, srcFiles []string) string {
	outputFile := filepath.Join(common.ModuleOutDir(ctx), ctx.ModuleName()+".aidl")
	ctx.Build(pctx, blueprint.BuildParams{
		Rule:      aidlPreprocessRule,
		Outputs:   []string{outputFile},
		Inputs:    srcFiles,
		Implicits: []string{"$aidlCmd"},
	})

	return outputFile
}
