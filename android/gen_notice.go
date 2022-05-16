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

package android

import (
	"fmt"
	"strings"

	"github.com/google/blueprint/proptools"
)

func init() {
	RegisterGenNoticeBuildComponents(InitRegistrationContext)
}

// Register the gen_notice module type.
func RegisterGenNoticeBuildComponents(ctx RegistrationContext) {
	ctx.RegisterSingletonType("gen_notice_build_rules", GenNoticeBuildRulesFactory)
	ctx.RegisterModuleType("gen_notice", GenNoticeFactory)
}

type genNoticeBuildRules struct {
	notices map[*genNoticeModule]struct{}
}

var genNoticeBuildRulesSingleton = &genNoticeBuildRules{make(map[*genNoticeModule]struct{})}

func (s *genNoticeBuildRules) buildNoticeFile(ctx ModuleContext, m *genNoticeModule) {
	s.notices[m] = struct{}{}
}

func (s *genNoticeBuildRules) GenerateBuildActions(ctx SingletonContext) {
	for m := range s.notices {
		out := BuildNoticeTextOutputFromLicenseMetadata
		if strings.HasSuffix(m.properties.Out, ".xml") || strings.HasSuffix(m.properties.Out, ".xml.gz") {
			out = BuildNoticeXmlOutputFromLicenseMetadata
		} else if strings.HasSuffix(m.properties.Out, ".html") || strings.HasSuffix(m.properties.Out, ".html.gz") || strings.HasSuffix(m.properties.Out, ".htm") {
			out = BuildNoticeHtmlOutputFromLicenseMetadata
		}
		defaultProduct := ""
		if len(m.properties.For) > 0 {
			defaultProduct = m.properties.For[0]
		}

		mctx := ctx.ContextForModule(m)
		size := 0
		for _, name := range m.properties.For {
			mods := ctx.ModuleVariantsFromName(mctx, name)
			for _, mod := range mods {
				if mod == nil {
					continue
				}
				size++
			}
		}
		modules := make([]Module, 0, size)
		for _, name := range m.properties.For {
			mods := ctx.ModuleVariantsFromName(mctx, name)
			for _, mod := range mods {
				if mod == nil {
					continue
				}
				modules = append(modules, mod)
			}
		}
		out(mctx, m.output, proptools.StringDefault(m.properties.ProductName, defaultProduct), "", modules...)
	}
}

func GenNoticeBuildRulesFactory() Singleton {
	return genNoticeBuildRulesSingleton
}

type genNoticeProperties struct {
	// For specifies the modules for which to generate a notice file.
	For []string
	// ProductName specifies the product name to use for the notice file.
	ProductName *string
	// Out specifies the name of the output file.
	Out string `android:"arch_variant"`
	// Visibility specifies where this license can be used
	Visibility []string
}

type genNoticeModule struct {
	ModuleBase
	DefaultableModuleBase

	properties genNoticeProperties

	output     OutputPath
}

func (m *genNoticeModule) DepsMutator(ctx BottomUpMutatorContext) {
	// Verify the modules for which to generate notices exist.
	var missingModules []string
	for _, otherMod := range m.properties.For {
		if !ctx.OtherModuleExists(otherMod) {
			missingModules = append(missingModules, otherMod)
		}
	}
	if len(missingModules) == 1 {
		ctx.PropertyErrorf("for", "no %q module exists", missingModules[0])
	} else if len(missingModules) > 1 {
		ctx.PropertyErrorf("for", "modules \"%s\" do not exist", strings.Join(missingModules, "\", \""))
	}
}

func (m *genNoticeModule) GenerateAndroidBuildActions(ctx ModuleContext) {
	m.output = PathForModuleOut(ctx, m.properties.Out).OutputPath
	genNoticeBuildRulesSingleton.buildNoticeFile(ctx, m)
}

func GenNoticeFactory() Module {
	module := &genNoticeModule{}

	base := module.base()
	module.AddProperties(&base.nameProperties, &module.properties)

	// The visibility property needs to be checked and parsed by the visibility module.
	setPrimaryVisibilityProperty(module, "visibility", &module.properties.Visibility)

	initAndroidModuleBase(module)
	InitDefaultableModule(module)

	return module
}

var _ OutputFileProducer = (*genNoticeModule)(nil)

// Implements OutputFileProducer
func (m *genNoticeModule) OutputFiles(tag string) (Paths, error) {
	if tag == "" {
		return Paths{m.output}, nil
	}
	return nil, fmt.Errorf("unrecognized tag %q", tag)
}
