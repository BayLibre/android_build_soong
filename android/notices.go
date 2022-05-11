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

package android

import (
	"strings"

	"github.com/google/blueprint"
)

func modulesOutputs(ctx ModuleContext, modules ...blueprint.Module) Paths {
	result := make(Paths, 0, len(modules))
	for _, module := range modules {
		paths, err := outputFilesForModule(ctx, module, "")
		if err != nil {
			continue
		}
		for _, path := range paths {
			if path != nil {
				result = append(result, path)
			}
		}
	}
	return result
}

func modulesLicenseMetadata(ctx ModuleContext, modules ...blueprint.Module) Paths {
	size := 0
	for _, module := range modules {
		m, ok := module.(Module)
		if !ok {
			continue
		}
		b := m.base()
		if b == nil {
			continue
		}
		if b.licenseMetadataFile == nil {
			continue
		}
		size++
	}
	result := make(Paths, 0, size)
	for _, module := range modules {
		m, ok := module.(Module)
		if !ok {
			continue
		}
		b := m.base()
		if b == nil {
			continue
		}
		if b.licenseMetadataFile == nil {
			continue
		}
		result = append(result, b.licenseMetadataFile)
	}
	return result
}

// buildNoticeOutputFromLicenseMetadata writes out a notice file.
func buildNoticeOutputFromLicenseMetadata(ctx ModuleContext, tool, name string, outputFile WritablePath, productName, stripPrefix string, modules ...blueprint.Module) {
	depsFile := outputFile.ReplaceExtension(ctx, strings.TrimPrefix(outputFile.Ext()+".d", "."))
	rule := NewRuleBuilder(pctx, ctx)
	if len(modules) == 0 {
		modules = []blueprint.Module{ctx.Module()}
	}
	if productName == "" {
		productName = modules[0].Name()
	}
	cmd := rule.Command().
		BuiltTool(tool).
		FlagWithOutput("-o ", outputFile).
		FlagWithDepFile("-d ", depsFile)
	if stripPrefix != "" {
		cmd = cmd.FlagWithArg("--strip_prefix ", stripPrefix)
	}
	outputs := modulesOutputs(ctx, modules...).Strings()
	if len(outputs) > 0 {
		cmd = cmd.FlagForEachArg("--strip_prefix ", modulesOutputs(ctx, modules...).Strings())
	}
	if productName != "" {
		cmd = cmd.FlagWithArg("--product ", productName)
	}
	cmd.Inputs(modulesLicenseMetadata(ctx, modules...))
	rule.Build(name, "container notice file")
}

// BuildNoticeTextOutputFromLicenseMetadata writes out a notice text file based on the input
// license metadata files.
func BuildNoticeTextOutputFromLicenseMetadata(ctx ModuleContext, outputFile WritablePath, productName, stripPrefix string, modules ...blueprint.Module) {
	buildNoticeOutputFromLicenseMetadata(ctx, "textnotice", "text_notice", outputFile, productName, stripPrefix, modules...)
}

// BuildNoticeHtmlOutputFromLicenseMetadata writes out a notice html file based on the input
// license metadata files.
func BuildNoticeHtmlOutputFromLicenseMetadata(ctx ModuleContext, outputFile WritablePath, productName, stripPrefix string, modules ...blueprint.Module) {
	buildNoticeOutputFromLicenseMetadata(ctx, "htmlnotice", "html_notice", outputFile, productName, stripPrefix, modules...)
}

// BuildNoticeXmlOutputFromLicenseMetadata writes out a notice xml file based on the input
// license metadata files.
func BuildNoticeXmlOutputFromLicenseMetadata(ctx ModuleContext, outputFile WritablePath, productName, stripPrefix string, modules ...blueprint.Module) {
	buildNoticeOutputFromLicenseMetadata(ctx, "xmlnotice", "xml_notice", outputFile, productName, stripPrefix, modules...)
}
