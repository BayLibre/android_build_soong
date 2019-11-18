// Copyright 2018 Google Inc. All rights reserved.
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
	"sort"
	"sync"

	"github.com/google/blueprint"
)

// ApexRef is a reference of an apex bundle with minial information
type ApexRef struct {
	Name        string
	Installable bool
	Test        bool
}

// this is for pseudo apex "//apex_available:platform"
var pseudoPlatformApex = ApexRef{availableToPlatform, false, false}

// ApexModule is the interface that a module type is expected to implement if
// the module has to be built differently depending on whether the module
// is destined for an apex or not (installed to one of the regular partitions).
//
// Native shared libraries are one such module type; when it is built for an
// APEX, it should depend only on stable interfaces such as NDK, stable AIDL,
// or C APIs from other APEXs.
//
// A module implementing this interface will be mutated into multiple
// variations by apex.apexMutator if it is directly or indirectly included
// in one or more APEXs. Specifically, if a module is included in apex.foo and
// apex.bar then three apex variants are created: platform, apex.foo and
// apex.bar. The platform variant is for the regular partitions
// (e.g., /system or /vendor, etc.) while the other two are for the APEXs,
// respectively.
type ApexModule interface {
	Module
	apexModuleBase() *ApexModuleBase

	// Returns the name of APEX that this module will be built for. Empty string
	// is returned when 'IsForPlatform() == true'. Note that a module can be
	// included in multiple APEXes, in which case, the module is mutated into
	// multiple modules each of which for an APEX. This method returns the
	// name of the APEX that a variant module is for.
	// Call this after apex.apexMutator is run.
	ApexName() string

	// Tests whether this module will be built for the platform or not.
	// This is a shortcut for ApexName() == ""
	IsForPlatform() bool

	// Tests if this module could have APEX variants. APEX variants are
	// created only for the modules that returns true here. This is useful
	// for not creating APEX variants for certain types of shared libraries
	// such as NDK stubs.
	CanHaveApexVariants() bool

	// Tests if this module can be installed to APEX as a file. For example,
	// this would return true for shared libs while return false for static
	// libs.
	IsInstallableToApex() bool

	// Mutate this module into one or more variants each of which is built
	// for an APEX marked via AddApexDep().
	CreateApexVariations(mctx BottomUpMutatorContext) []blueprint.Module

	// Sets the name of the apex variant of this module. Called inside
	// CreateApexVariations.
	setApexName(apexName string)

	// Tests if this module is available for the specified APEX or ":platform"
	AvailableFor(what string) bool

	// DepIsInSameApex tests if the other module 'dep' is installed to the same
	// APEX as this module
	DepIsInSameApex(ctx BaseModuleContext, dep Module) bool

	// Ensure apex_available is well-formed
	checkApexAvailableProperty(mctx BaseModuleContext)

	// Mark this module as it is used by given apex
	addApex(apex ApexRef, checkAvailability bool)

	// Propagates apex deps to its children
	PushDownApexDeps(mctx TopDownMutatorContext)

	// Returns true if the depedency to `child` requires a check against "apex_available".
	// Override this if "available" means different than simple dependency.
	// For example, "available" in `cc_*` modules means "able to include symbols".
	NeedToCheckApexAvailableForChild(mctx BaseModuleContext, child ApexModule) bool
}

type ApexProperties struct {
	// Availability of this module in APEXes. Only the listed APEXes can include this module.
	// "//apex_available:anyapex" is a pseudo APEX name that matches to any APEX.
	// "//apex_available:platform" refers to non-APEX partitions like "system.img".
	// Default is ["//apex_available:platform", "//apex_available:anyapex"].
	// TODO(b/128708192) change the default to ["//apex_available:platform"]
	// For cc_* modules, including a module means including symbols.
	Apex_available []string

	// Name of the apex variant that this module is mutated into
	ApexName string `blueprint:"mutated"`
}

// Provides default implementation for the ApexModule interface. APEX-aware
// modules are expected to include this struct and call InitApexModule().
type ApexModuleBase struct {
	ApexProperties ApexProperties

	canHaveApexVariants bool

	// key: apex which depends on this module.
	// value: tells the dependency needs to be checked against apex_available
	apexDeps     map[ApexRef]bool
	apexDepsLock sync.Mutex // protects apexDeps during parallel apexDepsMutator
}

func (m *ApexModuleBase) apexModuleBase() *ApexModuleBase {
	return m
}

func (m *ApexModuleBase) addApex(apex ApexRef, checkAvailable bool) {
	m.apexDepsLock.Lock()
	defer m.apexDepsLock.Unlock()
	m.apexDeps[apex] = m.apexDeps[apex] || checkAvailable
}

// AddApexDep adds direct/indirect depedency from `apex` to `m`.
// Before adding, "apex_available" is checked against
func AddApexDep(mctx TopDownMutatorContext, m Module, apex ApexRef, checkAvailable bool) {
	if apex.Installable && !apex.Test {
		// TODO(b/123892969): Workaround for not having any way to annotate test-apexs
		// non-installable apex's cannot be installed and so should not prevent libraries from being
		// installed to the system.
		UpdateApexDependency(apex.Name, mctx.OtherModuleName(m), apex.Name == mctx.ModuleName())
	}

	if am, ok := m.(ApexModule); ok && am.CanHaveApexVariants() {
		am.checkApexAvailableProperty(mctx)
		if !apex.Test && checkAvailable {
			if !am.AvailableFor(apex.Name) {
				mctx.ModuleErrorf("requires %q that is not available for the APEX %q",
					mctx.OtherModuleName(m), apex.Name)
			}
		}
		am.addApex(apex, checkAvailable)
	}
}

// PushDownApexDeps propagates apex deps to its children
func (m *ApexModuleBase) PushDownApexDeps(mctx TopDownMutatorContext) {
	parent := mctx.Module().(ApexModule)
	mctx.VisitDirectDeps(func(child Module) {
		if am, ok := child.(ApexModule); ok && am.CanHaveApexVariants() {
			checkChild := parent.NeedToCheckApexAvailableForChild(mctx, am)
			if parent.AvailableFor(availableToPlatform) {
				AddApexDep(mctx, child, pseudoPlatformApex, checkChild)
			}
			for apex, check := range m.apexDeps {
				AddApexDep(mctx, child, apex, checkChild && check)
			}
		}
	})
}

// NeedToCheckApexAvailableForChild returns true if the depedency to `child` requires a check against "apex_available".
// Override this if "available" means different than simple dependency.
// For example, "available" in `cc_*` modules means "able to include symbols".
func (m *ApexModuleBase) NeedToCheckApexAvailableForChild(mctx BaseModuleContext, child ApexModule) bool {
	return true
}

func (m *ApexModuleBase) ApexName() string {
	return m.ApexProperties.ApexName
}

func (m *ApexModuleBase) IsForPlatform() bool {
	return m.ApexProperties.ApexName == ""
}

func (m *ApexModuleBase) setApexName(apexName string) {
	m.ApexProperties.ApexName = apexName
}

func (m *ApexModuleBase) CanHaveApexVariants() bool {
	return m.canHaveApexVariants
}

func (m *ApexModuleBase) IsInstallableToApex() bool {
	// should be overriden if needed
	return false
}

const (
	availableToPlatform = "//apex_available:platform"
	availableToAnyApex  = "//apex_available:anyapex"
)

func CheckAvailableForApex(what string, apex_available []string) bool {
	if len(apex_available) == 0 {
		// apex_available defaults to ["//apex_available:platform", "//apex_available:anyapex"],
		// which means 'available to everybody'.
		return true
	}
	return InList(what, apex_available) ||
		(what != availableToPlatform && InList(availableToAnyApex, apex_available))
}

func (m *ApexModuleBase) AvailableFor(what string) bool {
	return CheckAvailableForApex(what, m.ApexProperties.Apex_available)
}

func (m *ApexModuleBase) DepIsInSameApex(ctx BaseModuleContext, dep Module) bool {
	// By default, if there is a dependency from A to B, we try to include both in the same APEX,
	// unless B is explicitly from outside of the APEX (i.e. a stubs lib). Thus, returning true.
	// This is overridden by some module types like apex.ApexBundle, cc.Module, java.Module, etc.
	return true
}

func (m *ApexModuleBase) checkApexAvailableProperty(mctx BaseModuleContext) {
	for _, n := range m.ApexProperties.Apex_available {
		if n == availableToPlatform || n == availableToAnyApex {
			continue
		}
		if !mctx.OtherModuleExists(n) && !mctx.Config().AllowMissingDependencies() {
			mctx.PropertyErrorf("apex_available", "%q is not a valid module name", n)
		}
	}
}

func (m *ApexModuleBase) CreateApexVariations(mctx BottomUpMutatorContext) []blueprint.Module {
	variations := []string{}
	for apex := range m.apexDeps {
		if apex != pseudoPlatformApex {
			variations = append(variations, apex.Name)
		}
	}
	if len(variations) > 0 {
		availableForPlatform := mctx.Module().(ApexModule).AvailableFor(availableToPlatform)
		if availableForPlatform {
			variations = append(variations, "") // Original variation for platform
		}
		sort.Strings(variations)

		modules := mctx.CreateVariations(variations...)
		for i, m := range modules {
			m.(ApexModule).setApexName(variations[i])
		}
		return modules
	}
	return nil
}

var apexData OncePer
var apexNamesMapMutex sync.Mutex
var apexNamesKey = NewOnceKey("apexNames")

// This structure maintains the global mapping in between modules and APEXes.
// Examples:
//
// apexNamesMap()["foo"]["bar"] == true: module foo is directly depended on by APEX bar
// apexNamesMap()["foo"]["bar"] == false: module foo is indirectly depended on by APEX bar
// apexNamesMap()["foo"]["bar"] doesn't exist: foo is not built for APEX bar
func apexNamesMap() map[string]map[string]bool {
	return apexData.Once(apexNamesKey, func() interface{} {
		return make(map[string]map[string]bool)
	}).(map[string]map[string]bool)
}

// Update the map to mark that a module named moduleName is directly or indirectly
// depended on by an APEX named apexName. Directly depending means that a module
// is explicitly listed in the build definition of the APEX via properties like
// native_shared_libs, java_libs, etc.
func UpdateApexDependency(apexName string, moduleName string, directDep bool) {
	apexNamesMapMutex.Lock()
	defer apexNamesMapMutex.Unlock()
	apexNames, ok := apexNamesMap()[moduleName]
	if !ok {
		apexNames = make(map[string]bool)
		apexNamesMap()[moduleName] = apexNames
	}
	apexNames[apexName] = apexNames[apexName] || directDep
}

// Tests whether a module named moduleName is directly depended on by an APEX
// named apexName.
func DirectlyInApex(apexName string, moduleName string) bool {
	apexNamesMapMutex.Lock()
	defer apexNamesMapMutex.Unlock()
	if apexNames, ok := apexNamesMap()[moduleName]; ok {
		return apexNames[apexName]
	}
	return false
}

type hostContext interface {
	Host() bool
}

// Tests whether a module named moduleName is directly depended on by any APEX.
func DirectlyInAnyApex(ctx hostContext, moduleName string) bool {
	if ctx.Host() {
		// Host has no APEX.
		return false
	}
	apexNamesMapMutex.Lock()
	defer apexNamesMapMutex.Unlock()
	if apexNames, ok := apexNamesMap()[moduleName]; ok {
		for an := range apexNames {
			if apexNames[an] {
				return true
			}
		}
	}
	return false
}

// Tests whether a module named module is depended on (including both
// direct and indirect dependencies) by any APEX.
func InAnyApex(moduleName string) bool {
	apexNamesMapMutex.Lock()
	defer apexNamesMapMutex.Unlock()
	apexNames, ok := apexNamesMap()[moduleName]
	return ok && len(apexNames) > 0
}

func GetApexesForModule(moduleName string) []string {
	ret := []string{}
	apexNamesMapMutex.Lock()
	defer apexNamesMapMutex.Unlock()
	if apexNames, ok := apexNamesMap()[moduleName]; ok {
		for an := range apexNames {
			ret = append(ret, an)
		}
	}
	return ret
}

func InitApexModule(m ApexModule) {
	base := m.apexModuleBase()
	base.canHaveApexVariants = true
	base.apexDeps = make(map[ApexRef]bool)
	m.AddProperties(&base.ApexProperties)
}
