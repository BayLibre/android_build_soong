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

import (
	"strconv"
	"strings"

	"github.com/google/blueprint"
	"github.com/google/blueprint/pathtools"

	"android/soong/android"
)

func init() {
	pctx.HostBinToolVariable("syspropCmd", "sysprop_java")
	pctx.SourcePathVariable("logtagsCmd", "build/make/tools/java-event-log-tags.py")
	pctx.SourcePathVariable("mergeLogtagsCmd", "build/make/tools/merge-event-log-tags.py")
	pctx.SourcePathVariable("logtagsLib", "build/make/tools/event_log_tags.py")
}

var (

	logtags = pctx.AndroidStaticRule("logtags",
		blueprint.RuleParams{
			Command:     "$logtagsCmd -o $out $in",
			CommandDeps: []string{"$logtagsCmd", "$logtagsLib"},
		})

	mergeLogtags = pctx.AndroidStaticRule("mergeLogtags",
		blueprint.RuleParams{
			Command:     "$mergeLogtagsCmd -o $out $in",
			CommandDeps: []string{"$mergeLogtagsCmd", "$logtagsLib"},
		})

	sysprop = pctx.AndroidStaticRule("sysprop",
		blueprint.RuleParams{
			Command: `rm -rf $out.tmp && mkdir -p $out.tmp && ` +
				`$syspropCmd --scope $scope --java-output-dir $out.tmp $in && ` +
				`${config.SoongZipCmd} -jar -o $out -C $out.tmp -D $out.tmp && rm -rf $out.tmp`,
			CommandDeps: []string{
				"$syspropCmd",
				"${config.SoongZipCmd}",
			},
		}, "scope")
)

func genAidl(ctx android.ModuleContext, aidlFiles android.Paths, aidlFlags string, deps android.Paths) android.Paths {
	// Shard proto files into groups of 100 to avoid having to recompile all of them if one changes and to avoid
	// hitting command line length limits.
	shards := shardPaths(aidlFiles, 50)

	srcJarFiles := make(android.Paths, 0, len(shards))

	for i, shard := range shards {
		srcJarFile := android.PathForModuleGen(ctx, "aidl", "aidl"+strconv.Itoa(i)+".srcjar")
		srcJarFiles = append(srcJarFiles, srcJarFile)

		outDir := srcJarFile.ReplaceExtension(ctx, "tmp")

		rule := android.NewRuleBuilder()

		rule.Command().Text("rm -rf").Flag(outDir.String())
		rule.Command().Text("mkdir -p").Flag(outDir.String())

		for _, aidlFile := range shard {
			depFile := srcJarFile.InSameDir(ctx, aidlFile.String()+".d")
			javaFile := outDir.Join(ctx, pathtools.ReplaceExtension(aidlFile.String(), "java"))
			rule.Command().
				Tool(ctx.Config().HostToolPath(ctx, "aidl")).
				FlagWithDepFile("-d", depFile).
				Flag(aidlFlags).
				Input(aidlFile).
				Output(javaFile).
				Implicits(deps)
			rule.Temporary(javaFile)
		}

		rule.Command().
			Tool(ctx.Config().HostToolPath(ctx, "soong_zip")).
			Flag("-srcjar").
			Flag("-write_if_changed").
			FlagWithOutput("-o ", srcJarFile).
			FlagWithArg("-C ", outDir.String()).
			FlagWithArg("-D ", outDir.String())

		rule.Command().Text("rm -rf").Flag(outDir.String())

		rule.Restat()

		rule.Build(pctx, ctx, "aidl_"+strconv.Itoa(i), "aidl "+strconv.Itoa(i))
	}

	return srcJarFiles
}

func genLogtags(ctx android.ModuleContext, logtagsFile android.Path) android.Path {
	javaFile := android.GenPathWithExt(ctx, "logtags", logtagsFile, "java")

	ctx.Build(pctx, android.BuildParams{
		Rule:        logtags,
		Description: "logtags " + logtagsFile.Rel(),
		Output:      javaFile,
		Input:       logtagsFile,
	})

	return javaFile
}

func genSysprop(ctx android.ModuleContext, syspropFile android.Path, scope string) android.Path {
	srcJarFile := android.GenPathWithExt(ctx, "sysprop", syspropFile, "srcjar")

	ctx.Build(pctx, android.BuildParams{
		Rule:        sysprop,
		Description: "sysprop_java " + syspropFile.Rel(),
		Output:      srcJarFile,
		Input:       syspropFile,
		Args: map[string]string{
			"scope": scope,
		},
	})

	return srcJarFile
}

func genAidlIncludeFlags(srcFiles android.Paths) string {
	var baseDirs []string
	for _, srcFile := range srcFiles {
		if srcFile.Ext() == ".aidl" {
			baseDir := strings.TrimSuffix(srcFile.String(), srcFile.Rel())
			if baseDir != "" && !android.InList(baseDir, baseDirs) {
				baseDirs = append(baseDirs, baseDir)
			}
		}
	}
	return android.JoinWithPrefix(baseDirs, " -I")
}

func (j *Module) genSources(ctx android.ModuleContext, srcFiles android.Paths,
	flags javaBuilderFlags) android.Paths {

	outSrcFiles := make(android.Paths, 0, len(srcFiles))
	var protoSrcs android.Paths
	var aidlSrcs android.Paths

	for _, srcFile := range srcFiles {
		switch srcFile.Ext() {
		case ".aidl":
			aidlSrcs = append(aidlSrcs, srcFile)
		case ".logtags":
			j.logtagsSrcs = append(j.logtagsSrcs, srcFile)
			javaFile := genLogtags(ctx, srcFile)
			outSrcFiles = append(outSrcFiles, javaFile)
		case ".proto":
			protoSrcs = append(protoSrcs, srcFile)
		case ".sysprop":
			// internal scope contains all properties
			// public scope only contains public properties
			// use public if the owner is different from client
			scope := "internal"
			if j.properties.Sysprop.Platform != nil {
				isProduct := ctx.ProductSpecific()
				isVendor := ctx.SocSpecific()
				isOwnerPlatform := Bool(j.properties.Sysprop.Platform)

				if isProduct {
					// product can't own any sysprop_library now, so product must use public scope
					scope = "public"
				} else if isVendor && !isOwnerPlatform {
					// vendor and odm can't use system's internal property.
					scope = "public"
				}

				// We don't care about clients under system.
				// They can't use sysprop_library owned by other partitions.
			}
			srcJarFile := genSysprop(ctx, srcFile, scope)
			outSrcFiles = append(outSrcFiles, srcJarFile)
		default:
			outSrcFiles = append(outSrcFiles, srcFile)
		}
	}

	if len(protoSrcs) > 0 {
		srcJarFiles := genProto(ctx, protoSrcs, flags.proto)
		outSrcFiles = append(outSrcFiles, srcJarFiles...)
	}

	if len(aidlSrcs) > 0 {
		srcJarFiles := genAidl(ctx, aidlSrcs, flags.aidlFlags, flags.aidlDeps)
		outSrcFiles = append(outSrcFiles, srcJarFiles...)
	}

	return outSrcFiles
}

func LogtagsSingleton() android.Singleton {
	return &logtagsSingleton{}
}

type logtagsProducer interface {
	logtags() android.Paths
}

type logtagsSingleton struct{}

func (l *logtagsSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	var allLogtags android.Paths
	ctx.VisitAllModules(func(module android.Module) {
		if logtags, ok := module.(logtagsProducer); ok {
			allLogtags = append(allLogtags, logtags.logtags()...)
		}
	})

	ctx.Build(pctx, android.BuildParams{
		Rule:        mergeLogtags,
		Description: "merge logtags",
		Output:      android.PathForIntermediates(ctx, "all-event-log-tags.txt"),
		Inputs:      allLogtags,
	})
}
