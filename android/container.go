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
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/google/blueprint"
)

type StubsAvailableModule interface {
	IsStubsModule() bool
}

var moduleIsStubsModule = func(m Module) bool {
	if stubsModule, ok := m.(StubsAvailableModule); ok {
		return stubsModule.IsStubsModule()
	}
	return false
}

type HidlStubsAvailableModule interface {
	IsHidlStubsModule() bool
}

var moduleIsHidlInterfaceStubsModule = func(m Module) bool {
	if hidlStubsAvailableModule, ok := m.(HidlStubsAvailableModule); ok {
		return hidlStubsAvailableModule.IsHidlStubsModule()
	}
	return false
}

type exceptionHandleFunc int

const (
	checkStubs exceptionHandleFunc = iota
	checkHidlInterface
	undefined
)

// Functions cannot be used as a value passed in providers, because functions are not
// hashable. As a workaround, the exceptionHandleFunc enum values are passed using providers,
// and the corresponding functions are called from this map.
var exceptionHandleFunctionsTable = map[exceptionHandleFunc]func(Module) bool{
	checkStubs:         moduleIsStubsModule,
	checkHidlInterface: moduleIsHidlInterfaceStubsModule,
	undefined:          func(Module) bool { return false },
}

type InstallableModule interface {
	EnforceApiContainerChecks() bool
}

type restriction struct {
	// container of the dependency
	dependency *container

	// Error message to be emitted to the user when the dependency meets this restriction
	errorMessage string

	// Optional exception function that allows bypassing this restriction.
	// If this function returns true, this dependency would be considered allowed and an error
	// will not be thrown.
	exceptionFunc exceptionHandleFunc
}
type container struct {
	// The name of the container i.e. partition, api domain
	name string

	// Map of dependency restricted containers.
	restricted []restriction
}

var (
	VendorContainer = &container{
		name:       VendorVariation,
		restricted: nil,
	}
	SystemContainer = &container{
		name: "system",
		restricted: []restriction{
			{
				dependency: VendorContainer,
				errorMessage: "Module belonging to the system partition other than HALs is " +
					"not allowed to depend on the vendor partition module, in order to support " +
					"independent development/update cycles and to support the Generic System " +
					"Image. Try depending on HALs, VNDK or AIDL instead.",
				exceptionFunc: checkHidlInterface,
			},
		},
	}
	ProductContainer = &container{
		name: ProductVariation,
		restricted: []restriction{
			{
				dependency: VendorContainer,
				errorMessage: "Module belonging to the product partition is not allowed to " +
					"depend on the vendor partition module, as this may lead to security " +
					"vulnerabilities. Try depending on the HALs or utilize AIDL instead.",
				exceptionFunc: checkHidlInterface,
			},
		},
	}
	ApexContainer = &container{
		name: "apex",
		restricted: []restriction{
			{
				dependency: SystemContainer,
				errorMessage: "Module belonging to Apex(es) is not allowed to depend on the " +
					"modules belonging to the system partition. Either statically depend on the " +
					"module or convert the depending module to java_sdk_library and depend on " +
					"the stubs.",
				exceptionFunc: checkStubs,
			},
		},
	}
	CtsContainer = &container{
		name: "cts",
		restricted: []restriction{
			{
				dependency: SystemContainer,
				errorMessage: "CTS module should not depend on the modules belonging to the " +
					"system partition, including \"framework\". Depending on the system " +
					"partition may lead to disclosure of implementation details and regression " +
					"due to API changes across platform versions. Try depending on the stubs instead.",
				exceptionFunc: checkStubs,
			},
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

func (c *ContainersInfo) GetViolations(dep Module, depInfo ContainersInfo) []string {
	var violations []string

	for _, depContainer := range depInfo.belongingContainers {
		for _, belongingContainer := range c.belongingContainers {
			for _, restriction := range belongingContainer.restricted {
				if depContainer == restriction.dependency {
					if exceptionHandleFunctionsTable[restriction.exceptionFunc](dep) {
						continue
					}
					violations = append(violations, restriction.errorMessage)
				}
			}
		}
	}
	return violations
}

var ContainersInfoProvider = blueprint.NewMutatorProvider[ContainersInfo]("container_generation")

func RegisterContainerMutator(ctx RegistrationContext) {
	ctx.FinalDepsMutators(registerContainerFinalDepsMutator)
}

func registerContainerFinalDepsMutator(ctx RegisterMutatorsContext) {
	ctx.BottomUp("container_generation", containerGenerationMutator).Parallel()
	ctx.BottomUp("container_enforcement", containerEnforcementMutator)
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

var visitedModuleNames map[string]ContainersInfo

func getContainerModuleInfo(ctx BottomUpMutatorContext, module Module) (info ContainersInfo, ok bool) {
	if visitedModuleNames == nil {
		visitedModuleNames = make(map[string]ContainersInfo)
	}

	if info, visited := visitedModuleNames[module.Name()]; visited {
		return info, visited
	} else {
		if containersInfo, ok := OtherModuleProvider(ctx, module, ContainersInfoProvider); ok {
			ctx.VisitAllModuleVariants(func(m Module) {
				variantContainersInfo, _ := OtherModuleProvider(ctx, m, ContainersInfoProvider)
				containersInfo.belongingContainers = append(containersInfo.belongingContainers, variantContainersInfo.belongingContainers...)
				containersInfo.apexNames = append(containersInfo.apexNames, variantContainersInfo.apexNames...)
			})
			containersInfo.belongingContainers = slices.Compact(containersInfo.belongingContainers)
			containersInfo.apexNames = slices.Compact(containersInfo.apexNames)

			visitedModuleNames[module.Name()] = containersInfo
			return containersInfo, ok
		}
	}
	return ContainersInfo{}, false
}

func containerEnforcementMutator(ctx BottomUpMutatorContext) {
	if containersInfo, ok := getContainerModuleInfo(ctx, ctx.Module()); ok {
		ctx.VisitDirectDepsIgnoreBlueprint(func(dep Module) {
			if depContainersInfo, ok := getContainerModuleInfo(ctx, dep); ok {
				violations := containersInfo.GetViolations(dep, depContainersInfo)
				if len(violations) > 0 {
					errorMessage := fmt.Sprintf("%s cannot depend on %s. ", ctx.ModuleName(), dep.Name())
					errorMessage += strings.Join(violations, " ")
					ctx.ModuleErrorf(errorMessage)
				}
			}
		})
	}
}
