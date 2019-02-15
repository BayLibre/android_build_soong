// Copyright 2019 Google Inc. All rights reserved.
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
	"sync"
	
	"github.com/google/blueprint"
)

// AtmModule is the interface that a module type is expected to implement if
// the module has to be built differently depending on whether the module
// is destined for an atm or not (installed to one of the regular partitions).
//
// Native shared libraries are one such module type; when it is built for an
// APEX, it should depend only on stable interfaces such as NDK, stable AIDL,
// or C APIs from other APEXs.
//
// A module implementing this interface will be mutated into multiple
// variations by atm.atmMutator if it is directly or indirectly included
// in one or more APEXs. Specifically, if a module is included in atm.foo and
// atm.bar then three atm variants are created: platform, atm.foo and
// atm.bar. The platform variant is for the regular partitions
// (e.g., /system or /vendor, etc.) while the other two are for the APEXs,
// respectively.
type AtmModule interface {
	Module
	atmModuleBase() *AtmModuleBase

	// Marks that this module should be built for the APEX of the specified name.
	// Call this before atm.atmMutator is run.
	BuildForAtm(atmName string)

	// Returns the name of APEX that this module will be built for. Empty string
	// is returned when 'IsForPlatform() == true'. Note that a module can be
	// included in multiple APEXes, in which case, the module is mutated into
	// multiple modules each of which for an APEX. This method returns the
	// name of the APEX that a variant module is for.
	// Call this after atm.atmMutator is run.
	AtmName() string

	// Tests whether this module will be built for the platform or not.
	// This is a shortcut for AtmName() == ""
	IsForPlatform() bool

	// Tests if this module could have APEX variants. APEX variants are
	// created only for the modules that returns true here. This is useful
	// for not creating APEX variants for certain types of shared libraries
	// such as NDK stubs.
	CanHaveAtmVariants() bool

	// Tests if this module can be installed to APEX as a file. For example,
	// this would return true for shared libs while return false for static
	// libs.
	IsInstallableToAtm() bool

	// Mutate this module into one or more variants each of which is built
	// for an APEX marked via BuildForAtm().
	CreateAtmVariations(mctx BottomUpMutatorContext) []blueprint.Module

	// Sets the name of the atm variant of this module. Called inside
	// CreateAtmVariations.
	setAtmName(atmName string)
}

type AtmProperties struct {
	// Name of the atm variant that this module is mutated into
	AtmName string `blueprint:"mutated"`
}

// Provides default implementation for the AtmModule interface. APEX-aware
// modules are expected to include this struct and call InitAtmModule().
type AtmModuleBase struct {
	AtmProperties AtmProperties

	canHaveAtmVariants bool
	atmVariations      []string
}

func (m *AtmModuleBase) atmModuleBase() *AtmModuleBase {
	return m
}

func (m *AtmModuleBase) BuildForAtm(atmName string) {
	if !InList(atmName, m.atmVariations) {
		m.atmVariations = append(m.atmVariations, atmName)
	}
}

func (m *AtmModuleBase) AtmName() string {
	return m.AtmProperties.AtmName
}

func (m *AtmModuleBase) IsForPlatform() bool {
	return m.AtmProperties.AtmName == ""
}

func (m *AtmModuleBase) setAtmName(atmName string) {
	m.AtmProperties.AtmName = atmName
}

func (m *AtmModuleBase) CanHaveAtmVariants() bool {
	return m.canHaveAtmVariants
}

func (m *AtmModuleBase) IsInstallableToAtm() bool {
	// should be overriden if needed
	return false
}

func (m *AtmModuleBase) CreateAtmVariations(mctx BottomUpMutatorContext) []blueprint.Module {
	if len(m.atmVariations) > 0 {
		variations := []string{""} // Original variation for platform
		variations = append(variations, m.atmVariations...)

		modules := mctx.CreateVariations(variations...)
		for i, m := range modules {
			if i == 0 {
				continue
			}
			fmt.Printf("atm_module: CreateAtmVariations setAtmName %s\n", variations[i])
			m.(AtmModule).setAtmName(variations[i])
		}
		return modules
	}
	return nil
}

var atmData OncePer
var atmNamesMapMutex sync.Mutex

// This structure maintains the global mapping in between modules and APEXes.
// Examples:
//
// atmNamesMap()["foo"]["bar"] == true: module foo is directly depended on by APEX bar
// atmNamesMap()["foo"]["bar"] == false: module foo is indirectly depended on by APEX bar
// atmNamesMap()["foo"]["bar"] doesn't exist: foo is not built for APEX bar
func atmNamesMap() map[string]map[string]bool {
	return atmData.Once("atmNames", func() interface{} {
		return make(map[string]map[string]bool)
	}).(map[string]map[string]bool)
}

// Update the map to mark that a module named moduleName is directly or indirectly
// depended on by an APEX named atmName. Directly depending means that a module
// is explicitly listed in the build definition of the APEX via properties like
// native_shared_libs, java_libs, etc.
func UpdateAtmDependency(atmName string, moduleName string, directDep bool) {
	atmNamesMapMutex.Lock()
	defer atmNamesMapMutex.Unlock()
	atmNames, ok := atmNamesMap()[moduleName]
	if !ok {
		atmNames = make(map[string]bool)
		atmNamesMap()[moduleName] = atmNames
	}
	atmNames[atmName] = atmNames[atmName] || directDep
}

// Tests whether a module named moduleName is directly depended on by an APEX
// named atmName.
func DirectlyInAtm(atmName string, moduleName string) bool {
	atmNamesMapMutex.Lock()
	defer atmNamesMapMutex.Unlock()
	if atmNames, ok := atmNamesMap()[moduleName]; ok {
		return atmNames[atmName]
	}
	return false
}

// Tests whether a module named moduleName is directly depended on by any APEX.
func DirectlyInAnyAtm(moduleName string) bool {
	atmNamesMapMutex.Lock()
	defer atmNamesMapMutex.Unlock()
	if atmNames, ok := atmNamesMap()[moduleName]; ok {
		for an := range atmNames {
			if atmNames[an] {
				return true
			}
		}
	}
	return false
}

// Tests whether a module named module is depended on (including both
// direct and indirect dependencies) by any APEX.
func InAnyAtm(moduleName string) bool {
	atmNamesMapMutex.Lock()
	defer atmNamesMapMutex.Unlock()
	atmNames, ok := atmNamesMap()[moduleName]
	return ok && len(atmNames) > 0
}

func GetAtmsForModule(moduleName string) []string {
	ret := []string{}
	atmNamesMapMutex.Lock()
	defer atmNamesMapMutex.Unlock()
	if atmNames, ok := atmNamesMap()[moduleName]; ok {
		for an := range atmNames {
			ret = append(ret, an)
		}
	}
	return ret
}

func InitAtmModule(m AtmModule) {
	base := m.atmModuleBase()
	base.canHaveAtmVariants = true

	m.AddProperties(&base.AtmProperties)
}
