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
	"android/soong/android"
)

type linter struct {
	name                string
	manifest            android.Path
	mergedManifest      android.Path
	srcs                android.Paths
	srcJars             android.Paths
	resources           android.Paths
	classpath           classpath
	extraLintCheckJars  classpath
	test                bool
	library             bool
	compileSdkVersion   string
	javaLanguageLevel   string
	kotlinLanguageLevel string
	errorChecks         []string
	warningChecks       []string
	disableChecks       []string
}

func (l *linter) writeLintProjectXML(ctx android.ModuleContext) (android.WritablePath, android.WritablePath, android.Paths) {
	projectXMLPath := android.PathForModuleOut(ctx, "lint", "project.xml")
	// Lint looks for a lint.xml file next to the project.xml file, give it one.
	configXMLPath := android.PathForModuleOut(ctx, "lint", "lint.xml")
	cacheDir := android.PathForModuleOut(ctx, "lint", "cache")

	rule := android.NewRuleBuilder()
	var deps android.Paths

	rule.Command().Text("rm -rf").Flag(cacheDir.String())

	srcJarDir := android.PathForModuleOut(ctx, "lint-srcjars")
	srcJarList := zipSyncCmd(ctx, rule, srcJarDir, l.srcJars)

	cmd := rule.Command().
		BuiltTool(ctx, "lint-project-xml.sh").
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

	cmd.FlagForEachArg("--resources ", l.resources.Strings())
	deps = append(deps, l.resources...)

	cmd.FlagForEachArg("--classpath ", l.classpath.Strings())
	deps = append(deps, l.classpath...)

	cmd.FlagForEachArg("--extra_checks_jar ", l.extraLintCheckJars.Strings())
	deps = append(deps, l.extraLintCheckJars...)

	cmd.FlagWithArg("--cache_dir ", cacheDir.String())

	cmd.FlagForEachArg("--error_check ", l.errorChecks)
	cmd.FlagForEachArg("--warning_check ", l.warningChecks)

	cmd.FlagForEachArg("--disable_check ", l.disableChecks)

	rule.Command().Text("rm -rf").Flag(cacheDir.String())

	rule.Build(pctx, ctx, "generate_project_xml", "generate project.xml")

	return projectXMLPath, configXMLPath, deps
}

func (l *linter) lint(ctx android.ModuleContext) android.WritablePath {
	projectXML, lintXML, deps := l.writeLintProjectXML(ctx)

	outputHTML := android.PathForModuleOut(ctx, "lint-report.html")
	outputText := android.PathForModuleOut(ctx, "lint-report.txt")
	outputXML := android.PathForModuleOut(ctx, "lint-report.xml")

	rule := android.RuleBuilder{}
	rule.Command().
		Text("LINT_OPTS=-Xmx2048m").
		Tool(android.PathForSource(ctx, "prebuilts/cmdline-tools/tools/bin/lint")).
		Implicit(android.PathForSource(ctx, "prebuilts/cmdline-tools/tools/lib/lint-classpath.jar")).
		Flag("--quiet").
		FlagWithInput("--project ", projectXML).
		FlagWithInput("--config ", lintXML).
		FlagWithOutput("--html ", outputHTML).
		FlagWithOutput("--text ", outputText).
		FlagWithOutput("--xml ", outputXML).
		FlagWithArg("--compile-sdk-version ", l.compileSdkVersion).
		FlagWithArg("--sdk-home ", "prebuilts/sdk").                          // TODO: deps?
		FlagWithArg("--jdk-home ", ctx.Config().Getenv("ANDROID_JAVA_HOME")). // TODO: deps?
		FlagWithArg("--java-language-level ", l.javaLanguageLevel).
		FlagWithArg("--kotlin-language-level ", l.kotlinLanguageLevel).
		Implicits(deps)
	rule.Build(pctx, ctx, "lint", "lint")

	return outputXML
}
