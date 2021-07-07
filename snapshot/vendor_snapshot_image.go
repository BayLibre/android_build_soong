// Copyright 2020 The Android Open Source Project
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
package snapshot

import (
	"android/soong/android"
)

// This file defines snapshot prebuilt modules, e.g. vendor snapshot and recovery snapshot. Such
// snapshot modules will override original source modules with setting BOARD_VNDK_VERSION, with
// snapshot mutators and snapshot information maps which are also defined in this file.

type vendorSnapshotImage struct{}

func (vendorSnapshotImage) Init(ctx android.RegistrationContext) {
	ctx.RegisterSingletonType("vendor-snapshot", VendorSnapshotSingleton)
	ctx.RegisterSingletonType("vendor-fake-snapshot", VendorFakeSnapshotSingleton)
}

func (vendorSnapshotImage) RegisterAdditionalModule(ctx android.RegistrationContext, name string, factory android.ModuleFactory) {
	ctx.RegisterModuleType(name, factory)
}

func (vendorSnapshotImage) shouldGenerateSnapshot(ctx android.SingletonContext) bool {
	// BOARD_VNDK_VERSION must be set to 'current' in order to generate a snapshot.
	return ctx.DeviceConfig().VndkVersion() == "current"
}

func (vendorSnapshotImage) InImage(m SnapshotModuleInterface) func() bool {
	return m.InVendor
}

func (vendorSnapshotImage) Private(m SnapshotModuleInterface) bool {
	return m.IsPrivate()
}

func (vendorSnapshotImage) IsProprietaryPath(dir string, deviceConfig android.DeviceConfig) bool {
	return isDirectoryExcluded(dir, deviceConfig.VendorSnapshotDirsExcludedMap(), deviceConfig.VendorSnapshotDirsIncludedMap())
}

// vendor snapshot includes static/header libraries with vndk: {enabled: true}.
func (vendorSnapshotImage) IncludeVndk() bool {
	return true
}

func (vendorSnapshotImage) ExcludeFromSnapshot(m SnapshotModuleInterface) bool {
	return m.ExcludeFromVendorSnapshot()
}

func (vendorSnapshotImage) IsUsingSnapshot(cfg android.DeviceConfig) bool {
	vndkVersion := cfg.VndkVersion()
	return vndkVersion != "current" && vndkVersion != ""
}

func (vendorSnapshotImage) TargetSnapshotVersion(cfg android.DeviceConfig) string {
	return cfg.VndkVersion()
}

// returns true iff a given module SHOULD BE EXCLUDED, false if included
func (vendorSnapshotImage) ExcludeFromDirectedSnapshot(cfg android.DeviceConfig, name string) bool {
	// If we're using full snapshot, not directed snapshot, capture every module
	if !cfg.DirectedVendorSnapshot() {
		return false
	}
	// Else, checks if name is in VENDOR_SNAPSHOT_MODULES.
	return !cfg.VendorSnapshotModules()[name]
}

func (vendorSnapshotImage) ImageVariantName(cfg android.DeviceConfig) string {
	return "vendor." + cfg.VndkVersion()
}

func (vendorSnapshotImage) ModuleNameSuffix() string {
	return ".vendor"
}

var VendorSnapshotImageSingleton vendorSnapshotImage

func init() {
	VendorSnapshotImageSingleton.Init(android.InitRegistrationContext)
}
