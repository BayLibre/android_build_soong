// Copyright 2023 Google Inc. All rights reserved.
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

package build_flags

import (
	"fmt"
	"strings"

	"android/soong/android"

	"github.com/google/blueprint"
)

type DeclarationsModule struct {
	android.ModuleBase
	android.DefaultableModuleBase

	// Properties for "aconfig_declarations"
	properties struct {
		// aconfig files, relative to this Android.bp file
		Srcs []string `android:"path"`
	}

	intermediatePath android.WritablePath
}

func DeclarationsFactory() android.Module {
	module := &DeclarationsModule{}

	android.InitAndroidModule(module)
	android.InitDefaultableModule(module)
	module.AddProperties(&module.properties)

	return module
}

func (module *DeclarationsModule) OutputFiles(tag string) (android.Paths, error) {
	switch tag {
	case "":
		// The default output of this module is the intermediates format, which is
		// not installable and in a private format that no other rules can handle
		// correctly.
		return []android.Path{module.intermediatePath}, nil
	default:
		return nil, fmt.Errorf("unsupported build_flags_declarations module reference tag %q", tag)
	}
}

func joinAndPrefix(prefix string, values []string) string {
	var sb strings.Builder
	for _, v := range values {
		sb.WriteString(prefix)
		sb.WriteString(v)
	}
	return sb.String()
}

func optionalVariable(prefix string, value string) string {
	var sb strings.Builder
	if value != "" {
		sb.WriteString(prefix)
		sb.WriteString(value)
	}
	return sb.String()
}

func (module *DeclarationsModule) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	// Get the values that came from the global RELEASE_ACONFIG_VALUE_SETS flag
	valuesFiles := make([]android.Path, 0)

	// Intermediate format
	declarationFiles := android.PathsForModuleSrc(ctx, module.properties.Srcs)
	intermediateCacheFilePath := android.PathForModuleOut(ctx, "build_flag_intermediate.pb")
	inputFiles := make([]android.Path, len(declarationFiles))
	copy(inputFiles, declarationFiles)

	// TODO(lamont): generate the rc_proto.FlagArtifacts message for the sources.
	args := map[string]string{
		"release_version":    ctx.Config().ReleaseVersion(),
		"package":            module.properties.Package,
		"declarations":       android.JoinPathsWithPrefix(declarationFiles, "--declarations "),
		"values":             joinAndPrefix(" --values ", module.properties.Values),
		"default-permission": optionalVariable(" --default-permission ", defaultPermission),
	}
	if len(module.properties.Container) > 0 {
		args["container"] = "--container " + module.properties.Container
	}
	ctx.Build(pctx, android.BuildParams{
		Rule:        aconfigRule,
		Output:      intermediateCacheFilePath,
		Inputs:      inputFiles,
		Description: "aconfig_declarations",
		Args:        args,
	})

	intermediateDumpFilePath := android.PathForModuleOut(ctx, "intermediate.txt")
	ctx.Build(pctx, android.BuildParams{
		Rule:        aconfigTextRule,
		Output:      intermediateDumpFilePath,
		Inputs:      android.Paths{intermediateCacheFilePath},
		Description: "aconfig_text",
	})

	android.SetProvider(ctx, android.AconfigDeclarationsProviderKey, android.AconfigDeclarationsProviderData{
		Package:                     module.properties.Package,
		Container:                   module.properties.Container,
		Exportable:                  module.properties.Exportable,
		IntermediateCacheOutputPath: intermediateCacheFilePath,
		IntermediateDumpOutputPath:  intermediateDumpFilePath,
	})

}
