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
package snapshot

import "android/soong/android"

var vendorSnapshotSingleton = SnapshotSingleton{
	"vendor",
	"SOONG_VENDOR_SNAPSHOT_ZIP",
	android.OptionalPath{},
	true,
	VendorSnapshotImageSingleton,
	false, /* fake */
}

var vendorFakeSnapshotSingleton = SnapshotSingleton{
	"vendor",
	"SOONG_VENDOR_FAKE_SNAPSHOT_ZIP",
	android.OptionalPath{},
	true,
	VendorSnapshotImageSingleton,
	true, /* fake */
}

func VendorSnapshotSingleton() android.Singleton {
	return &vendorSnapshotSingleton
}

func VendorFakeSnapshotSingleton() android.Singleton {
	return &vendorFakeSnapshotSingleton
}

// Determine if a dir under source tree is an SoC-owned proprietary directory based
// on vendor snapshot configuration
// Examples: device/, vendor/
func isVendorProprietaryPath(dir string, deviceConfig android.DeviceConfig) bool {
	return VendorSnapshotSingleton().(*SnapshotSingleton).Image.IsProprietaryPath(dir, deviceConfig)
}

func IsVendorProprietaryModule(ctx android.BaseModuleContext) bool {
	// Any module in a vendor proprietary path is a vendor proprietary
	// module.
	if isVendorProprietaryPath(ctx.ModuleDir(), ctx.DeviceConfig()) {
		return true
	}

	// However if the module is not in a vendor proprietary path, it may
	// still be a vendor proprietary module. This happens for cc modules
	// that are excluded from the vendor snapshot, and it means that the
	// vendor has assumed control of the framework-provided module.
	if c, ok := ctx.Module().(Snapshotable); ok {
		if c.ExcludeFromVendorSnapshot() {
			return true
		}
	}

	return false
}

var VendorSnapshotImageName = "vendor"

type VendorSnapshotImage struct{}

func (VendorSnapshotImage) Init(ctx android.RegistrationContext) {
	ctx.RegisterSingletonType("vendor-snapshot", VendorSnapshotSingleton)
	ctx.RegisterSingletonType("vendor-fake-snapshot", VendorFakeSnapshotSingleton)
}

func (VendorSnapshotImage) RegisterAdditionalModule(ctx android.RegistrationContext, name string, factory android.ModuleFactory) {
	ctx.RegisterModuleType(name, factory)
}

func (VendorSnapshotImage) shouldGenerateSnapshot(ctx android.SingletonContext) bool {
	// BOARD_VNDK_VERSION must be set to 'current' in order to generate a snapshot.
	return ctx.DeviceConfig().VndkVersion() == "current"
}

func (VendorSnapshotImage) InImage(m Snapshotable) func() bool {
	return m.InVendor
}

func (VendorSnapshotImage) IsProprietaryPath(dir string, deviceConfig android.DeviceConfig) bool {
	return isDirectoryExcluded(dir, deviceConfig.VendorSnapshotDirsExcludedMap(), deviceConfig.VendorSnapshotDirsIncludedMap())
}

func (VendorSnapshotImage) ExcludeFromSnapshot(m Snapshotable) bool {
	return m.ExcludeFromVendorSnapshot()
}

func (VendorSnapshotImage) IsUsingSnapshot(cfg android.DeviceConfig) bool {
	vndkVersion := cfg.VndkVersion()
	return vndkVersion != "current" && vndkVersion != ""
}

func (VendorSnapshotImage) TargetSnapshotVersion(cfg android.DeviceConfig) string {
	return cfg.VndkVersion()
}

// returns true iff a given module SHOULD BE EXCLUDED, false if included
func (VendorSnapshotImage) ExcludeFromDirectedSnapshot(cfg android.DeviceConfig, name string) bool {
	// If we're using full snapshot, not directed snapshot, capture every module
	if !cfg.DirectedVendorSnapshot() {
		return false
	}
	// Else, checks if name is in VENDOR_SNAPSHOT_MODULES.
	return !cfg.VendorSnapshotModules()[name]
}

func (VendorSnapshotImage) ImageName() string {
	return VendorSnapshotImageName
}

var VendorSnapshotImageSingleton VendorSnapshotImage

func init() {
	VendorSnapshotImageSingleton.Init(android.InitRegistrationContext)
}
