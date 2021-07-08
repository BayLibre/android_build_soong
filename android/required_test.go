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

package android

import (
	"io"
	"sort"
	"testing"

	"github.com/google/blueprint/proptools"
)

type requiredTestModule struct {
	ModuleBase

	properties struct {
		Recovery_available *bool
	}

	data    AndroidMkData
	subName string
}

func requiredTestModuleFactory() Module {
	m := &requiredTestModule{}
	m.AddProperties(&m.properties)
	InitAndroidArchModule(m, DeviceSupported, MultilibFirst)
	return m
}

func (m *requiredTestModule) SubName() string {
	return m.subName
}

func (m *requiredTestModule) GenerateAndroidBuildActions(ctx ModuleContext) {
	if m.ModuleBase.InRecovery() {
		m.subName = ".recovery"
	} else if m.ModuleBase.InRamdisk() {
		m.subName = ".ramdisk"
	} else if m.ModuleBase.InVendorRamdisk() {
		m.subName = ".vendor_ramdisk"
	} else if m.ModuleBase.InDebugRamdisk() {
		m.subName = ".debug_ramdisk"
	}
}

var _ ImageInterface = (*requiredTestModule)(nil)

func (m *requiredTestModule) ImageMutatorBegin(ctx BaseModuleContext) {}

func (m *requiredTestModule) CoreVariantNeeded(ctx BaseModuleContext) bool {
	return !m.ModuleBase.InstallInRecovery() && !m.ModuleBase.InstallInRamdisk() &&
		!m.ModuleBase.InstallInVendorRamdisk() && !m.ModuleBase.InstallInDebugRamdisk()
}

func (m *requiredTestModule) RecoveryVariantNeeded(ctx BaseModuleContext) bool {
	return proptools.Bool(m.properties.Recovery_available) || m.ModuleBase.InstallInRecovery()
}

func (m *requiredTestModule) RamdiskVariantNeeded(ctx BaseModuleContext) bool {
	return m.ModuleBase.InstallInRamdisk()
}

func (m *requiredTestModule) VendorRamdiskVariantNeeded(ctx BaseModuleContext) bool {
	return m.ModuleBase.InstallInVendorRamdisk()
}

func (m *requiredTestModule) DebugRamdiskVariantNeeded(ctx BaseModuleContext) bool {
	return m.ModuleBase.InstallInDebugRamdisk()
}

func (m *requiredTestModule) ExtraImageVariations(ctx BaseModuleContext) []string {
	return nil
}

func (m *requiredTestModule) SetImageVariation(ctx BaseModuleContext, variation string, module Module) {
}

var _ AndroidMkNamesInterface = (*requiredTestModule)(nil)

func (m *requiredTestModule) AndroidMkNames() []string {
	return []string{m.BaseModuleName() + m.SubName()}
}

var _ AndroidMkDataProvider = (*requiredTestModule)(nil)

// Spy on the AndroidMkData filled by AndroidMkSingleton.
func (m *requiredTestModule) AndroidMk() AndroidMkData {
	return AndroidMkData{
		Custom: func(w io.Writer, name, prefix, moduleDir string, data AndroidMkData) {
			m.data = data
		},
	}
}

var prepareForRequiredTest = GroupFixturePreparers(
	PrepareForTestWithArchMutator,
	PrepareForTestWithAndroidMk,
	FixtureRegisterWithContext(func(ctx RegistrationContext) {
		ctx.RegisterModuleType("test_module", requiredTestModuleFactory)
	}),
)

func TestRequiredModuleSingleton(t *testing.T) {
	bp := `
		test_module {
			name: "foo",
			required: [
				"bar{.}",
				"bar{.recovery}",
				"baz{.ramdisk}",
				"baz{.recovery}",
			],
		}

		test_module {
			name: "bar",
			recovery_available: true,
		}

		test_module {
			name: "baz",
			ramdisk: true,
			recovery_available: true,
		}
	`
	result := GroupFixturePreparers(prepareForRequiredTest).RunTestWithBp(t, bp)

	config := result.Config
	foo := result.ModuleForTests("foo", config.AndroidFirstDeviceTarget.String()).Module().(*requiredTestModule)

	expectedRequired := []string{"bar", "bar.recovery", "baz.ramdisk", "baz.recovery"}
	fooRequired := CopyOf(foo.data.Required)
	sort.Strings(fooRequired)
	AssertArrayString(t, "foo required", expectedRequired, fooRequired)
}
