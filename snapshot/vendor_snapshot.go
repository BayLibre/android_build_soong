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
	if c, ok := ctx.Module().(SnapshotModuleInterface); ok {
		if c.ExcludeFromVendorSnapshot() {
			return true
		}
	}

	return false
}
