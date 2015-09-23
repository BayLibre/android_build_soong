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

// This file generates the final rules for compiling all Java.  All properties related to
// compiling should have been translated into javaBuilderFlags or another argument to the Transform*
// functions.

import (
	"strings"

	"github.com/google/blueprint"

	"android/soong/common"
)

var (
	aaptCreateResourceJavaFile = pctx.StaticRule("aaptCreateResourceJavaFile",
		blueprint.RuleParams{
			Command: `rm -rf "$javaDir" && mkdir -p "$javaDir" && ` +
				`$aaptCmd package -m $aaptFlags -P $publicResourcesFile -G $proguardOptionsFile ` +
				`-J $javaDir || ( rm -rf "$javaDir/*"; exit 41 ) && ` +
				`find $javaDir -name "*.java" > $javaFileList`,
			CommandDeps: []string{"$aaptCmd"},
			Description: "aapt create R.java $out",
		},
		"aaptFlags", "publicResourcesFile", "proguardOptionsFile", "javaDir", "javaFileList")

	aaptCreateAssetsPackage = pctx.StaticRule("aaptCreateAssetsPackage",
		blueprint.RuleParams{
			Command:     `rm -f $out && $aaptCmd package $aaptFlags -F $out`,
			CommandDeps: []string{"$aaptCmd"},
			Description: "aapt export package $out",
		},
		"aaptFlags", "publicResourcesFile", "proguardOptionsFile", "javaDir", "javaFileList")

	aaptAddResources = pctx.StaticRule("aaptAddResources",
		blueprint.RuleParams{
			// TODO: add-jni-shared-libs-to-package
			Command:     `cp -f $in $out.tmp && $aaptCmd package -u $aaptFlags -F $out.tmp && mv $out.tmp $out`,
			CommandDeps: []string{"$aaptCmd"},
			Description: "aapt package $out",
		},
		"aaptFlags")

	zipalign = pctx.StaticRule("zipalign",
		blueprint.RuleParams{
			Command:     `$zipalignCmd -f $zipalignFlags 4 $in $out`,
			CommandDeps: []string{"$zipalignCmd"},
			Description: "zipalign $out",
		},
		"zipalignFlags")

	signapk = pctx.StaticRule("signapk",
		blueprint.RuleParams{
			Command:     `java -jar $signapkCmd $certificates $in $out`,
			CommandDeps: []string{"$signapkCmd"},
			Description: "signapk $out",
		},
		"certificates")

	androidManifestMerger = pctx.StaticRule("androidManifestMerger",
		blueprint.RuleParams{
			Command: "java -classpath $androidManifestMergerCmd com.android.manifmerger.Main merge " +
				"--main $in --libs $libsManifests --out $out",
			CommandDeps: []string{"$androidManifestMergerCmd"},
			Description: "merge manifest files $out",
		},
		"libsManifests")
)

func init() {
	pctx.StaticVariable("androidManifestMergerCmd", "${srcDir}/prebuilts/devtools/tools/lib/manifest-merger.jar")
	pctx.VariableFunc("aaptCmd", common.HostBinToolVariableFunc("aapt"))
	pctx.VariableFunc("zipalignCmd", common.HostBinToolVariableFunc("zipalign"))
	pctx.VariableFunc("signapkCmd", common.HostJavaToolVariableFunc("signapk.jar"))
}

func CreateResourceJavaFiles(ctx common.AndroidModuleContext, flags []string,
	deps common.Paths) (common.Path, common.Path, common.Path) {
	javaDir := common.PathForModuleGen(ctx, "R")
	javaFileList := common.PathForModuleOut(ctx, "R.filelist")
	publicResourcesFile := common.PathForModuleOut(ctx, "public_resources.xml")
	proguardOptionsFile := common.PathForModuleOut(ctx, "proguard.options")

	ctx.ModuleBuild(pctx, common.ModuleBuildParams{
		Rule:      aaptCreateResourceJavaFile,
		Outputs:   common.Paths{publicResourcesFile, proguardOptionsFile, javaFileList},
		Implicits: deps,
		Args: map[string]string{
			"aaptFlags":           strings.Join(flags, " "),
			"publicResourcesFile": publicResourcesFile.String(),
			"proguardOptionsFile": proguardOptionsFile.String(),
			"javaDir":             javaDir.String(),
			"javaFileList":        javaFileList.String(),
		},
	})

	return publicResourcesFile, proguardOptionsFile, javaFileList
}

func CreateExportPackage(ctx common.AndroidModuleContext, flags []string, deps common.Paths) common.ModuleOutPath {
	outputFile := common.PathForModuleOut(ctx, "package-export.apk")

	ctx.ModuleBuild(pctx, common.ModuleBuildParams{
		Rule:      aaptCreateAssetsPackage,
		Outputs:   common.Paths{outputFile},
		Implicits: deps,
		Args: map[string]string{
			"aaptFlags": strings.Join(flags, " "),
		},
	})

	return outputFile
}

func CreateAppPackage(ctx common.AndroidModuleContext, flags []string, jarFile common.Path,
	certificates []string) common.Path {

	resourceApk := common.PathForModuleOut(ctx, "resources.apk")

	ctx.ModuleBuild(pctx, common.ModuleBuildParams{
		Rule:    aaptAddResources,
		Outputs: common.Paths{resourceApk},
		Inputs:  common.Paths{jarFile},
		Args: map[string]string{
			"aaptFlags": strings.Join(flags, " "),
		},
	})

	signedApk := common.PathForModuleOut(ctx, "signed.apk")

	var certificateArgs []string
	for _, c := range certificates {
		certificateArgs = append(certificateArgs, c+".x509.pem", c+".pk8")
	}

	ctx.ModuleBuild(pctx, common.ModuleBuildParams{
		Rule:    signapk,
		Outputs: common.Paths{signedApk},
		Inputs:  common.Paths{resourceApk},
		Args: map[string]string{
			"certificates": strings.Join(certificateArgs, " "),
		},
	})

	outputFile := common.PathForModuleOut(ctx, "package.apk")

	ctx.ModuleBuild(pctx, common.ModuleBuildParams{
		Rule:    zipalign,
		Outputs: common.Paths{outputFile},
		Inputs:  common.Paths{signedApk},
		Args: map[string]string{
			"zipalignFlags": "",
		},
	})

	return outputFile
}
