// Copyright 2021 The Android Open Source Project
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This file defines a singleton that exports cc module alias as Make $(BUILD_PHONY_PACKAGE).
//
// With this non-core variants of a cc module can be specified as "${module name}.${partition}" in
// Make syntax. The main user of this feature is LOCAL_REQUIRED_MODULES and PRODUCT_PACKAGES.
// For example, if "foo" is a "recovery: true" module, then it would be exported to Make as "foo"
// and a phony package "foo.recovery" would be exported to Make as an alias to "foo".

package cc

import (
	"fmt"
	"io"

	"android/soong/android"
)

func init() {
	registerModuleAliasForRequiredSingleton(android.InitRegistrationContext)
}

func registerModuleAliasForRequiredSingleton(ctx android.RegistrationContext) {
	ctx.RegisterSingletonModuleType("cc_module_alias_for_required", moduleAliasForRequiredSingletonFactory)
}

type moduleReference struct {
	name string
	dir  string
}

type moduleAliasMap map[string]moduleReference

type moduleAliasForRequiredSingleton struct {
	android.SingletonModuleBase

	aliasMap moduleAliasMap
}

var moduleAliasForRequired *moduleAliasForRequiredSingleton

func moduleAliasForRequiredSingletonFactory() android.SingletonModule {
	moduleAliasForRequired = &moduleAliasForRequiredSingleton{}
	android.InitAndroidModule(moduleAliasForRequired)
	moduleAliasForRequired.aliasMap = make(moduleAliasMap)
	return moduleAliasForRequired
}

func (s *moduleAliasForRequiredSingleton) GenerateAndroidBuildActions(ctx android.ModuleContext) {
}

func (s *moduleAliasForRequiredSingleton) GenerateSingletonBuildActions(ctx android.SingletonContext) {
	ctx.VisitAllModules(func(m android.Module) {
		// Skip if module is not cc.Module.
		c, ok := m.(*Module)
		if !ok {
			return
		}
		// Skip if module is not for device.
		if !c.Device() {
			return
		}
		// Skip if module is not exported to Make.
		if s.isHiddenFromMake(m, c) {
			return
		}
		// Skip uninstallable stubs.
		if c.isNDKStubLibrary() || c.IsLlndk() || c.IsStubs() {
			return
		}
		// Skip native_bridge variants.
		if c.Target().NativeBridge == android.NativeBridgeEnabled {
			return
		}
		// Skip sdk variants.
		if c.IsSdkVariant() && c.Properties.SdkAndPlatformVariantVisibleToMake {
			return
		}
		// Skip uninstallable static and header libraries.
		if c.CcLibraryInterface() && !c.Shared() {
			return
		}
		// Skip VNDK libraries because they are installed by an APEX.
		if c.IsVndk() {
			return
		}
		// Skip native tests.
		if c.testBinary() {
			return
		}

		// Create alias if module doesn't have suffix and isn't core-variant.
		var subName string
		var aliasSuffix string

		if s, ok := c.linker.(SnapshotInterface); ok {
			subName = s.SnapshotAndroidMkSuffix()
		} else {
			subName = c.SubName()
		}
		if subName == "" {
			if c.InVendor() {
				aliasSuffix = VendorSuffix
			} else if c.InProduct() {
				aliasSuffix = productSuffix
			} else if c.InRecovery() {
				aliasSuffix = recoverySuffix
			} else if c.InRamdisk() {
				aliasSuffix = ramdiskSuffix
			} else if c.InVendorRamdisk() {
				aliasSuffix = VendorRamdiskSuffix
			}
		}
		if aliasSuffix != "" {
			aliasName := c.BaseModuleName() + aliasSuffix
			s.addAlias(ctx, aliasName, c)
		}
	})

	// Validation check. Make sure module Make names don't collide with any alias name.
	ctx.VisitAllModules(func(m android.Module) {
		// Skip if module is not cc.Module.
		c, ok := m.(*Module)
		if !ok {
			return
		}
		// Skip if module is not for device.
		if !c.Device() {
			return
		}
		// Skip if module is not exported to Make.
		if s.isHiddenFromMake(m, c) {
			return
		}

		var subName string

		if s, ok := c.linker.(SnapshotInterface); ok {
			subName = s.SnapshotAndroidMkSuffix()
		} else {
			subName = c.SubName()
		}
		aliasName := c.BaseModuleName() + subName
		if a, ok := s.aliasMap[aliasName]; ok {
			ctx.ModuleErrorf(m, "name conflict with module alias '%s' -> '%s:%s': %s", aliasName, a.dir, a.name, m)
		}
	})
}

func (s *moduleAliasForRequiredSingleton) isHiddenFromMake(m android.Module, c *Module) bool {
	return !m.Enabled() || !m.ExportedToMake() || m.IsHideFromMake() || c.HiddenFromMake() || c.hideApexVariantFromMake
}

func (s *moduleAliasForRequiredSingleton) addAlias(ctx android.SingletonContext, aliasName string, aliasModule *Module) {
	if _, ok := s.aliasMap[aliasName]; !ok {
		s.aliasMap[aliasName] = moduleReference{name: aliasModule.BaseModuleName(), dir: ctx.ModuleDir(aliasModule.Module())}
	}
}

func (s *moduleAliasForRequiredSingleton) AndroidMk() android.AndroidMkData {
	return android.AndroidMkData{
		Custom: func(w io.Writer, name, prefix, moduleDir string, data android.AndroidMkData) {
			for _, aliasName := range android.SortedStringKeys(s.aliasMap) {
				aliasModule := s.aliasMap[aliasName]
				fmt.Fprintln(w, "\ninclude $(CLEAR_VARS)")
				fmt.Fprintln(w, "LOCAL_PATH :=", aliasModule.dir)
				fmt.Fprintln(w, "LOCAL_MODULE :=", aliasName)
				fmt.Fprintln(w, "LOCAL_REQUIRED_MODULES :=", aliasModule.name)
				fmt.Fprintln(w, "include $(BUILD_PHONY_PACKAGE)")
			}
		},
	}
}
