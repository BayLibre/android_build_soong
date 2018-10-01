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
	"fmt"
)

func getTestConfigTemplate(ctx android.ModuleContext, prop *string) android.OptionalPath {
	return ctx.ExpandOptionalSource(prop, "test_config_template")
}

func getTestConfig(ctx android.ModuleContext, prop *string) android.Path {
	if p := ctx.ExpandOptionalSource(prop, "test_config"); p.Valid() {
		return p.Path()
	} else if p := android.ExistentPathForSource(ctx, ctx.ModuleDir(), "AndroidTest.xml"); p.Valid() {
		return p.Path()
	}
	return nil
}

var autogenTestConfig = pctx.StaticRule("autogenTestConfig", blueprint.RuleParams{
	Command:     "sed 's&{MODULE}&${name}&g' $template > $out",
	CommandDeps: []string{"$template"},
}, "name", "template")

var autogenTestConfigWithUID = pctx.StaticRule("autogenTestConfigWithUID", blueprint.RuleParams{
	Command: "sed 's&{MODULE}&${name}&g' $template > $out ;" +
		"${uidCmd} $out",
	CommandDeps: []string{"$template"},
}, "name", "template", "uidCmd")

func testConfigPath(ctx android.ModuleContext, prop *string) (path android.Path, autogenPath android.WritablePath) {
	if p := getTestConfig(ctx, prop); p != nil {
		return p, nil
	} else if !strings.HasPrefix(ctx.ModuleDir(), "cts/") {
		outputFile := android.PathForModuleOut(ctx, ctx.ModuleName()+".config")
		return nil, outputFile
	} else {
		// CTS modules can be used for test data, so test config files must be
		// explicitly created using AndroidTest.xml
		// TODO(b/112602712): remove the path check
		return nil, nil
	}
}

func autogenTemplate(ctx android.ModuleContext, output android.WritablePath, template string) {
	ctx.Build(pctx, android.BuildParams{
		Rule:        autogenTestConfig,
		Description: "test config",
		Output:      output,
		Args: map[string]string{
			"name":     ctx.ModuleName(),
			"template": template,
		},
	})
}

func autogenNativeConfigTemplate(ctx android.ModuleContext, output android.WritablePath, template string, runTestAsProp *string) {
	//If no runTestAsProp then delete {UID_OPTION} line, else replace it to corresponding run-test-as option format.
	uidCmd := "sed -i '/{UID_OPTION}/d'"
	if runTestAsProp != nil {
		uidOption := fmt.Sprintf("<option name=\"run-test-as\" value=\"%s\" />", *runTestAsProp)
		uidCmd = fmt.Sprintf("sed -i 's&{UID_OPTION}&%s&g'", uidOption)
	}
	ctx.Build(pctx, android.BuildParams{
		Rule:        autogenTestConfigWithUID,
		Description: "test config",
		Output:      output,
		Args: map[string]string{
			"name":     ctx.ModuleName(),
			"template": template,
			"uidCmd":   uidCmd,
		},
	})
}

func AutoGenNativeTestConfig(ctx android.ModuleContext, testConfigProp *string,
	testConfigTemplateProp *string, runTestAsProp *string) android.Path {
	path, autogenPath := testConfigPath(ctx, testConfigProp)
	if autogenPath != nil {
		templatePath := getTestConfigTemplate(ctx, testConfigTemplateProp)
		if templatePath.Valid() {
			autogenNativeConfigTemplate(ctx, autogenPath, templatePath.String(), runTestAsProp)
		} else {
			if ctx.Device() {
				autogenNativeConfigTemplate(ctx, autogenPath, "${NativeTestConfigTemplate}", runTestAsProp)
			} else {
				autogenNativeConfigTemplate(ctx, autogenPath, "${NativeHostTestConfigTemplate}", runTestAsProp)
			}
		}
		return autogenPath
	}
	return path
}

func AutoGenNativeBenchmarkTestConfig(ctx android.ModuleContext, testConfigProp *string,
	testConfigTemplateProp *string) android.Path {
	path, autogenPath := testConfigPath(ctx, testConfigProp)
	if autogenPath != nil {
		templatePath := getTestConfigTemplate(ctx, testConfigTemplateProp)
		if templatePath.Valid() {
			autogenTemplate(ctx, autogenPath, templatePath.String())
		} else {
			autogenTemplate(ctx, autogenPath, "${NativeBenchmarkTestConfigTemplate}")
		}
		return autogenPath
	}
	return path
}

func AutoGenJavaTestConfig(ctx android.ModuleContext, testConfigProp *string, testConfigTemplateProp *string) android.Path {
	path, autogenPath := testConfigPath(ctx, testConfigProp)
	if autogenPath != nil {
		templatePath := getTestConfigTemplate(ctx, testConfigTemplateProp)
		if templatePath.Valid() {
			autogenTemplate(ctx, autogenPath, templatePath.String())
		} else {
			if ctx.Device() {
				autogenTemplate(ctx, autogenPath, "${JavaTestConfigTemplate}")
			} else {
				autogenTemplate(ctx, autogenPath, "${JavaHostTestConfigTemplate}")
			}
		}
		return autogenPath
	}
	return path
}

var autogenInstrumentationTest = pctx.StaticRule("autogenInstrumentationTest", blueprint.RuleParams{
	Command: "${AutoGenTestConfigScript} $out $in ${EmptyTestConfig} $template",
	CommandDeps: []string{
		"${AutoGenTestConfigScript}",
		"${EmptyTestConfig}",
		"$template",
	},
}, "name", "template")

func AutoGenInstrumentationTestConfig(ctx android.ModuleContext, testConfigProp *string, testConfigTemplateProp *string, manifest android.Path) android.Path {
	path, autogenPath := testConfigPath(ctx, testConfigProp)
	if autogenPath != nil {
		template := "${InstrumentationTestConfigTemplate}"
		moduleTemplate := getTestConfigTemplate(ctx, testConfigTemplateProp)
		if moduleTemplate.Valid() {
			template = moduleTemplate.String()
		}
		ctx.Build(pctx, android.BuildParams{
			Rule:        autogenInstrumentationTest,
			Description: "test config",
			Input:       manifest,
			Output:      autogenPath,
			Args: map[string]string{
				"name":     ctx.ModuleName(),
				"template": template,
			},
		})
		return autogenPath
	}
	return path
}
