// Copyright 2019 Google Inc. All rights reserved.
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

func init() {
	android.RegisterSingletonType("hiddenapi", hiddenAPISingletonFactory)
}

type hiddenAPISingletonPathsStruct struct {
	stubFlags android.OutputPath
	flags     android.OutputPath
	metadata  android.OutputPath
}

var hiddenAPISingletonPathsKey = android.NewOnceKey("hiddenAPISingletonPathsKey")

func hiddenAPISingletonPaths(ctx android.PathContext) hiddenAPISingletonPathsStruct {
	return ctx.Config().Once(hiddenAPISingletonPathsKey, func() interface{} {
		return hiddenAPISingletonPathsStruct{
			stubFlags: android.PathForOutput(ctx, "hiddenapi", "hiddenapi-stub-flags.txt"),
			flags:     android.PathForOutput(ctx, "hiddenapi", "hiddenapi-flags.csv"),
			metadata:  android.PathForOutput(ctx, "hiddenapi", "hiddenapi-greylist.csv"),
		}
	}).(hiddenAPISingletonPathsStruct)
}

func hiddenAPISingletonFactory() android.Singleton {
	return hiddenAPISingleton{}
}

type hiddenAPISingleton struct{}

// hiddenAPI singleton rules
func (hiddenAPISingleton) GenerateBuildActions(ctx android.SingletonContext) {
	if ctx.Config().IsEnvTrue("UNSAFE_DISABLE_HIDDENAPI_FLAGS") {
		return
	}

	stubFlagsRule(ctx)

	if ctx.Config().FrameworksBaseDirExists(ctx) {
		flagsRule(ctx)
		metadataRule(ctx)
	}
}

func stubFlagsRule(ctx android.SingletonContext) {
	// Public API stubs
	publicStubModules := []string{
		"android_stubs_current",
		"android.test.base.stubs",
	}

	// System API stubs
	systemStubModules := []string{
		"android_system_stubs_current",
	}

	// Test API stubs
	testStubModules := []string{
		"android_test_stubs_current",
	}

	// Core Platform API stubs
	corePlatformStubModules := []string{
		"core.platform.api.stubs",
	}

	// Allow products to define their own stubs for custom product jars that apps can use.
	publicStubModules = append(publicStubModules, ctx.Config().ProductHiddenAPIStubs()...)
	systemStubModules = append(systemStubModules, ctx.Config().ProductHiddenAPIStubsSystem()...)
	testStubModules = append(testStubModules, ctx.Config().ProductHiddenAPIStubsTest()...)

	publicStubPaths := make(android.Paths, len(publicStubModules))
	systemStubPaths := make(android.Paths, len(systemStubModules))
	testStubPaths := make(android.Paths, len(testStubModules))
	corePlatformStubPaths := make(android.Paths, len(corePlatformStubModules))

	moduleListToPathList := map[*[]string]android.Paths{
		&publicStubModules:       publicStubPaths,
		&systemStubModules:       systemStubPaths,
		&testStubModules:         testStubPaths,
		&corePlatformStubModules: corePlatformStubPaths,
	}

	var bootDexJars android.Paths

	ctx.VisitAllModules(func(module android.Module) {
		if j, ok := module.(Dependency); ok {
			name := ctx.ModuleName(module)
			for moduleList, pathList := range moduleListToPathList {
				if i := android.IndexList(name, *moduleList); i != -1 {
					pathList[i] = j.DexJar()
				}
			}
		}

		if h, ok := module.(hiddenAPIIntf); ok {
			if jar := h.bootDexJar(); jar != nil {
				bootDexJars = append(bootDexJars, jar)
			}
		}
	})

	// Singleton rule which applies hiddenapi on all boot class path dex files.
	rule := android.NewRuleBuilder()

	outputPath := hiddenAPISingletonPaths(ctx).stubFlags
	tempPath := android.PathForOutput(ctx, outputPath.Rel()+".tmp")

	rule.Command().
		Tool(pctx.HostBinToolPath(ctx, "hiddenapi").String()).
		Text("list").
		FlagForEachInput("--boot-dex=", bootDexJars.Strings()).
		FlagWithInputList("--public-stub-classpath=", publicStubPaths.Strings(), ":").
		FlagWithInputList("--public-stub-classpath=", systemStubPaths.Strings(), ":").
		FlagWithInputList("--public-stub-classpath=", testStubPaths.Strings(), ":").
		FlagWithInputList("--core-platform-stub-classpath=", corePlatformStubPaths.Strings(), ":").
		FlagWithOutput("--out-api-flags=", tempPath.String())

	commitChangeForRestat(rule, tempPath, outputPath)

	rule.Build(pctx, ctx, "hiddenAPIStubFlagsFile", "hiddenapi stub flags")
}

func flagsRule(ctx android.SingletonContext) {
	var flagsCSV android.Paths

	var greylistIgnoreConflicts android.Path

	ctx.VisitAllModules(func(module android.Module) {
		if h, ok := module.(hiddenAPIIntf); ok {
			if csv := h.flagsCSV(); csv != nil {
				flagsCSV = append(flagsCSV, csv)
			}
		} else if ds, ok := module.(*Droidstubs); ok && ctx.ModuleName(module) == "hiddenapi-lists-docs" {
			greylistIgnoreConflicts = ds.removedDexApiFile
		}
	})

	if greylistIgnoreConflicts == nil {
		ctx.Errorf("failed to find removed_dex_api_filename from hiddenapi-lists-docs module")
		return
	}

	rule := android.NewRuleBuilder()

	outputPath := hiddenAPISingletonPaths(ctx).flags
	tempPath := android.PathForOutput(ctx, outputPath.Rel()+".tmp")

	stubFlags := hiddenAPISingletonPaths(ctx).stubFlags

	rule.Command().
		Tool(android.PathForSource(ctx, "frameworks/base/tools/hiddenapi/generate_hiddenapi_lists.py").String()).
		FlagWithInput("--csv ", stubFlags.String()).
		Inputs(flagsCSV.Strings()).
		FlagWithInput("--greylist ",
			android.PathForSource(ctx, "frameworks/base/config/hiddenapi-greylist.txt").String()).
		FlagWithInput("--greylist-ignore-conflicts ",
			greylistIgnoreConflicts.String()).
		FlagWithInput("--greylist-max-p ",
			android.PathForSource(ctx, "frameworks/base/config/hiddenapi-greylist-max-p.txt").String()).
		FlagWithInput("--greylist-max-o-ignore-conflicts ",
			android.PathForSource(ctx, "frameworks/base/config/hiddenapi-greylist-max-o.txt").String()).
		FlagWithInput("--blacklist ",
			android.PathForSource(ctx, "frameworks/base/config/hiddenapi-force-blacklist.txt").String()).
		FlagWithOutput("--output ", tempPath.String())

	commitChangeForRestat(rule, tempPath, outputPath)

	rule.Build(pctx, ctx, "hiddenAPIFlagsFile", "hiddenapi flags")
}

func metadataRule(ctx android.SingletonContext) {
	var metadataCSV android.Paths

	ctx.VisitAllModules(func(module android.Module) {
		if h, ok := module.(hiddenAPIIntf); ok {
			if csv := h.metadataCSV(); csv != nil {
				metadataCSV = append(metadataCSV, csv)
			}
		}
	})

	rule := android.NewRuleBuilder()

	outputPath := hiddenAPISingletonPaths(ctx).metadata

	rule.Command().
		Tool(android.PathForSource(ctx, "frameworks/base/tools/hiddenapi/merge_csv.py").String()).
		Inputs(metadataCSV.Strings()).
		Text(">").
		Output(outputPath.String())

	rule.Build(pctx, ctx, "hiddenAPIGreylistMetadataFile", "hiddenapi greylist metadata")
}

func commitChangeForRestat(rule *android.RuleBuilder, tempPath, outputPath android.WritablePath) {
	rule.Restat()
	rule.Temporary(tempPath.String())
	rule.Command().
		Text("(").
		Text("if").
		Text("cmp -s").Input(tempPath.String()).Output(outputPath.String()).Text(";").
		Text("then").
		Text("rm").Input(tempPath.String()).Text(";").
		Text("else").
		Text("mv").Input(tempPath.String()).Output(outputPath.String()).Text(";").
		Text("fi").
		Text(")")
}

func init() {
	android.RegisterMakeVarsProvider(pctx, hiddenAPIMakeVars)
}

func hiddenAPIMakeVars(ctx android.MakeVarsContext) {
	singletonPaths := ctx.Config().Get(hiddenAPISingletonPathsKey).(hiddenAPISingletonPathsStruct)
	ctx.Strict("INTERNAL_PLATFORM_HIDDENAPI_FLAGS", singletonPaths.flags.String())
	ctx.Strict("INTERNAL_PLATFORM_HIDDENAPI_GREYLIST_METADATA", singletonPaths.metadata.String())
}
