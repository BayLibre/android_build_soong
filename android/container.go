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

	"github.com/google/blueprint"
)

type InstallableModule interface {
	IsInstallableModule() bool
}

type ContainersInfo struct {
	System_container  bool
	Product_container bool
	Vendor_container  bool
	Apex_containers   []string
	Cts_container     bool
}

var ContainersInfoProvider = blueprint.NewMutatorProvider[ContainersInfo]("container_generation")

var PrepareForTestWithContainer = FixtureRegisterWithContext(RegisterContainerMutator)

func RegisterContainerMutator(ctx RegistrationContext) {
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
	if _, ok := ctx.Module().(InstallableModule); ok {
		container := ContainersInfo{}
		container.System_container = installInSystemPartition(ctx)
		container.Product_container = ctx.Module().InstallInProduct()
		container.Vendor_container = ctx.Module().InstallInVendor()

		if m, ok := ctx.Module().(ImageInterface); ok {
			container.Product_container = container.Product_container || m.ProductVariantNeeded(ctx)
			container.Vendor_container = container.Vendor_container || m.VendorVariantNeeded(ctx)
		}

		props := ctx.Module().GetProperties()
		for _, prop := range props {
			val := reflect.ValueOf(prop).Elem()
			if val.Kind() == reflect.Struct {
				testSuites := val.FieldByName("Test_suites")
				if testSuites.IsValid() && testSuites.Kind() == reflect.Slice && slices.Contains(testSuites.Interface().([]string), "cts") {
					container.Cts_container = true
				}
			}
		}
		if apexInfo, ok := ModuleProvider(ctx, ApexInfoProvider); ok {
			container.Apex_containers = apexInfo.InApexModules
			if apexInfo.Updatable {
				container.System_container = false
			}
		}

		SetProvider(ctx, ContainersInfoProvider, container)
	}
}
