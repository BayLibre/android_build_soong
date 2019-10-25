// Copyright 2017 Google Inc. All rights reserved.
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
	"io"
	"path/filepath"
	"strings"

	"android/soong/java/config"

	"android/soong/android"
)

// OpenJDK 9 introduces the concept of "system modules", which replace the bootclasspath.  This
// file will produce the rules necessary to convert each unique set of bootclasspath jars into
// system modules in a runtime image using the jmod and jlink tools.

func init() {
	android.RegisterModuleType("java_system_modules", SystemModulesFactory)
}

func TransformJarsToSystemModules(ctx android.ModuleContext, jars android.Paths) (android.Path, android.Paths) {
	outDir := android.PathForModuleOut(ctx, "system")
	workDir := android.PathForModuleOut(ctx, "modules")
	outputFile := android.PathForModuleOut(ctx, "system/lib/modules")
	jrtFsJar := android.PathForModuleOut(ctx, "system/lib/jrt-fs.jar")
	releaseFile := android.PathForModuleOut(ctx, "system/release")

	rule := android.NewRuleBuilder()

	rule.Command().Text("rm -rf").Text(outDir.String()).Text(workDir.String())

	rule.Command().Text("mkdir -p").Text(filepath.Join(workDir.String(), "jmod"))

	// Generate module-info.java into workDir
	moduleInfoJava := workDir.Join(ctx, "module-info.java")
	rule.Command().Tool(android.PathForSource(ctx, "build/soong/scripts/jars-to-module-info-java.sh")).
		Text("java.base").
		Inputs(jars).
		Text(">").
		Output(moduleInfoJava)
	rule.Temporary(moduleInfoJava)

	// Compile module-info.java into module-info.class
	moduleInfoClass := moduleInfoJava.ReplaceExtension(ctx, "class")
	rule.Command().Tool(config.JavacCmd(ctx)).
		FlagWithArg("--system=", "none").
		FlagWithInputList("--patch-module=java.base=", jars, ":").
		Input(moduleInfoJava).
		ImplicitOutput(moduleInfoClass)
	rule.Temporary(moduleInfoClass)

	// Jar module-info.class into classes.jar
	classesJar := workDir.Join(ctx, "classes.jar")
	rule.Command().BuiltTool(ctx, "soong_zip").
		Flag("-jar").
		FlagWithOutput("-o ", classesJar).
		FlagWithArg("-C ", workDir.String()).
		FlagWithInput("-f ", moduleInfoClass)
	rule.Temporary(classesJar)

	// Combine classes.jar and input jars into module.jar
	moduleJar := workDir.Join(ctx, "module.jar")
	rule.Command().BuiltTool(ctx, "merge_zips").
		Flag("-j").
		Output(moduleJar).
		Input(classesJar).
		Inputs(jars)
	rule.Temporary(moduleJar)

	moduleVersion := "9"
	if ctx.Config().IsEnvTrue("EXPERIMENTAL_USE_OPENJDK11_TOOLCHAIN") {
		moduleVersion = "11"
	}

	// Create java.base.jmod from module.jar
	// Note: The version of the java.base module created must match the version
	// of the jlink tool which consumes it.
	javaBaseJmod := workDir.Join(ctx, "jmod/java.base.jmod")
	rule.Command().Tool(config.JmodCmd(ctx)).
		Text("create").
		FlagWithArg("--module-version ", moduleVersion).
		FlagWithArg("--target-platform ", "android").
		FlagWithInput("--class-path ", moduleJar).
		Output(javaBaseJmod)
	rule.Temporary(javaBaseJmod)

	rule.Command().Tool(config.JlinkCmd(ctx)).
		FlagWithArg("--module-path ", workDir.Join(ctx, "jmod").String()).Implicit(javaBaseJmod).
		FlagWithArg("--add-modules ", "java.base").
		FlagWithArg("--output ", outDir.String()).
		ImplicitOutput(outputFile).
		ImplicitOutput(releaseFile).
		// Note: The system-modules jlink plugin is disabled because (a) it is not
		// useful on Android, and (b) it causes errors with later versions of jlink
		// when the jdk.internal.module is absent from java.base (as it is here).
		FlagWithArg("--disable-plugin ", "system-modules")

	// Copy jrt-fs.jar into the system modules directory
	rule.Command().Text("cp").Input(config.JrtFsJar(ctx)).Output(jrtFsJar)

	rule.Build(pctx, ctx, "system_modules", "system modules")

	return outDir, android.Paths{
		outputFile,
		jrtFsJar,
		releaseFile,
	}
}

func SystemModulesFactory() android.Module {
	module := &SystemModules{}
	module.AddProperties(&module.properties)
	android.InitAndroidArchModule(module, android.HostAndDeviceSupported, android.MultilibCommon)
	android.InitDefaultableModule(module)
	return module
}

type SystemModules struct {
	android.ModuleBase
	android.DefaultableModuleBase

	properties SystemModulesProperties

	// The aggregated header jars from all jars specified in the libs property.
	// Used when system module is added as a dependency to bootclasspath.
	headerJars android.Paths
	outputDir  android.Path
	outputDeps android.Paths
}

type SystemModulesProperties struct {
	// List of java library modules that should be included in the system modules
	Libs []string
}

func (system *SystemModules) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	var jars android.Paths

	ctx.VisitDirectDepsWithTag(libTag, func(module android.Module) {
		dep, _ := module.(Dependency)
		jars = append(jars, dep.HeaderJars()...)
	})

	system.headerJars = jars

	system.outputDir, system.outputDeps = TransformJarsToSystemModules(ctx, jars)
}

func (system *SystemModules) DepsMutator(ctx android.BottomUpMutatorContext) {
	ctx.AddVariationDependencies(nil, libTag, system.properties.Libs...)
}

func (system *SystemModules) AndroidMk() android.AndroidMkData {
	return android.AndroidMkData{
		Custom: func(w io.Writer, name, prefix, moduleDir string, data android.AndroidMkData) {
			fmt.Fprintln(w)

			makevar := "SOONG_SYSTEM_MODULES_" + name
			fmt.Fprintln(w, makevar, ":=$=", system.outputDir.String())
			fmt.Fprintln(w)

			makevar = "SOONG_SYSTEM_MODULES_LIBS_" + name
			fmt.Fprintln(w, makevar, ":=$=", strings.Join(system.properties.Libs, " "))
			fmt.Fprintln(w)

			makevar = "SOONG_SYSTEM_MODULES_DEPS_" + name
			fmt.Fprintln(w, makevar, ":=$=", strings.Join(system.outputDeps.Strings(), " "))
			fmt.Fprintln(w)

			fmt.Fprintln(w, name+":", "$("+makevar+")")
			fmt.Fprintln(w, ".PHONY:", name)
		},
	}
}
