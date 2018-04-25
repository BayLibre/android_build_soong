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

package xml

import (
	"android/soong/android"

	"github.com/google/blueprint"
)

// prebuilt_etc_xml installs an xml file under <partition>/etc/<subdir>.
// It also optionally validates the xml file against the schema.

var (
	pctx = android.NewPackageContext("android/soong/xml")

	xmllint = pctx.AndroidStaticRule("xmllint",
		blueprint.RuleParams{
			Command:     `rm -rf $out && $XmlLintCmd --dtdvalid $dtd $in > /dev/null && touch $out`,
			CommandDeps: []string{"$XmlLintCmd"},
		},
		"dtd")
)

func init() {
	android.RegisterModuleType("prebuilt_etc_xml", PrebuiltEtcXmlFactory)
	pctx.HostBinToolVariable("XmlLintCmd", "xmllint")
}

type prebuiltEtcXmlProperties struct {
	// Optional DTD that will be used to validate the xml file.
	Schema string
}

type prebuiltEtcXml struct {
	android.PrebuiltEtc

	properties prebuiltEtcXmlProperties
}

func (p *prebuiltEtcXml) timestampFilePath(ctx android.ModuleContext) android.WritablePath {
	return android.PathForModuleOut(ctx, p.PrebuiltEtc.SourceFilePath(ctx).Base()+"-timestamp")
}

func (p *prebuiltEtcXml) DepsMutator(ctx android.BottomUpMutatorContext) {
	p.PrebuiltEtc.DepsMutator(ctx)

	// To support ":modulename" in schema
	android.ExtractSourceDeps(ctx, &p.properties.Schema)
}

func (p *prebuiltEtcXml) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	p.PrebuiltEtc.GenerateAndroidBuildActions(ctx)

	if p.properties.Schema != "" {
		p.SetAdditionalDependencies([]android.Path{p.timestampFilePath(ctx)})

		schema := ctx.ExpandSource(p.properties.Schema, "schema")

		ctx.Build(pctx, android.BuildParams{
			Rule:        xmllint,
			Description: "xmllint",
			Input:       p.PrebuiltEtc.SourceFilePath(ctx),
			Output:      p.timestampFilePath(ctx),
			Args: map[string]string{
				"dtd": schema.String(),
			},
		})
	}
}

func (p *prebuiltEtcXml) AndroidMk() android.AndroidMkData {
	return p.PrebuiltEtc.AndroidMk()
}

func PrebuiltEtcXmlFactory() android.Module {
	module := &prebuiltEtcXml{}
	module.AddProperties(&module.properties)

	android.InitPrebuiltEtcModule(&module.PrebuiltEtc)
	// This module is device-only
	android.InitAndroidArchModule(module, android.DeviceSupported, android.MultilibCommon)
	return module
}
