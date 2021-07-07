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

import (
	"path/filepath"
	"sort"

	"android/soong/android"
)

// This file contains singletons to capture vendor and recovery snapshot. They consist of prebuilt
// modules under AOSP so older vendor and recovery can be built with a newer system in a single
// source tree.

var pctx = android.NewPackageContext("android/soong/snapshot")

type SnapshotModuleInterface interface {
	InRecovery() bool
	InVendor() bool
	IsPrivate() bool
	ExcludeFromVendorSnapshot() bool
	ExcludeFromRecoverySnapshot() bool
}

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

var recoverySnapshotSingleton = SnapshotSingleton{
	"recovery",
	"SOONG_RECOVERY_SNAPSHOT_ZIP",
	android.OptionalPath{},
	false,
	RecoverySnapshotImageSingleton,
	false, /* fake */
}

func VendorSnapshotSingleton() android.Singleton {
	return &vendorSnapshotSingleton
}

func VendorFakeSnapshotSingleton() android.Singleton {
	return &vendorFakeSnapshotSingleton
}

func RecoverySnapshotSingleton() android.Singleton {
	return &recoverySnapshotSingleton
}

type SnapshotSingleton struct {
	// Name, e.g., "vendor", "recovery", "ramdisk".
	name string

	// Make variable that points to the snapshot file, e.g.,
	// "SOONG_RECOVERY_SNAPSHOT_ZIP".
	makeVar string

	// Path to the snapshot zip file.
	snapshotZipFile android.OptionalPath

	// Whether the image supports VNDK extension modules.
	SupportsVndkExt bool

	// Implementation of the image interface specific to the image
	// associated with this snapshot (e.g., specific to the vendor image,
	// recovery image, etc.).
	Image SnapshotImage

	// Whether this singleton is for fake snapshot or not.
	// Fake snapshot is a snapshot whose prebuilt binaries and headers are empty.
	// It is much faster to generate, and can be used to inspect dependencies.
	Fake bool
}

// Determine if a dir under source tree is an SoC-owned proprietary directory based
// on vendor snapshot configuration
// Examples: device/, vendor/
func isVendorProprietaryPath(dir string, deviceConfig android.DeviceConfig) bool {
	return VendorSnapshotSingleton().(*SnapshotSingleton).Image.IsProprietaryPath(dir, deviceConfig)
}

// Determine if a dir under source tree is an SoC-owned proprietary directory based
// on recovery snapshot configuration
// Examples: device/, vendor/
func isRecoveryProprietaryPath(dir string, deviceConfig android.DeviceConfig) bool {
	return RecoverySnapshotSingleton().(*SnapshotSingleton).Image.IsProprietaryPath(dir, deviceConfig)
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

func IsRecoveryProprietaryModule(ctx android.BaseModuleContext) bool {

	// Any module in a recovery proprietary path is a recovery proprietary
	// module.
	if isRecoveryProprietaryPath(ctx.ModuleDir(), ctx.DeviceConfig()) {
		return true
	}

	// However if the module is not in a recovery proprietary path, it may
	// still be a recovery proprietary module. This happens for cc modules
	// that are excluded from the recovery snapshot, and it means that the
	// vendor has assumed control of the framework-provided module.

	if c, ok := ctx.Module().(SnapshotModuleInterface); ok {
		if c.ExcludeFromRecoverySnapshot() {
			return true
		}
	}

	return false
}

// Interface of function to capture snapshot from each module
type GenerateSnapshotAction func(snapshot SnapshotSingleton, ctx android.SingletonContext, snapshotArchDir string) android.Paths

var snapshotActionList []GenerateSnapshotAction

// Register GenerateSnapshotAction function so it can be called while generating snapshot
func RegisterSnapshotAction(x GenerateSnapshotAction) {
	snapshotActionList = append(snapshotActionList, x)
}

func (c *SnapshotSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	if !c.Image.shouldGenerateSnapshot(ctx) {
		return
	}

	var snapshotOutputs android.Paths

	// Snapshot zipped artifacts will be captured under {SNAPSHOT_ARCH} directory

	snapshotDir := c.name + "-snapshot"
	if c.Fake {
		// If this is a fake snapshot singleton, place all files under fake/ subdirectory to avoid
		// collision with real snapshot files
		snapshotDir = filepath.Join("fake", snapshotDir)
	}
	snapshotArchDir := filepath.Join(snapshotDir, ctx.DeviceConfig().DeviceArch())

	for _, f := range snapshotActionList {
		snapshotOutputs = append(snapshotOutputs, f(*c, ctx, snapshotArchDir)...)
	}

	// All artifacts are ready. Sort them to normalize ninja and then zip.
	sort.Slice(snapshotOutputs, func(i, j int) bool {
		return snapshotOutputs[i].String() < snapshotOutputs[j].String()
	})

	zipPath := android.PathForOutput(
		ctx,
		snapshotDir,
		c.name+"-"+ctx.Config().DeviceName()+".zip")
	zipRule := android.NewRuleBuilder(pctx, ctx)

	// filenames in rspfile from FlagWithRspFileInputList might be single-quoted. Remove it with tr
	snapshotOutputList := android.PathForOutput(
		ctx,
		snapshotDir,
		c.name+"-"+ctx.Config().DeviceName()+"_list")
	rspFile := snapshotOutputList.ReplaceExtension(ctx, "rsp")
	zipRule.Command().
		Text("tr").
		FlagWithArg("-d ", "\\'").
		FlagWithRspFileInputList("< ", rspFile, snapshotOutputs).
		FlagWithOutput("> ", snapshotOutputList)

	zipRule.Temporary(snapshotOutputList)

	zipRule.Command().
		BuiltTool("soong_zip").
		FlagWithOutput("-o ", zipPath).
		FlagWithArg("-C ", android.PathForOutput(ctx, snapshotDir).String()).
		FlagWithInput("-l ", snapshotOutputList)

	zipRule.Build(zipPath.String(), c.name+" snapshot "+zipPath.String())
	zipRule.DeleteTemporaryFiles()
	c.snapshotZipFile = android.OptionalPathForPath(zipPath)
}

func (c *SnapshotSingleton) MakeVars(ctx android.MakeVarsContext) {
	ctx.Strict(
		c.makeVar,
		c.snapshotZipFile.String())
}
