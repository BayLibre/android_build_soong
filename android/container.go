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

	InSystemContainer() bool
	InProductContainer() bool
	InVendorContainer() bool
	InApexes() bool
	InCts() bool

	SetSystemContainer(isSystem bool)
	SetProductContainer(isProduct bool)
	SetVendorContainer(isVendor bool)
	SetApexes(apexNames []string)
	SetCts(isCts bool)

	WithinSameContainer(other InstallableModule) bool

	Containers() []string
}

type InstallableModuleBase struct {
	system    bool
	product   bool
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

func (i *InstallableModuleBase) InSystemContainer() bool {
	return i.system
}

func (i *InstallableModuleBase) InProductContainer() bool {
	return i.product
}

func (i *InstallableModuleBase) InVendorContainer() bool {
	return i.vendor
}

func (i *InstallableModuleBase) InApexes() bool {
	return len(i.Apexes()) > 0
}

func (i *InstallableModuleBase) InCts() bool {
	return i.cts
}

func (i *InstallableModuleBase) SetSystemContainer(isSystem bool) {
	i.system = isSystem
}

func (i *InstallableModuleBase) SetProductContainer(isProduct bool) {
	i.product = isProduct
}

func (i *InstallableModuleBase) SetVendorContainer(isVendor bool) {
	i.vendor = isVendor
}

func (i *InstallableModuleBase) SetApexes(apexNames []string) {
	i.apexNames = apexNames
}

func (i *InstallableModuleBase) SetCts(isCts bool) {
	i.cts = isCts
}

func (i *InstallableModuleBase) WithinSameContainer(other InstallableModule) bool {
	return (i.InSystemContainer() && other.InSystemContainer()) ||
		(i.InProductContainer() && other.InProductContainer()) ||
		(i.InVendorContainer() && other.InVendorContainer()) ||
		(i.InCts() && other.InCts()) ||
		HasIntersection(i.Apexes(), other.Apexes())
}

func (i *InstallableModuleBase) Containers() []string {
	containers := []string{}
	if i.InSystemContainer() {
		containers = append(containers, "system")
	}
	if i.InProductContainer() {
		containers = append(containers, "product")
	}
	if i.InVendorContainer() {
		containers = append(containers, "vendor")
	}
	if i.InCts() {
		containers = append(containers, "cts")
	}
	if i.InApexes() {
		containers = append(containers, i.apexNames...)
	}
	return containers
}

var prepareForTestWithContainer = FixtureRegisterWithContext(registerContainerMutator)

func registerContainerMutator(ctx RegistrationContext) {
	ctx.FinalDepsMutators(registerContainerFinalDepsMutator)
}

func registerContainerFinalDepsMutator(ctx RegisterMutatorsContext) {
	ctx.BottomUp("container_generation", containerGenerationMutator).Parallel()
}

// Determines if the module can be installed in the system partition or not.
// Logic is identical to that of modulePartition(...) defined in paths.go
func installInSystemPartition(ctx BottomUpMutatorContext) bool {
	module := ctx.Module()
	return !module.InstallInTestcases() &&
		!module.InstallInData() &&
		!module.InstallInRamdisk() &&
		!module.InstallInVendorRamdisk() &&
		!module.InstallInDebugRamdisk() &&
		!module.InstallInRecovery() &&
		!module.InstallInVendor() &&
		!module.InstallInOdm() &&
		!module.InstallInProduct() &&
		determineModuleKind(module.base(), ctx.blueprintBaseModuleContext()) == platformModule
}

func containerGenerationMutator(ctx BottomUpMutatorContext) {
	if m, ok := ctx.Module().(InstallableModule); ok {
		base := m.Base()
		if installInSystemPartition(ctx) {
			base.SetSystemContainer(true)
		}
		if ctx.Module().InstallInProduct() {
			base.SetProductContainer(true)
		}
		if ctx.Module().InstallInVendor() {
			base.SetVendorContainer(true)
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
	}
}
