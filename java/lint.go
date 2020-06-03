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

package java

import (
	"fmt"
	"sort"

	"android/soong/android"
)

type linter struct {
	name                string
	manifest            android.Path
	mergedManifest      android.Path
	srcs                android.Paths
	srcJars             android.Paths
	resources           android.Paths
	classpath           android.Paths
	classes             android.Path
	extraLintCheckJars  android.Paths
	test                bool
	library             bool
	compileSdkVersion   string
	javaLanguageLevel   string
	kotlinLanguageLevel string
	fatalChecks         []string
	errorChecks         []string
	warningChecks       []string
	disableChecks       []string
	flags               []string
	outputs             lintOutputs
}

type lintOutputs struct {
	html android.ModuleOutPath
	text android.ModuleOutPath
	xml  android.ModuleOutPath
}

func (l *linter) writeLintProjectXML(ctx android.ModuleContext,
	rule *android.RuleBuilder) (projectXMLPath, configXMLPath, cacheDir android.WritablePath, deps android.Paths) {

	var resourcesList android.WritablePath
	if len(l.resources) > 0 {
		// The list of resources may be too long to put on the command line, but
		// we can't use the rsp file because it is already being used for srcs.
		// Insert a second rule to write out the list of resources to a file.
		resourcesList = android.PathForModuleOut(ctx, "lint", "resources.list")
		resListRule := android.NewRuleBuilder()
		resListRule.Command().Text("cp").FlagWithRspFileInputList("", l.resources).Output(resourcesList)
		resListRule.Build(pctx, ctx, "lint_resources_list", "lint resources list")
		deps = append(deps, l.resources...)
	}

	projectXMLPath = android.PathForModuleOut(ctx, "lint", "project.xml")
	// Lint looks for a lint.xml file next to the project.xml file, give it one.
	configXMLPath = android.PathForModuleOut(ctx, "lint", "lint.xml")
	cacheDir = android.PathForModuleOut(ctx, "lint", "cache")

	srcJarDir := android.PathForModuleOut(ctx, "lint-srcjars")
	srcJarList := zipSyncCmd(ctx, rule, srcJarDir, l.srcJars)

	cmd := rule.Command().
		BuiltTool(ctx, "lint-project-xml").
		FlagWithOutput("--project_out ", projectXMLPath).
		FlagWithOutput("--config_out ", configXMLPath).
		FlagWithArg("--name ", ctx.ModuleName())

	if l.library {
		cmd.Flag("--library")
	}
	if l.test {
		cmd.Flag("--test")
	}
	if l.manifest != nil {
		deps = append(deps, l.manifest)
		cmd.FlagWithArg("--manifest ", l.manifest.String())
	}
	if l.mergedManifest != nil {
		deps = append(deps, l.mergedManifest)
		cmd.FlagWithArg("--merged_manifest ", l.mergedManifest.String())
	}

	cmd.FlagWithRspFileInputList("--srcs ", l.srcs)
	deps = append(deps, l.srcs...)

	cmd.FlagWithInput("--generated_srcs ", srcJarList)
	deps = append(deps, l.srcJars...)

	if resourcesList != nil {
		cmd.FlagWithInput("--resources ", resourcesList)
	}

	if l.classes != nil {
		deps = append(deps, l.classes)
		cmd.FlagWithArg("--classes ", l.classes.String())
	}

	cmd.FlagForEachArg("--classpath ", l.classpath.Strings())
	deps = append(deps, l.classpath...)

	cmd.FlagForEachArg("--extra_checks_jar ", l.extraLintCheckJars.Strings())
	deps = append(deps, l.extraLintCheckJars...)

	// The cache tag in project.xml is relative to the project.xml file.
	cmd.FlagWithArg("--cache_dir ", "cache")

	cmd.FlagWithInput("@",
		android.PathForSource(ctx, "build/soong/java/lint_defaults.txt"))

	cmd.FlagForEachArg("--disable_check ", l.disableChecks)
	cmd.FlagForEachArg("--warning_check ", l.warningChecks)
	cmd.FlagForEachArg("--error_check ", l.errorChecks)
	cmd.FlagForEachArg("--fatal_check ", l.fatalChecks)

	return projectXMLPath, configXMLPath, cacheDir, deps
}

func (l *linter) lint(ctx android.ModuleContext) {
	rule := android.NewRuleBuilder()

	projectXML, lintXML, cacheDir, deps := l.writeLintProjectXML(ctx, rule)

	l.outputs.html = android.PathForModuleOut(ctx, "lint-report.html")
	l.outputs.text = android.PathForModuleOut(ctx, "lint-report.txt")
	l.outputs.xml = android.PathForModuleOut(ctx, "lint-report.xml")

	rule.Command().Text("rm -rf").Flag(cacheDir.String())
	rule.Command().Text("mkdir -p").Flag(cacheDir.String())

	rule.Command().
		Text("(").
		Text("LINT_OPTS=-Xmx2048m").
		Tool(android.PathForSource(ctx, "prebuilts/cmdline-tools/tools/bin/lint")).
		Implicit(android.PathForSource(ctx, "prebuilts/cmdline-tools/tools/lib/lint-classpath.jar")).
		Flag("--quiet").
		FlagWithInput("--project ", projectXML).
		FlagWithInput("--config ", lintXML).
		FlagWithOutput("--html ", l.outputs.html).
		FlagWithOutput("--text ", l.outputs.text).
		FlagWithOutput("--xml ", l.outputs.xml).
		FlagWithArg("--compile-sdk-version ", l.compileSdkVersion).
		FlagWithArg("--sdk-home ", "prebuilts/sdk").                          // TODO: deps?
		FlagWithArg("--jdk-home ", ctx.Config().Getenv("ANDROID_JAVA_HOME")). // TODO: deps?
		FlagWithArg("--java-language-level ", l.javaLanguageLevel).
		FlagWithArg("--kotlin-language-level ", l.kotlinLanguageLevel).
		FlagWithArg("--url ", fmt.Sprintf(".=.,%s=out", android.PathForOutput(ctx).String())).
		Flag("--exitcode").
		Flags(l.flags).
		Implicits(deps).
		Text("|| (").Text("cat").Input(l.outputs.text).Text("; exit 7)").
		Text(")")

	rule.Command().Text("rm -rf").Flag(cacheDir.String())

	rule.Build(pctx, ctx, "lint", "lint")
}

func (l *linter) lintOutputs() *lintOutputs {
	return &l.outputs
}

type lintOutputIntf interface {
	lintOutputs() *lintOutputs
}

var _ lintOutputIntf = (*linter)(nil)

type lintSingleton struct {
	htmlZip android.WritablePath
	textZip android.WritablePath
	xmlZip  android.WritablePath
}

func (l *lintSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	var outputs []*lintOutputs
	var dirs []string
	ctx.VisitAllModules(func(m android.Module) {
		if l, ok := m.(lintOutputIntf); ok {
			outputs = append(outputs, l.lintOutputs())
		}
	})

	dirs = android.SortedUniqueStrings(dirs)

	zip := func(outputPath android.WritablePath, get func(*lintOutputs) android.Path) {
		var paths android.Paths

		for _, output := range outputs {
			paths = append(paths, get(output))
		}

		sort.Slice(paths, func(i, j int) bool {
			return paths[i].String() < paths[j].String()
		})

		rule := android.NewRuleBuilder()

		rule.Command().BuiltTool(ctx, "soong_zip").
			FlagWithOutput("-o ", outputPath).
			FlagWithArg("-C ", android.PathForIntermediates(ctx).String()).
			FlagWithRspFileInputList("-l ", paths)

		rule.Build(pctx, ctx, outputPath.Base(), outputPath.Base())
	}

	l.htmlZip = android.PathForOutput(ctx, "lint-report-html.zip")
	zip(l.htmlZip, func(l *lintOutputs) android.Path { return l.html })

	l.textZip = android.PathForOutput(ctx, "lint-report-text.zip")
	zip(l.textZip, func(l *lintOutputs) android.Path { return l.text })

	l.xmlZip = android.PathForOutput(ctx, "lint-report-xml.zip")
	zip(l.xmlZip, func(l *lintOutputs) android.Path { return l.xml })

	ctx.Phony("lint-check", l.htmlZip, l.textZip, l.xmlZip)
}

func (l *lintSingleton) MakeVars(ctx android.MakeVarsContext) {
	ctx.DistForGoal("lint-check", l.htmlZip, l.textZip, l.xmlZip)
}

var _ android.SingletonMakeVarsProvider = (*lintSingleton)(nil)

func init() {
	android.RegisterSingletonType("lint",
		func() android.Singleton { return &lintSingleton{} })
}
