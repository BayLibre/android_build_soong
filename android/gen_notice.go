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

type genNoticeBuildRules struct{}

func (s *genNoticeBuildRules) GenerateBuildActions(ctx SingletonContext) {
	ctx.VisitAllModules(func(m Module) {
		gm, ok := m.(*genNoticeModule)
		if !ok {
			return
		}
		if len(gm.missing) > 0 {
			ctx.missingReferencesRule("notice", "notice for container", m, gm.output, gm.missing...)
			return
		}
		out := BuildNoticeTextOutputFromLicenseMetadata
		suffix := gm.getSuffix()
		if strings.HasSuffix(suffix, ".xml") || strings.HasSuffix(suffix, ".xml.gz") {
			out = BuildNoticeXmlOutputFromLicenseMetadata
		} else if strings.HasSuffix(suffix, ".html") || strings.HasSuffix(suffix, ".html.gz") || strings.HasSuffix(suffix, ".htm") {
			out = BuildNoticeHtmlOutputFromLicenseMetadata
		}
		defaultLibrary := ""
		if len(gm.properties.For) > 0 {
			defaultLibrary = gm.properties.For[0]
		}

		modules := make([]Module, 0)
		for _, name := range gm.properties.For {
			mods := ctx.ModuleVariantsFromName(gm, name)
			for _, mod := range mods {
				if mod == nil {
					continue
				}
				modules = append(modules, mod)
			}
		}
		if ctx.Failed() {
			return
		}
		out(ctx, gm.output, proptools.StringDefault(gm.properties.LibraryName, defaultLibrary), "", modules...)
	})
}

func GenNoticeBuildRulesFactory() Singleton {
	return &genNoticeBuildRules{}
}

type genNoticeProperties struct {
	// For specifies the modules for which to generate a notice file.
	For []string
	// LibraryName specifies the internal name to use for the notice file.
	LibraryName *string
	// Stem specifies the base name of the output file.
	Stem *string `android:"arch_variant"`
	// Suffix specifies the file extension to use.
	Suffix *string
	// Visibility specifies where this license can be used
	Visibility []string
}

type genNoticeModule struct {
	ModuleBase
	DefaultableModuleBase

	properties genNoticeProperties

	output  OutputPath
	missing []string
}

func (m *genNoticeModule) DepsMutator(ctx BottomUpMutatorContext) {
	if !ctx.Config().AllowMissingDependencies() {
		var missing []string
		// Verify the modules for which to generate notices exist.
		for _, otherMod := range m.properties.For {
			if !ctx.OtherModuleExists(otherMod) {
				missing = append(missing, otherMod)
			}
		}
		if len(missing) == 1 {
			ctx.PropertyErrorf("for", "no %q module exists", missing[0])
		} else if len(missing) > 1 {
			ctx.PropertyErrorf("for", "modules \"%s\" do not exist", strings.Join(missing, "\", \""))
		}
	}
}

func (m *genNoticeModule) getStem() string {
	stem := m.base().BaseModuleName()
	if m.properties.Stem != nil {
		stem = proptools.String(m.properties.Stem)
	}
	return stem
}

func (m *genNoticeModule) getSuffix() string {
	return proptools.StringDefault(m.properties.Suffix, "")
}

func (m *genNoticeModule) GenerateAndroidBuildActions(ctx ModuleContext) {
	if ctx.Config().AllowMissingDependencies() {
		// Verify the modules for which to generate notices exist.
		for _, otherMod := range m.properties.For {
			if !ctx.OtherModuleExists(otherMod) {
				m.missing = append(m.missing, otherMod)
			}
		}
		m.missing = append(m.missing, ctx.GetMissingDependencies()...)
		m.missing = FirstUniqueStrings(m.missing)
	}
	out := m.getStem() + m.getSuffix()
	m.output = PathForModuleOut(ctx, out).OutputPath
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
