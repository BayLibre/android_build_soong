// Copyright 2017 Google Inc. All rights reserved.
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

package cc

import (
	"android/soong/android"
	"android/soong/genrule"

	"github.com/google/blueprint"
)

func init() {
	android.RegisterModuleType("cc_genrule", genRuleFactory)
}

type GenruleExtraProperties struct {
	Vendor_available   *bool
	Recovery_available *bool

	Shared_libs []string
	Static_libs []string

	// This genrule is for recovery variant
	InRecovery bool `blueprint:"mutated"`
}

type GenruleModule struct {
	*genrule.Module

	Properties GenruleExtraProperties
}

func (g *GenruleModule) DepsMutator(ctx android.BottomUpMutatorContext) {
	g.Module.DepsMutator(ctx)

	for _, lib := range g.Properties.Shared_libs {
		ctx.AddFarVariationDependencies([]blueprint.Variation{
			{Mutator: "link", Variation: "shared"},
		}, &genrule.ExtraDependencyTag{
			Label: lib,
		}, lib)
	}
	for _, lib := range g.Properties.Static_libs {
		// TODO: add tag
		ctx.AddFarVariationDependencies([]blueprint.Variation{
			{Mutator: "link", Variation: "static"},
		}, &genrule.ExtraDependencyTag{
			Label: lib,
		}, lib)
	}
}

// cc_genrule is a genrule that can depend on other cc_* objects.
// The cmd may be run multiple times, once for each of the different arch/etc
// variations.
func genRuleFactory() android.Module {
	module := &GenruleModule{
		Module: genrule.NewGenRule(),
	}

	module.AddProperties(&module.Properties)

	android.InitAndroidArchModule(module, android.HostAndDeviceSupported, android.MultilibBoth)

	android.InitApexModule(module)

	return module
}
