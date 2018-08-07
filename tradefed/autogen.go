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
	"strings"

	"github.com/google/blueprint"

	"android/soong/android"
)

func getTestConfig(ctx android.ModuleContext, prop *string) android.Path {
	if p := ctx.ExpandOptionalSource(prop, "test_config"); p.Valid() {
		return p.Path()
	} else if p := android.ExistentPathForSource(ctx, ctx.ModuleDir(), "AndroidTest.xml"); p.Valid() {
		return p.Path()
	}
	return nil
}

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

var autogenHostJavaTest = pctx.StaticRule("autogenHostJavaTest", blueprint.RuleParams{
	Command:     "sed 's&{MODULE}&${name}&g' ${HostJavaTestConfigTemplate} > $out",
	CommandDeps: []string{"${JavaTestConfigTemplate}"},
}, "name")

func autogen(ctx android.ModuleContext, rule blueprint.Rule, prop *string, input android.Path) android.Path {
	if p := getTestConfig(ctx, prop); p != nil {
		return p
	} else if !strings.HasPrefix(ctx.ModuleDir(), "cts/") {
		outputFile := android.PathForModuleOut(ctx, ctx.ModuleName()+".config")

		ctx.Build(pctx, android.BuildParams{
			Rule:        rule,
			Description: "test config",
			Input:       input,
			Output:      outputFile,
			Args: map[string]string{
				"name": ctx.ModuleName(),
			},
		})

		return outputFile
	} else {
		// CTS modules can be used for test data, so test config files must be
		// explicitly created using AndroidTest.xml
		return nil
	}
}

func AutoGenNativeTestConfig(ctx android.ModuleContext, prop *string) android.Path {
	return autogen(ctx, autogenNativeTest, prop, nil)
}

func AutoGenNativeBenchmarkTestConfig(ctx android.ModuleContext, prop *string) android.Path {
	return autogen(ctx, autogenNativeBenchmark, prop, nil)
}

func AutoGenJavaTestConfig(ctx android.ModuleContext, prop *string) android.Path {
	if ctx.Device() {
		return autogen(ctx, autogenJavaTest, prop, nil)
	} else {
		return autogen(ctx, autogenHostJavaTest, prop, nil)
	}
}

var autogenInstrumentationTest = pctx.StaticRule("autogenInstrumentationTest", blueprint.RuleParams{
	Command: "${AutoGenTestConfigScript} $out $in ${EmptyTestConfig} ${InstrumentationTestConfigTemplate}",
	CommandDeps: []string{
		"${AutoGenTestConfigScript}",
		"${EmptyTestConfig}",
		"${InstrumentationTestConfigTemplate}",
	},
})

func AutoGenInstrumentationTestConfig(ctx android.ModuleContext, prop *string, manifest android.Path) android.Path {
	return autogen(ctx, autogenInstrumentationTest, prop, manifest)
}
