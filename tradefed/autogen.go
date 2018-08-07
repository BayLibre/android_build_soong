// Copyright 2018 Google Inc. All rights reserved.
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

package tradefed

import (
	"github.com/google/blueprint"

	"android/soong/android"
)

var autogenNativeTest = pctx.StaticRule("autogenNativeTest", blueprint.RuleParams{
	Command:     "sed 's&{MODULE}&${name}&g' ${NativeTestConfigTemplate} > $out",
	CommandDeps: []string{"${NativeTestConfigTemplate}"},
}, "name")

var autogenNativeBenchmark = pctx.StaticRule("autogenNativeBenchmark", blueprint.RuleParams{
	Command:     "sed 's&{MODULE}&${name}&g' ${NativeBenchmarkTestConfigTemplate} > $out",
	CommandDeps: []string{"${NativeBenchmarkTestConfigTemplate}"},
}, "name")

var autogenJavaTest = pctx.StaticRule("autogenJavaTest", blueprint.RuleParams{
	Command:     "sed 's&{MODULE}&${name}&g' ${JavaTestConfigTemplate} > $out",
	CommandDeps: []string{"${JavaTestConfigTemplate}"},
}, "name")

func autogen(ctx android.ModuleContext, rule blueprint.Rule) android.WritablePath {
	outputFile := android.PathForModuleOut(ctx, ctx.ModuleName()+".config")

	ctx.Build(pctx, android.BuildParams{
		Rule:        rule,
		Description: "test config",
		Output:      outputFile,
		Args: map[string]string{
			"name": ctx.ModuleName(),
		},
	})

	return outputFile
}

func AutoGenNativeTestConfig(ctx android.ModuleContext) android.WritablePath {
	return autogen(ctx, autogenNativeTest)
}

func AutoGenNativeBenchmarkTestConfig(ctx android.ModuleContext) android.WritablePath {
	return autogen(ctx, autogenNativeBenchmark)
}

func AutoGenJavaTestConfig(ctx android.ModuleContext) android.WritablePath {
	return autogen(ctx, autogenJavaTest)
}

var autogenInstrumentationTest = pctx.StaticRule("autogenInstrumentationTest", blueprint.RuleParams{
	Command: "${AutoGenTestConfigScript} $out $in ${EmptyTestConfig} ${InstrumentationTestConfigTemplate}",
	CommandDeps: []string{
		"${AutoGenTestConfigScript}",
		"${EmptyTestConfig}",
		"${InstrumentationTestConfigTemplate}",
	},
})

func AutoGenInstrumentationTestConfig(ctx android.ModuleContext, manifest android.Path) android.WritablePath {
	outputFile := android.PathForModuleOut(ctx, ctx.ModuleName()+".config")

	ctx.Build(pctx, android.BuildParams{
		Rule:        autogenInstrumentationTest,
		Description: "test config",
		Input:       manifest,
		Output:      outputFile,
	})

	return outputFile
}
