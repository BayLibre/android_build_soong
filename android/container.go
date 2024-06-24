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
	EnforceApiContainerChecks() bool
}

type container struct {
	// The name of the container i.e. partition, api domain
	name string

	// Map of dependency restricted containers.
	// Keys are the containers that this container is not allowed to depend on,
	// and the values are reasons why they are not allowed to depend on.
	restricted map[*container]string
}

var (
	VendorContainer = &container{
		name:       VendorVariation,
		restricted: nil,
	}
	SystemContainer = &container{
		name: "system",
		restricted: map[*container]string{
			VendorContainer: "Module belonging to the system partition other than HALs is " +
				"not allowed to depend on the vendor partition module, in order to support " +
				"independent development/update cycles and to support the Generic System Image.",
		},
	}
	ProductContainer = &container{
		name: ProductVariation,
		restricted: map[*container]string{
			VendorContainer: "Module belonging to the product partition is not allowed to " +
				"depend on the vendor partition module, as this may lead to security " +
				"vulnerabilities. Try depending on the HALs or utilize AIDL instead.",
		},
	}
	ApexContainer = &container{
		name: "apex",
		restricted: map[*container]string{
			SystemContainer: "Module belonging to Apex(es) is not allowed to depend on the " +
				"modules belonging to the system partition. Either statically depend on the " +
				"module or convert the depending module to java_sdk_library and depend on " +
				"the stubs.",
		},
	}
	CtsContainer = &container{
		name: "cts",
		restricted: map[*container]string{
			SystemContainer: "CTS module should not depend on the modules belonging to the " +
				"system partition, including \"framework\". Depending on the system " +
				"partition may lead to disclosure of implementation details and regression " +
				"due to API changes across platform versions. Try depending on the stubs instead.",
		},
	}
)

type ContainersInfo struct {
	belongingContainers []*container

	apexNames []string
}

func (c *ContainersInfo) BelongingContainers() []*container {
	return c.belongingContainers
}

func (c *ContainersInfo) ApexNames() []string {
	return c.apexNames
}

var ContainersInfoProvider = blueprint.NewMutatorProvider[ContainersInfo]("container_generation")

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
		inSystem := installInSystemPartition(ctx)
		inProduct := ctx.Module().InstallInProduct()
		inVendor := ctx.Module().InstallInVendor()
		inCts := false

		if m, ok := ctx.Module().(ImageInterface); ok {
			inProduct = inProduct || m.ProductVariantNeeded(ctx)
			inVendor = inVendor || m.VendorVariantNeeded(ctx)
		}

		props := ctx.Module().GetProperties()
		for _, prop := range props {
			val := reflect.ValueOf(prop).Elem()
			if val.Kind() == reflect.Struct {
				testSuites := val.FieldByName("Test_suites")
				inCts = testSuites.IsValid() && testSuites.Kind() == reflect.Slice && slices.Contains(testSuites.Interface().([]string), "cts")
			}
		}

		var apexNames []string
		if apexInfo, ok := ModuleProvider(ctx, ApexInfoProvider); ok {
			apexNames = apexInfo.InApexModules
			if apexInfo.Updatable {
				inSystem = false
			}
		}

		containers := []*container{}
		if inSystem {
			containers = append(containers, SystemContainer)
		}
		if inProduct {
			containers = append(containers, ProductContainer)
		}
		if inVendor {
			containers = append(containers, VendorContainer)
		}
		if inCts {
			containers = append(containers, CtsContainer)
		}
		if len(apexNames) > 0 {
			containers = append(containers, ApexContainer)
		}

		SetProvider(ctx, ContainersInfoProvider, ContainersInfo{
			belongingContainers: containers,
			apexNames:           apexNames,
		})
	}
}
