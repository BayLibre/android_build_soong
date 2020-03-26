// Copyright 2020 Arm Ltd. All rights reserved.
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
	"fmt"
)

type HardeningProperties struct {
	// Select the hardening level.  Possible values are "enabled",
	// "enforced", "disabled". Omit or leave blank, to select the default.
	Hardening *string `android:"arch_variant"`
	SelectedHardening string `blueprint:"mutated"`
}

type hardening struct {
	Properties HardeningProperties
}

func init() {
	// Update all of the dependencies of the enforced module to enforced.
	RegisterHardeningStuff(android.InitRegistrationContext)
}

func RegisterHardeningStuff(ctx android.RegistrationContext) {
	ctx.PostDepsMutators(func(ctx android.RegisterMutatorsContext) {
		ctx.TopDown("hardening_deps", hardeningDepsMutator())
	})
}

func hardeningDepsMutator() func(android.TopDownMutatorContext) {
	return func(mctx android.TopDownMutatorContext) {
		if m, ok := mctx.Module().(*Module); ok {
			if !m.isDependencyRoot() || m.GetHardening() != "enforced" {
				return
			}
			mctx.WalkDeps(func(module, parent android.Module) bool {
				if dep, ok := module.(*Module); ok  {
					if dep.GetHardening() == "disabled" {
						panic( fmt.Errorf("Hardening on module %q is disabled but requested by %q",
							dep.Name(), m.Name()))
					}
					dep.UpdateHardening("enforced")
				}
				return true
			})
		}
	}
}

func (hardening *hardening) props() []interface{} {
	return []interface{}{&hardening.Properties}
}

func (m *Module) GetHardening() string {
	return m.hardening.Properties.SelectedHardening
}

func (m *Module) UpdateHardening(SelectedHardening string) {
	m.hardening.Properties.SelectedHardening = SelectedHardening
}

func (hardening *hardening) begin(ctx BaseModuleContext) {
	hardening.Properties.SelectedHardening = func() string {
		s := ""
		if hardening.Properties.Hardening != nil {
			s = *hardening.Properties.Hardening
		}
		return s
	}()
}

func (hardening *hardening) flags(ctx ModuleContext, flags Flags) Flags {
	s := hardening.Properties.SelectedHardening
	if s == "" {
		s = ctx.Config().DefaultHardening()
	}

	switch s {
	case "enabled", "disabled", "enforced":
		// These are the default values.
	default:
		panic(fmt.Errorf("Unknown hardening: [%q]", s))
	}
	return flags
}
