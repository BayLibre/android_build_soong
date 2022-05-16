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
	"path/filepath"
	"strings"
)

func modulesOutputDirs(ctx ModuleContext, modules ...Module) []string {
	dirs := make(map[string]struct{})
	for _, module := range modules {
		paths, err := outputFilesForModule(ctx, module, "")
		if err != nil {
			continue
		}
		for _, path := range paths {
			if path != nil {
				dirs[filepath.Dir(path.String())] = struct{}{}
			}
		}
	}
	result := make([]string, 0, len(dirs))
	for dir := range dirs {
		result = append(result, dir)
	}
	return result
}

func modulesLicenseMetadata(ctx ModuleContext, modules ...Module) Paths {
	size := 0
	for _, module := range modules {
		b := module.base()
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
		b := module.base()
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
func buildNoticeOutputFromLicenseMetadata(ctx ModuleContext, tool, name string, outputFile WritablePath, productName, stripPrefix string, modules ...Module) {
	depsFile := outputFile.ReplaceExtension(ctx, strings.TrimPrefix(outputFile.Ext()+".d", "."))
	rule := NewRuleBuilder(pctx, ctx)
	if len(modules) == 0 {
		modules = []Module{ctx.Module()}
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
	outputs := modulesOutputDirs(ctx, modules...)
	if len(outputs) > 0 {
		cmd = cmd.FlagForEachArg("--strip_prefix ", outputs)
	}
	if productName != "" {
		cmd = cmd.FlagWithArg("--product ", productName)
	}
	cmd = cmd.Inputs(modulesLicenseMetadata(ctx, modules...))
	rule.Build(name, "container notice file")
}

// BuildNoticeTextOutputFromLicenseMetadata writes out a notice text file based
// on the license metadata files for the input `modules` defaulting to the
// current context module if none given.
func BuildNoticeTextOutputFromLicenseMetadata(ctx ModuleContext, outputFile WritablePath, productName, stripPrefix string, modules ...Module) {
	buildNoticeOutputFromLicenseMetadata(ctx, "textnotice", "text_notice", outputFile, productName, stripPrefix, modules...)
}

// BuildNoticeHtmlOutputFromLicenseMetadata writes out a notice text file based
// on the license metadata files for the input `modules` defaulting to the
// current context module if none given.
func BuildNoticeHtmlOutputFromLicenseMetadata(ctx ModuleContext, outputFile WritablePath, productName, stripPrefix string, modules ...Module) {
	buildNoticeOutputFromLicenseMetadata(ctx, "htmlnotice", "html_notice", outputFile, productName, stripPrefix, modules...)
}

// BuildNoticeXmlOutputFromLicenseMetadata writes out a notice text file based
// on the license metadata files for the input `modules` defaulting to the
// current context module if none given.
func BuildNoticeXmlOutputFromLicenseMetadata(ctx ModuleContext, outputFile WritablePath, productName, stripPrefix string, modules ...Module) {
	buildNoticeOutputFromLicenseMetadata(ctx, "xmlnotice", "xml_notice", outputFile, productName, stripPrefix, modules...)
}
