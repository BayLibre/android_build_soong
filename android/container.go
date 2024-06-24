// Copyright 2024 Google Inc. All rights reserved.
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
	"reflect"
	"slices"
)

type InstallableModule interface {
	Module

	Base() *InstallableModuleBase
	Apexes() []string

	InSystemDomain() bool
	InVendorDomain() bool
	InApexes() bool
	InCts() bool

	SetSystemDomain(isSystem bool)
	SetVendorDomain(isVendor bool)
	SetApexes(apexNames []string)
	SetCts(isCts bool)

	WithinSameContainer(other InstallableModule) bool

	Domains() []string
}

type InstallableModuleBase struct {
	system    bool
	vendor    bool
	apexNames []string
	cts       bool
}

func (i *InstallableModuleBase) Base() *InstallableModuleBase {
	return i
}

func (i *InstallableModuleBase) Apexes() []string {
	return i.apexNames
}

func (i *InstallableModuleBase) InSystemDomain() bool {
	return i.system
}

func (i *InstallableModuleBase) InVendorDomain() bool {
	return i.vendor
}

func (i *InstallableModuleBase) InApexes() bool {
	return len(i.Apexes()) > 0
}

func (i *InstallableModuleBase) InCts() bool {
	return i.cts
}

func (i *InstallableModuleBase) SetVendorDomain(isVendor bool) {
	i.vendor = isVendor
}

func (i *InstallableModuleBase) SetSystemDomain(isSystem bool) {
	i.system = isSystem
}

func (i *InstallableModuleBase) SetApexes(apexNames []string) {
	i.apexNames = apexNames
}

func (i *InstallableModuleBase) SetCts(isCts bool) {
	i.cts = isCts
}

func (i *InstallableModuleBase) WithinSameContainer(other InstallableModule) bool {
	return (i.InSystemDomain() && other.InSystemDomain()) ||
		(i.InVendorDomain() && other.InVendorDomain()) ||
		(i.InCts() && other.InCts()) ||
		HasIntersection(i.Apexes(), other.Apexes())
}

func (i *InstallableModuleBase) Domains() []string {
	domains := []string{}
	if i.InSystemDomain() {
		domains = append(domains, "system")
	}
	if i.InVendorDomain() {
		domains = append(domains, "vendor")
	}
	if i.InCts() {
		domains = append(domains, "cts")
	}
	if i.InApexes() {
		domains = append(domains, i.apexNames...)
	}
	return domains
}

var prepareForTestWithContainer = FixtureRegisterWithContext(registerContainerMutator)

func registerContainerMutator(ctx RegistrationContext) {
	ctx.FinalDepsMutators(registerContainerFinalDepsMutator)
}

func registerContainerFinalDepsMutator(ctx RegisterMutatorsContext) {
	ctx.BottomUp("container_generation", containerGenerationMutator).Parallel()
	// ctx.BottomUp("container", containerMutator).Parallel()
}

func containerGenerationMutator(ctx BottomUpMutatorContext) {
	if m, ok := ctx.Module().(InstallableModule); ok {
		base := m.Base()

		if ctx.Module().InstallInProduct() {
			base.SetSystemDomain(true)
		}

		if ctx.Module().InstallInVendor() {
			base.SetVendorDomain(true)
		}

		props := ctx.Module().GetProperties()
		for _, prop := range props {
			val := reflect.ValueOf(prop).Elem()
			if val.Kind() == reflect.Struct {
				testSuites := val.FieldByName("Test_suites")
				if testSuites.IsValid() && testSuites.Kind() == reflect.Slice && slices.Contains(testSuites.Interface().([]string), "cts") {
					base.SetCts(true)
				}
			}
		}

		if apexInfo, ok := ModuleProvider(ctx, ApexInfoProvider); ok {
			base.SetApexes(apexInfo.InApexModules)
		}

		// fmt.Printf("%s: system: %t, vendor: %t, cts: %t, apex: %t\n", ctx.ModuleName(), base.InSystemDomain(), base.InVendorDomain(), base.InCts(), base.InApexes())
	}
}

func containerMutator(ctx BottomUpMutatorContext) {
	if m, ok := ctx.Module().(InstallableModule); ok {
		ctx.VisitDirectDepsIgnoreBlueprint(func(dep Module) {
			if other, ok := dep.(InstallableModule); ok {
				if !m.WithinSameContainer(other) {
					ctx.ModuleErrorf("Module %s belongs to %s domains, but depends on %s, which belongs to %s domains", ctx.ModuleName(), m.Domains(), dep.Name(), other.Domains())
				}
			}
		})
	}
}
